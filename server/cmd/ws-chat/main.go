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
	reqID := req.RequestContext.RequestID
	log.Printf("ws-chat: conn=%s req=%s", connID, reqID)

	// Parse message
	var msg chatRequest
	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		log.Printf("ws-chat: bad body conn=%s: %v", connID, err)
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

	// Block concurrent chats: check if ANY connection for this session is already streaming.
	gameConns, gcErr := dbClient.GetConnectionsByGameID(ctx, conn.GameID)
	if gcErr != nil {
		log.Printf("ws-chat: GetConnectionsByGameID: %v", gcErr)
		// Fall back to checking just this connection
		gameConns = []db.Connection{conn}
	}
	for _, gc := range gameConns {
		if gc.Streaming {
			ws, _ := wsutil.New(ctx)
			_ = ws.Send(ctx, connID, wsutil.Frame{Type: wsutil.FrameStreamingBlocked})
			return events.APIGatewayProxyResponse{StatusCode: 200}, nil
		}
	}

	// Mark streaming = true on this connection only
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
	log.Printf("ws-chat: conn=%s user=%s game=%s", connID, userID, conn.GameID)
	userRecord, err := dbClient.GetUser(ctx, userID)
	if err != nil {
		log.Printf("ws-chat: GetUser error user=%s: %v", userID, err)
		_ = ws.SendError(ctx, connID, "internal_error")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
	if userRecord == nil {
		log.Printf("ws-chat: user record not found for user=%s — rejecting chat", userID)
		_ = ws.SendError(ctx, connID, "user_not_found")
		return events.APIGatewayProxyResponse{StatusCode: 403}, nil
	}
	if !userRecord.AIEnabled {
		log.Printf("ws-chat: ai_access_not_enabled for user=%s role=%s", userID, userRecord.Role)
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

	// Load D&D characters for this invocation (binds a fresh event bus)
	if saveState.PlayersData != nil {
		if _, loadErr := g.LoadDnDCharacters(ctx, saveState.PlayersData); loadErr != nil {
			log.Printf("ws-chat: LoadDnDCharacters (non-fatal): %v", loadErr)
		}
	}

	// Set up Bedrock client
	aiClient, err := ai.New(ctx)
	if err != nil {
		log.Printf("ws-chat: ai init: %v", err)
		_ = ws.SendError(ctx, connID, "AI unavailable")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Capture pre-turn state for delta calculation
	preTurnOwner, _ := g.OwnerCharacter()
	preTurnPlayerLoc := preTurnOwner.LocationID

	// Build connection ID list for broadcast (refresh after streaming flag is set)
	allGameConns, _ := dbClient.GetConnectionsByGameID(ctx, conn.GameID)
	allConnIDs := make([]string, 0, len(allGameConns))
	for _, gc := range allGameConns {
		allConnIDs = append(allConnIDs, gc.ConnectionID)
	}
	if len(allConnIDs) == 0 {
		allConnIDs = []string{connID} // fallback to sender only
	}

	// Step 1: Stream narrator prose — broadcast each chunk to all party members.
	narratorResult, err := aiClient.NarrateStream(
		ctx, g, saveState.Narrative, msg.Content,
		func(chunk string) {
			chunkFrame := wsutil.Frame{
				Type:    wsutil.FrameNarrativeChunk,
				Payload: map[string]string{"content": chunk},
			}
			stale, _ := ws.Broadcast(ctx, allConnIDs, chunkFrame)
			for _, s := range stale {
				_ = dbClient.DeleteConnection(ctx, s)
			}
		},
	)
	if err != nil {
		log.Printf("ws-chat: narrator error: %v", err)
		_ = ws.SendError(ctx, connID, "Narrator error — please try again")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Step 2: Signal streaming complete — broadcast to all party members.
	staleEnds, _ := ws.Broadcast(ctx, allConnIDs, wsutil.Frame{Type: wsutil.FrameNarrativeEnd})
	for _, s := range staleEnds {
		_ = dbClient.DeleteConnection(ctx, s)
	}

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

	// Step 7: Send per-member state delta — each party member gets their own
	// perspective (their own character's location and inventory).
	postTurnOwner, _ := g.OwnerCharacter()
	postTurnOwnerLoc := postTurnOwner.LocationID
	// Refresh connection list for delta fanout (some may have disconnected)
	freshConns, _ := dbClient.GetConnectionsByGameID(ctx, conn.GameID)
	for _, gc := range freshConns {
		memberUID := string(gc.UserID)
		memberView := g.BuildGameStateView(memberUID, nil)
		delta := game.StateDelta{
			Events: engineerResult.Events,
			Player: &memberView.Player,
			Self:   &memberView.Self,
		}
		if postTurnOwnerLoc != preTurnPlayerLoc || true { // always send current room
			delta.CurrentRoom = &memberView.CurrentRoom
		}
		if sendErr := ws.SendDelta(ctx, gc.ConnectionID, delta); sendErr != nil {
			log.Printf("ws-chat: send delta to %s: %v", gc.ConnectionID, sendErr)
			_ = dbClient.DeleteConnection(ctx, gc.ConnectionID)
		}
	}

	log.Printf("ws-chat: complete conn=%s user=%s game=%s turns=%d tokens=%d", connID, userID, conn.GameID, g.ConversationCount, g.TotalTokens)
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
