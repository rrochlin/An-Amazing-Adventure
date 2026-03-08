// ws-chat handles the WebSocket "chat" route.
//
// Turn flow:
//  0. RBAC check     — verify AI access is enabled and token quota not exceeded
//  1. NarrateStream  — streams pure narrative prose (no tools) to the client
//  2. narrative_end  — signals streaming is complete
//  3. EngineerScan   — infers world mutations from the narrative, executes them
//  4. PutMutation    — persists audit log entries (best-effort)
//  5. PutGame        — persists updated game state + chat history
//  6. UpdateTokens   — increments per-user token counter (best-effort)
//  7. SendDelta      — sends state delta (player, room, world events)
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

type chatRequest struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connID := req.RequestContext.ConnectionID

	// Parse message
	var msg chatRequest
	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		log.Printf("ws-chat: bad body: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}
	if msg.Content == "" {
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		log.Printf("ws-chat: db init: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Load connection record
	conn, err := dbClient.GetConnection(ctx, connID)
	if err != nil {
		log.Printf("ws-chat: get connection: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 410}, nil // Gone
	}

	// Block concurrent chats
	if conn.Streaming {
		ws, _ := wsutil.New(ctx)
		_ = ws.Send(ctx, connID, wsutil.Frame{Type: wsutil.FrameStreamingBlocked})
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	// Mark streaming = true
	if err := dbClient.SetStreaming(ctx, connID, true); err != nil {
		log.Printf("ws-chat: set streaming: %v", err)
	}
	defer func() {
		_ = dbClient.SetStreaming(ctx, connID, false)
	}()

	// Set up WebSocket sender early so we can send error frames during RBAC check
	ws, err := wsutil.New(ctx)
	if err != nil {
		log.Printf("ws-chat: ws sender init: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// RBAC + quota check — must pass before loading game or calling Bedrock
	userID := string(conn.UserID)
	userRecord, err := dbClient.GetUser(ctx, userID)
	if err != nil {
		log.Printf("ws-chat: GetUser error (treating as restricted): %v", err)
	}
	if userRecord == nil || !userRecord.AIEnabled {
		_ = ws.SendError(ctx, connID, "ai_access_not_enabled")
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}
	if userRecord.TokenLimit > 0 && userRecord.TokensUsed >= userRecord.TokenLimit {
		_ = ws.SendError(ctx, connID, "quota_exceeded")
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	// Load game state
	saveState, err := dbClient.GetGame(ctx, conn.GameID)
	if err != nil {
		log.Printf("ws-chat: get game %s: %v", conn.GameID, err)
		return events.APIGatewayProxyResponse{StatusCode: 404}, nil
	}

	g, err := game.FromSaveState(saveState)
	if err != nil {
		log.Printf("ws-chat: load game: %v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Set up Bedrock client
	aiClient, err := ai.New(ctx)
	if err != nil {
		log.Printf("ws-chat: ai init: %v", err)
		_ = ws.SendError(ctx, connID, "AI unavailable")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Capture pre-turn state for delta calculation
	preTurnPlayerLoc := g.Player.LocationID

	// Step 1: Stream narrator prose — no tools, pure narrative.
	narratorResult, err := aiClient.NarrateStream(
		ctx, g, saveState.Narrative, msg.Content,
		func(chunk string) {
			if sendErr := ws.SendNarrativeChunk(ctx, connID, chunk); sendErr != nil {
				log.Printf("ws-chat: send chunk: %v", sendErr)
			}
		},
	)
	if err != nil {
		log.Printf("ws-chat: narrator error: %v", err)
		_ = ws.SendError(ctx, connID, "Narrator error — please try again")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Step 2: Signal streaming complete (client commits the narrative bubble).
	_ = ws.SendNarrativeEnd(ctx, connID)

	// Step 3: Engineer infers world mutations from the narrative and executes them.
	// Runs after narrative_end so the client never waits on the Engineer for prose.
	engineerResult, err := aiClient.EngineerScan(ctx, g, narratorResult.Narrative)
	if err != nil {
		// Non-fatal: log the error but continue — game state may be partially mutated,
		// but the narrative has already been delivered successfully.
		log.Printf("ws-chat: engineer scan error: %v", err)
	}

	// Step 4: Persist mutation audit log entries (best-effort — failure is non-fatal).
	for _, m := range engineerResult.Mutations {
		if err := dbClient.PutMutation(ctx, m); err != nil {
			log.Printf("ws-chat: put mutation (tool=%s): %v", m.Tool, err)
		}
	}

	// Append chat history — attach world events to the narrative message so they
	// survive reconnection/reload.
	history := saveState.ChatHistory
	history = append(history, game.ChatMessage{Type: "player", Content: msg.Content})
	history = append(history, game.ChatMessage{
		Type:    "narrative",
		Content: narratorResult.Narrative,
		Events:  engineerResult.Events,
	})

	// Update stats (include both Narrator and Engineer token usage).
	g.ConversationCount++
	g.TotalTokens += narratorResult.Tokens.Total() + engineerResult.Tokens.Total()

	// Step 5: Persist updated game state with optimistic locking retry.
	g.Version++
	saved := g.ToSaveState(narratorResult.NewMessages, history)
	for attempt := 0; attempt < 3; attempt++ {
		if err := dbClient.PutGame(ctx, saved); err != nil {
			log.Printf("ws-chat: put game attempt %d: %v", attempt+1, err)
			if attempt == 2 {
				_ = ws.SendError(ctx, connID, "Failed to save game state")
				return events.APIGatewayProxyResponse{StatusCode: 500}, nil
			}
			// Reload and re-apply on conflict
			fresh, loadErr := dbClient.GetGame(ctx, conn.GameID)
			if loadErr == nil {
				saved.Version = fresh.Version + 1
			}
			continue
		}
		break
	}

	// Step 6: Account for token usage — best-effort, non-fatal.
	totalTokenDelta := narratorResult.Tokens.Total() + engineerResult.Tokens.Total()
	if accountErr := dbClient.UpdateUserTokens(ctx, userID, totalTokenDelta); accountErr != nil {
		log.Printf("ws-chat: UpdateUserTokens (non-fatal): %v", accountErr)
	}

	// Step 7: Send state delta — player, current room, and any world events.
	delta := game.StateDelta{
		Events: engineerResult.Events,
	}
	playerView := g.BuildGameStateView(nil).Player
	delta.Player = &playerView
	if g.Player.LocationID != preTurnPlayerLoc || true { // always send current room
		roomView := g.BuildGameStateView(nil).CurrentRoom
		delta.CurrentRoom = &roomView
	}

	_ = ws.SendDelta(ctx, connID, delta)

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
