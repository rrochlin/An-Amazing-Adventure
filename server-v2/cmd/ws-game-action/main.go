// ws-game-action handles direct player actions that mutate game state without AI:
// move, pick_up, drop.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

type actionRequest struct {
	Action    string `json:"action"`
	SubAction string `json:"sub_action"` // "move" | "pick_up" | "drop"
	Payload   string `json:"payload"`    // direction or item name
}

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connID := req.RequestContext.ConnectionID

	var msg actionRequest
	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	conn, err := dbClient.GetConnection(ctx, connID)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 410}, nil
	}

	if conn.Streaming {
		ws, _ := wsutil.New(ctx)
		_ = ws.Send(ctx, connID, wsutil.Frame{Type: wsutil.FrameStreamingBlocked})
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	saveState, err := dbClient.GetGame(ctx, conn.GameID)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 404}, nil
	}

	g, err := game.FromSaveState(saveState)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	ws, err := wsutil.New(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Execute the action
	var actionErr error
	switch msg.SubAction {
	case "move":
		_, actionErr = g.MovePlayer(msg.Payload)
	case "pick_up":
		item, findErr := g.GetItemByName(msg.Payload)
		if findErr != nil {
			actionErr = findErr
			break
		}
		currentRoom, roomErr := g.GetRoom(g.Player.LocationID)
		if roomErr != nil {
			actionErr = roomErr
			break
		}
		if !currentRoom.HasItem(item.ID) {
			actionErr = fmt.Errorf("item %q is not in this room", msg.Payload)
			break
		}
		_ = currentRoom.RemoveItemID(item.ID)
		g.UpdateRoom(currentRoom)
		actionErr = g.Player.AddItemID(item.ID)
	case "drop":
		item, findErr := g.GetItemByName(msg.Payload)
		if findErr != nil {
			actionErr = findErr
			break
		}
		if !g.Player.HasItem(item.ID) {
			actionErr = fmt.Errorf("you don't have %q", msg.Payload)
			break
		}
		room, roomErr := g.GetRoom(g.Player.LocationID)
		if roomErr != nil {
			actionErr = roomErr
			break
		}
		actionErr = g.TakeItemFromPlayer(item.ID, room.ID)
	default:
		actionErr = fmt.Errorf("unknown sub_action: %s", msg.SubAction)
	}

	if actionErr != nil {
		log.Printf("ws-game-action: %s: %v", msg.SubAction, actionErr)
		_ = ws.SendError(ctx, connID, actionErr.Error())
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	// Persist
	g.Version++
	saved := g.ToSaveState(saveState.Narrative, saveState.ChatHistory)
	if err := dbClient.PutGame(ctx, saved); err != nil {
		log.Printf("ws-game-action: put game: %v", err)
		_ = ws.SendError(ctx, connID, "Failed to save game state")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Send full state update after any direct action
	stateView := g.BuildGameStateView(saveState.ChatHistory)
	_ = ws.SendFullState(ctx, connID, stateView)

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
