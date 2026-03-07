// ws-chat handles the WebSocket "chat" route.
// It streams the narrator's response back to the client chunk-by-chunk,
// executes any tool calls the AI makes against the game state,
// then persists the updated game state and sends a state delta.
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

	// Set up WebSocket sender
	ws, err := wsutil.New(ctx)
	if err != nil {
		log.Printf("ws-chat: ws sender init: %v", err)
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

	// Stream narrator response
	result, err := aiClient.NarrateStream(
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

	// Signal streaming complete
	_ = ws.SendNarrativeEnd(ctx, connID)

	// Append chat history entries
	history := saveState.ChatHistory
	history = append(history, game.ChatMessage{Type: "player", Content: msg.Content})
	history = append(history, game.ChatMessage{Type: "narrative", Content: result.Narrative})

	// Persist updated game state with optimistic locking retry
	g.Version++
	saved := g.ToSaveState(result.NewMessages, history)
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

	// Send state delta — always include player and current room;
	// include any rooms that changed (player moved or NPCs/items mutated)
	delta := game.StateDelta{
		NewMessage: &game.ChatMessage{Type: "narrative", Content: result.Narrative},
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
