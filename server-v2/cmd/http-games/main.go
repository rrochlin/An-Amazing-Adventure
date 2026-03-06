// http-games handles all /api/games* REST routes via API Gateway V2 HTTP API.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	awslambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// worldGenPayload is passed to the world-gen Lambda as its event.
type worldGenPayload struct {
	SessionID  string `json:"session_id"`
	UserID     string `json:"user_id"`
	PlayerName string `json:"player_name"`
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// Extract authenticated user ID from Cognito JWT authorizer claims
	userID := req.RequestContext.Authorizer.JWT.Claims["sub"]
	if userID == "" {
		return jsonResponse(401, map[string]string{"error": "unauthorized"}), nil
	}

	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path

	switch {
	case method == "GET" && path == "/api/games":
		return handleListGames(ctx, userID)
	case method == "POST" && path == "/api/games":
		return handleCreateGame(ctx, req, userID)
	case method == "GET" && matchesGamePath(path):
		return handleGetGame(ctx, req, userID)
	case method == "DELETE" && matchesGamePath(path):
		return handleDeleteGame(ctx, req, userID)
	case method == "GET" && matchesWorldReadyPath(path):
		return handleWorldReady(ctx, req, userID)
	default:
		return jsonResponse(404, map[string]string{"error": "not found"}), nil
	}
}

func handleListGames(ctx context.Context, userID string) (events.APIGatewayV2HTTPResponse, error) {
	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}
	saves, err := dbClient.ListGames(ctx, userID)
	if err != nil {
		log.Printf("list games: %v", err)
		return serverError(), nil
	}
	type gameInfo struct {
		SessionID  string `json:"session_id"`
		PlayerName string `json:"player_name"`
		Ready      bool   `json:"ready"`
	}
	results := make([]gameInfo, 0, len(saves))
	for _, s := range saves {
		results = append(results, gameInfo{
			SessionID:  s.SessionID,
			PlayerName: s.Player.Name,
			Ready:      s.Ready,
		})
	}
	return jsonResponse(200, results), nil
}

func handleCreateGame(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	var body struct {
		PlayerName string `json:"player_name"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.PlayerName == "" {
		return jsonResponse(400, map[string]string{"error": "player_name is required"}), nil
	}

	sessionID := game.NewSessionID()
	player := game.NewCharacter(body.PlayerName, "The hero of the adventure")

	g := game.NewGame(sessionID, userID)
	g.Player = player

	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	// Save the initial (not-ready) game record
	saved := g.ToSaveState(nil, nil)
	if err := dbClient.PutGame(ctx, saved); err != nil {
		log.Printf("create game put: %v", err)
		return serverError(), nil
	}

	// Kick off world generation asynchronously
	payload, _ := json.Marshal(worldGenPayload{
		SessionID:  sessionID,
		UserID:     userID,
		PlayerName: body.PlayerName,
	})
	if err := invokeWorldGen(ctx, payload); err != nil {
		log.Printf("invoke world-gen: %v (game still created, world gen may be delayed)", err)
	}

	return jsonResponse(201, map[string]any{
		"session_id": sessionID,
		"ready":      false,
	}), nil
}

func handleGetGame(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	sessionID := req.PathParameters["uuid"]
	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}
	saveState, err := dbClient.GetGame(ctx, sessionID)
	if err != nil {
		return jsonResponse(404, map[string]string{"error": "game not found"}), nil
	}
	if saveState.UserID != userID {
		return jsonResponse(403, map[string]string{"error": "forbidden"}), nil
	}
	g, err := game.FromSaveState(saveState)
	if err != nil {
		return serverError(), nil
	}
	stateView := g.BuildGameStateView(saveState.ChatHistory)
	return jsonResponse(200, map[string]any{
		"session_id": sessionID,
		"ready":      saveState.Ready,
		"state":      stateView,
	}), nil
}

func handleDeleteGame(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	sessionID := req.PathParameters["uuid"]
	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}
	if err := dbClient.DeleteGame(ctx, sessionID, userID); err != nil {
		log.Printf("delete game %s: %v", sessionID, err)
		return jsonResponse(404, map[string]string{"error": "game not found or not owned by user"}), nil
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: 204}, nil
}

func handleWorldReady(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	sessionID := req.PathParameters["uuid"]
	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}
	ready, err := dbClient.GetGameReady(ctx, sessionID)
	if err != nil {
		return jsonResponse(404, map[string]string{"error": "game not found"}), nil
	}
	if ready {
		return events.APIGatewayV2HTTPResponse{StatusCode: 200}, nil
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: 204}, nil
}

// invokeWorldGen fires the world-gen Lambda asynchronously (Event invocation type).
func invokeWorldGen(ctx context.Context, payload []byte) error {
	fnName := os.Getenv("WORLD_GEN_ARN")
	if fnName == "" {
		return nil
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	client := awslambda.NewFromConfig(cfg)
	_, err = client.Invoke(ctx, &awslambda.InvokeInput{
		FunctionName:   aws.String(fnName),
		InvocationType: awslambdatypes.InvocationTypeEvent, // async, no wait
		Payload:        payload,
	})
	return err
}

func matchesGamePath(path string) bool {
	// matches /api/games/{uuid}
	return len(path) > len("/api/games/")
}

func matchesWorldReadyPath(path string) bool {
	return len(path) > len("/api/worldready/")
}

func jsonResponse(code int, body any) events.APIGatewayV2HTTPResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func serverError() events.APIGatewayV2HTTPResponse {
	return jsonResponse(500, map[string]string{"error": "internal server error"})
}

func main() {
	lambda.Start(handler)
}
