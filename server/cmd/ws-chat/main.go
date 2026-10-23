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
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

var (
	campaignRegistryOnce sync.Once
	campaignRegistry     *campaigns.Registry
	campaignRegistryErr  error
)

type dbClient interface {
	GetConnection(context.Context, string) (db.Connection, error)
	GetConnectionsByGameID(context.Context, string) ([]db.Connection, error)
	SetStreaming(context.Context, string, bool) error
	DeleteConnection(context.Context, string) error
	GetUser(context.Context, string) (*db.UserRecord, error)
	GetGame(context.Context, string) (game.SaveState, error)
	PutMutation(context.Context, game.MutationEntry) error
	PutGame(context.Context, game.SaveState) error
	UpdateUserTokens(context.Context, string, int) error
}

// aiClient is the model access the story pipelines need.
type aiClient interface {
	story.Models
	story.GMModels
}

type wsSender interface {
	Send(context.Context, string, wsutil.Frame) error
	Broadcast(context.Context, []string, wsutil.Frame) ([]string, error)
	SendDelta(context.Context, string, any) error
	SendError(context.Context, string, string) error
}

var (
	newDBClient = func(ctx context.Context) (dbClient, error) { return db.New(ctx) }
	newAIClient = func(ctx context.Context) (aiClient, error) { return ai.New(ctx) }
	newWSSender = func(ctx context.Context) (wsSender, error) { return wsutil.New(ctx) }
	// newPipeline picks the story pipeline for a turn: the campaign's choice.
	newPipeline = func(models aiClient, def *campaigns.CampaignDefinition) story.Pipeline {
		return story.PipelineFor(def, models)
	}
)

type chatRequest struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}

func getCampaignRegistry() (*campaigns.Registry, error) {
	campaignRegistryOnce.Do(func() {
		campaignRegistry, campaignRegistryErr = campaigns.LoadEmbeddedRegistry()
	})
	return campaignRegistry, campaignRegistryErr
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

	dbClient, err := newDBClient(ctx)
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
	ws, err := newWSSender(ctx)
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

	// Record the whole turn for debugging; stored however the turn ends.
	tr := trace.Start(trace.KindChat, "ws-chat", conn.GameID, g)
	tr.RequestID, tr.ConnectionID, tr.UserID, tr.Input = reqID, connID, userID, msg.Content
	tr.Before.World = trace.WorldOf(saveState) // for replaying the turn
	defer trace.Save(ctx, dbClient, tr, g)
	sendError := func(message string) {
		tr.Errorf("sent error: %s", message)
		tr.Frame(string(wsutil.FrameError), connID, message)
		_ = ws.SendError(ctx, connID, message)
	}

	// Load D&D characters for this invocation (binds a fresh event bus)
	if saveState.PlayersData != nil {
		if _, loadErr := g.LoadDnDCharacters(ctx, saveState.PlayersData); loadErr != nil {
			log.Printf("ws-chat: LoadDnDCharacters (non-fatal): %v", loadErr)
		}
	}

	var campaignDef *campaigns.CampaignDefinition
	if g.Campaign != nil && g.Campaign.CampaignID != "" {
		reg, regErr := getCampaignRegistry()
		if regErr != nil {
			log.Printf("ws-chat: load campaigns: %v", regErr)
			tr.Errorf("load campaigns: %v", regErr)
			return events.APIGatewayProxyResponse{StatusCode: 500}, nil
		}
		var ok bool
		campaignDef, ok = reg.Get(g.Campaign.CampaignID)
		if !ok {
			log.Printf("ws-chat: campaign %q missing for game %s", g.Campaign.CampaignID, conn.GameID)
			sendError("campaign_not_found")
			return events.APIGatewayProxyResponse{StatusCode: 500}, nil
		}
	}

	// Set up Bedrock client
	aiClient, err := newAIClient(ctx)
	if err != nil {
		log.Printf("ws-chat: ai init: %v", err)
		sendError("AI unavailable")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Capture pre-turn state for delta calculation
	preTurnOwner, _ := g.OwnerCharacter()
	preTurnPlayerLoc := preTurnOwner.LocationID

	// Build connection ID list for broadcast (refresh after streaming flag is set)
	allGameConns, allConnsErr := dbClient.GetConnectionsByGameID(ctx, conn.GameID)
	if allConnsErr != nil {
		log.Printf("ws-chat: initial connections fallback for game %s: %v", conn.GameID, allConnsErr)
		allGameConns = []db.Connection{conn}
	} else {
		allGameConns = ensureSenderConnection(allGameConns, conn)
	}
	allConnIDs := make([]string, 0, len(allGameConns))
	for _, gc := range allGameConns {
		allConnIDs = append(allConnIDs, gc.ConnectionID)
	}
	if len(allConnIDs) == 0 {
		allConnIDs = []string{connID}
	}

	if g.Campaign.RunEnded() {
		sendError("run_ended")
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}
	if g.Campaign != nil && g.Campaign.ActiveDialogue != nil && g.Campaign.ActiveDialogue.AwaitingChoice {
		sendError("dialogue_choice_required")
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	// Steps 1-3: run the story pipeline. Narration streams to every party
	// member as it arrives; narrative_end goes out as soon as it's complete,
	// so the client never waits on the post-narration work.
	turn, err := newPipeline(aiClient, campaignDef).RunTurn(ctx, story.TurnInput{
		Game:        g,
		Campaign:    campaignDef,
		History:     saveState.Narrative,
		PlayerInput: msg.Content,
		OnChunk: func(chunk string) {
			chunkFrame := wsutil.Frame{
				Type:    wsutil.FrameNarrativeChunk,
				Payload: map[string]string{"content": chunk},
			}
			tr.Chunk(string(wsutil.FrameNarrativeChunk))
			stale, _ := ws.Broadcast(ctx, allConnIDs, chunkFrame)
			for _, s := range stale {
				_ = dbClient.DeleteConnection(ctx, s)
			}
		},
		OnNarrationDone: func() {
			tr.Frame(string(wsutil.FrameNarrativeEnd), "", nil)
			staleEnds, _ := ws.Broadcast(ctx, allConnIDs, wsutil.Frame{Type: wsutil.FrameNarrativeEnd})
			for _, s := range staleEnds {
				_ = dbClient.DeleteConnection(ctx, s)
			}
		},
	})
	tr.Call(turn.Calls...)
	for _, e := range turn.Errors {
		tr.Errorf("%s", e)
	}
	if turn.Decision != nil {
		tr.Decision = *turn.Decision
	}
	if turn.Resolution != nil {
		tr.Resolution = *turn.Resolution
	}
	if turn.Transition != nil {
		tr.Transition = *turn.Transition
	}
	if err != nil {
		log.Printf("ws-chat: %v", err)
		sendError("Narrator error — please try again")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Append chat history — attach world events to the narrative message so they
	// survive reconnection/reload.
	history := saveState.ChatHistory
	history = append(history, game.ChatMessage{Type: "player", Content: msg.Content})
	history = append(history, game.ChatMessage{
		Type:    "narrative",
		Content: turn.Narrative,
		Events:  turn.NarrativeEvents,
	})
	if turn.AuthoredDialogue != "" {
		history = append(history, game.ChatMessage{Type: "narrative", Content: turn.AuthoredDialogue, Events: turn.SceneEvents})
		chunkFrame := wsutil.Frame{
			Type:    wsutil.FrameNarrativeChunk,
			Payload: map[string]string{"content": turn.AuthoredDialogue},
		}
		tr.Frame(string(chunkFrame.Type), "", chunkFrame.Payload)
		stale, _ := ws.Broadcast(ctx, allConnIDs, chunkFrame)
		for _, s := range stale {
			_ = dbClient.DeleteConnection(ctx, s)
		}
		tr.Frame(string(wsutil.FrameNarrativeEnd), "", nil)
		staleEnds, _ := ws.Broadcast(ctx, allConnIDs, wsutil.Frame{Type: wsutil.FrameNarrativeEnd})
		for _, s := range staleEnds {
			_ = dbClient.DeleteConnection(ctx, s)
		}
	}

	tr.Events = append(append([]game.WorldEvent(nil), turn.NarrativeEvents...), turn.SceneEvents...)

	// Step 4: Persist mutation audit log entries (best-effort — failure is non-fatal).
	putMutations(ctx, dbClient, turn.Mutations)

	// Update stats (include both Narrator and Engineer token usage).
	g.ConversationCount++
	g.TotalTokens += turn.Tokens.Total()

	// Step 5: Persist updated game state with optimistic locking retry.
	g.Version++
	saved := g.ToSaveState(turn.NewMessages, history)
	for attempt := 0; attempt < 3; attempt++ {
		if err := dbClient.PutGame(ctx, saved); err != nil {
			log.Printf("ws-chat: put game attempt %d: %v", attempt+1, err)
			tr.Errorf("put game attempt %d: %v", attempt+1, err)
			if attempt == 2 {
				sendError("Failed to save game state")
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
	totalTokenDelta := turn.Tokens.Total()
	if accountErr := dbClient.UpdateUserTokens(ctx, userID, totalTokenDelta); accountErr != nil {
		log.Printf("ws-chat: UpdateUserTokens (non-fatal): %v", accountErr)
	}

	// Step 7: Send per-member state delta — each party member gets their own
	// perspective (their own character's location and inventory).
	postTurnOwner, _ := g.OwnerCharacter()
	postTurnOwnerLoc := postTurnOwner.LocationID
	// Refresh connection list for delta fanout (some may have disconnected).
	// Fall back to the sender connection so campaign/objective UI still updates
	// even if the connection query is temporarily unavailable.
	freshConns, freshErr := dbClient.GetConnectionsByGameID(ctx, conn.GameID)
	if freshErr != nil {
		log.Printf("ws-chat: refresh connections fallback for game %s: %v", conn.GameID, freshErr)
		freshConns = []db.Connection{conn}
	} else {
		freshConns = ensureSenderConnection(freshConns, conn)
	}
	for _, gc := range freshConns {
		memberUID := string(gc.UserID)
		memberView := g.BuildGameStateView(memberUID, nil)
		delta := game.StateDelta{
			Events:   append(append([]game.WorldEvent(nil), turn.NarrativeEvents...), turn.SceneEvents...),
			Player:   &memberView.Player,
			Self:     &memberView.Self,
			Campaign: g.BuildCampaignStateView(),
			// Every room, so characters moving anywhere show on the map.
			UpdatedRooms: memberView.Rooms,
		}
		if postTurnOwnerLoc != preTurnPlayerLoc || true { // always send current room
			delta.CurrentRoom = &memberView.CurrentRoom
		}
		tr.Frame(string(wsutil.FrameStateDelta), gc.ConnectionID, delta)
		if sendErr := ws.SendDelta(ctx, gc.ConnectionID, delta); sendErr != nil {
			log.Printf("ws-chat: send delta to %s: %v", gc.ConnectionID, sendErr)
			tr.Errorf("send delta to %s: %v", gc.ConnectionID, sendErr)
			_ = dbClient.DeleteConnection(ctx, gc.ConnectionID)
		}
	}

	log.Printf("ws-chat: complete conn=%s user=%s game=%s turns=%d tokens=%d", connID, userID, conn.GameID, g.ConversationCount, g.TotalTokens)
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func putMutations(ctx context.Context, dbClient dbClient, mutations []game.MutationEntry) {
	if len(mutations) == 0 {
		return
	}
	if os.Getenv("MUTATIONS_TABLE") == "" {
		panic("required env var MUTATIONS_TABLE is not set")
	}
	for _, m := range mutations {
		if err := dbClient.PutMutation(ctx, m); err != nil {
			log.Printf("ws-chat: put mutation (tool=%s): %v", m.Tool, err)
		}
	}
}

func ensureSenderConnection(conns []db.Connection, sender db.Connection) []db.Connection {
	for _, existing := range conns {
		if existing.ConnectionID == sender.ConnectionID {
			return conns
		}
	}
	return append(conns, sender)
}

func main() {
	lambda.Start(handler)
}

// isSceneEvent reports events that announce a new scene or its objective.
