// http-games handles all /api/games* REST routes via API Gateway V2 HTTP API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	SessionID         string   `json:"session_id"`
	UserID            string   `json:"user_id"`
	PlayerName        string   `json:"player_name"`
	PlayerDescription string   `json:"player_description,omitempty"`
	PlayerAge         string   `json:"player_age,omitempty"`
	PlayerBackstory   string   `json:"player_backstory,omitempty"`
	ThemeHint         string   `json:"theme_hint,omitempty"`
	Preferences       []string `json:"preferences,omitempty"`
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
	default:
		return jsonResponse(404, map[string]string{"error": "not found"}), nil
	}
}

type gameListItem struct {
	SessionID         string `json:"session_id"`
	PlayerName        string `json:"player_name"`
	Ready             bool   `json:"ready"`
	Title             string `json:"title,omitempty"`
	Theme             string `json:"theme,omitempty"`
	QuestGoal         string `json:"quest_goal,omitempty"`
	ConversationCount int    `json:"conversation_count,omitempty"`
	TotalTokens       int    `json:"total_tokens,omitempty"`
}

type userQuotaInfo struct {
	TokensUsed int    `json:"tokens_used"`
	TokenLimit int    `json:"token_limit"` // 0 = unlimited
	AIEnabled  bool   `json:"ai_enabled"`
	Role       string `json:"role"`
}

func handleListGames(ctx context.Context, userID string) (events.APIGatewayV2HTTPResponse, error) {
	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	// Query 1: sessions the user owns
	ownedIDs, err := dbClient.ListGamesByOwner(ctx, userID)
	if err != nil {
		log.Printf("list games (owned): %v", err)
		return serverError(), nil
	}

	// Query 2: sessions the user has joined as a member
	memberIDs, err := dbClient.GetMemberSessions(ctx, userID)
	if err != nil {
		log.Printf("list games (memberships): %v", err)
		// Non-fatal — fall back to owned only
		memberIDs = nil
	}

	// Merge and deduplicate
	seen := make(map[string]bool, len(ownedIDs))
	allIDs := make([]string, 0, len(ownedIDs)+len(memberIDs))
	for _, id := range append(ownedIDs, memberIDs...) {
		if !seen[id] {
			seen[id] = true
			allIDs = append(allIDs, id)
		}
	}

	saves, err := dbClient.BatchGetSessions(ctx, allIDs)
	if err != nil {
		log.Printf("batch get sessions: %v", err)
		return serverError(), nil
	}

	results := make([]gameListItem, 0, len(saves))
	for _, s := range saves {
		// Determine player name: prefer Players map (v2), fall back to legacy Player field.
		playerName := s.Player.Name
		if s.Players != nil {
			ownerKey := s.OwnerID
			if ownerKey == "" {
				ownerKey = s.UserID
			}
			if pc, ok := s.Players[ownerKey]; ok && pc.Name != "" {
				playerName = pc.Name
			}
		}
		results = append(results, gameListItem{
			SessionID:         s.SessionID,
			PlayerName:        playerName,
			Ready:             s.Ready,
			Title:             s.Title,
			Theme:             s.Theme,
			QuestGoal:         s.QuestGoal,
			ConversationCount: s.ConversationCount,
			TotalTokens:       s.TotalTokens,
		})
	}

	// Include user quota info so the frontend can display usage bar
	quota := userQuotaInfo{Role: "restricted"}
	if ur, err := dbClient.GetUser(ctx, userID); err == nil && ur != nil {
		quota = userQuotaInfo{
			TokensUsed: ur.TokensUsed,
			TokenLimit: ur.TokenLimit,
			AIEnabled:  ur.AIEnabled,
			Role:       ur.Role,
		}
	}

	return jsonResponse(200, map[string]any{
		"games":      results,
		"user_quota": quota,
	}), nil
}

func handleCreateGame(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	var body struct {
		PlayerName        string   `json:"player_name"`
		PlayerDescription string   `json:"player_description"`
		PlayerAge         string   `json:"player_age"`
		PlayerBackstory   string   `json:"player_backstory"`
		ThemeHint         string   `json:"theme_hint"`
		Preferences       []string `json:"preferences"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonResponse(400, map[string]string{"error": "invalid request body"}), nil
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	// Load user record to enforce game limit and determine preview mode.
	// On error or missing record, default to restricted (safe fallback).
	userRecord, err := dbClient.GetUser(ctx, userID)
	if err != nil {
		log.Printf("http-games POST: GetUser error (treating as restricted): %v", err)
	}
	if userRecord == nil {
		userRecord = &db.UserRecord{Role: "restricted", AIEnabled: false, GamesLimit: 1}
	}

	// Enforce games limit
	if userRecord.GamesLimit > 0 {
		count, countErr := dbClient.CountUserGames(ctx, userID)
		if countErr == nil && count >= userRecord.GamesLimit {
			return jsonResponse(403, map[string]string{
				"error":   "games_limit_reached",
				"message": fmt.Sprintf("Game limit of %d reached", userRecord.GamesLimit),
			}), nil
		}
	}

	previewMode := !userRecord.AIEnabled

	sessionID := game.NewSessionID()
	// Player name may be blank — world-gen will invent one if so
	playerName := body.PlayerName
	if playerName == "" {
		playerName = "Adventurer" // placeholder until world-gen writes back the AI name
	}
	player := game.NewCharacter(playerName, body.PlayerDescription)
	player.Age = body.PlayerAge
	player.Backstory = body.PlayerBackstory

	g := game.NewGame(sessionID, userID)
	g.SetPlayerCharacter(userID, player)
	g.CreationParams = game.AdventureCreationParams{
		PlayerDescription: body.PlayerDescription,
		PlayerAge:         body.PlayerAge,
		PlayerBackstory:   body.PlayerBackstory,
		ThemeHint:         body.ThemeHint,
		Preferences:       body.Preferences,
	}

	// Save the initial (not-ready) game record
	saved := g.ToSaveState(nil, nil)
	if err := dbClient.PutGame(ctx, saved); err != nil {
		log.Printf("create game put: %v", err)
		return serverError(), nil
	}

	// Write owner membership record so the user appears in GetMemberSessions results
	if err := dbClient.PutMembership(ctx, db.MembershipRecord{
		UserID:    db.BinaryID(userID),
		SessionID: db.BinaryID(sessionID),
		Role:      "owner",
		JoinedAt:  0, // zero is fine — not currently queried
	}); err != nil {
		log.Printf("create game PutMembership (non-fatal): %v", err)
	}

	// Kick off world generation asynchronously (skipped in preview mode)
	if !previewMode {
		payload, _ := json.Marshal(worldGenPayload{
			SessionID:         sessionID,
			UserID:            userID,
			PlayerName:        body.PlayerName, // pass original (possibly empty) name to world-gen
			PlayerDescription: body.PlayerDescription,
			PlayerAge:         body.PlayerAge,
			PlayerBackstory:   body.PlayerBackstory,
			ThemeHint:         body.ThemeHint,
			Preferences:       body.Preferences,
		})
		if err := invokeWorldGen(ctx, payload); err != nil {
			log.Printf("invoke world-gen: %v (game still created)", err)
		}
	}

	return jsonResponse(201, map[string]any{
		"session_id":   sessionID,
		"ready":        false,
		"preview_mode": previewMode,
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
	if !isAuthorizedForSession(saveState, userID) {
		return jsonResponse(403, map[string]string{"error": "forbidden"}), nil
	}
	g, err := game.FromSaveState(saveState)
	if err != nil {
		return serverError(), nil
	}
	stateView := g.BuildGameStateView(userID, saveState.ChatHistory)
	return jsonResponse(200, map[string]any{
		"session_id":         sessionID,
		"ready":              saveState.Ready,
		"state":              stateView,
		"title":              saveState.Title,
		"theme":              saveState.Theme,
		"quest_goal":         saveState.QuestGoal,
		"total_tokens":       saveState.TotalTokens,
		"conversation_count": saveState.ConversationCount,
		"creation_params":    saveState.CreationParams,
	}), nil
}

func handleDeleteGame(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	sessionID := req.PathParameters["uuid"]
	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	// Load session to verify caller is the owner (not just a member)
	saveState, err := dbClient.GetGame(ctx, sessionID)
	if err != nil {
		return jsonResponse(404, map[string]string{"error": "game not found"}), nil
	}
	ownerID := saveState.OwnerID
	if ownerID == "" {
		ownerID = saveState.UserID
	}
	if ownerID != userID {
		return jsonResponse(403, map[string]string{"error": "only the session owner can delete a game"}), nil
	}

	// Delete the session record
	if err := dbClient.DeleteGame(ctx, sessionID, userID); err != nil {
		log.Printf("delete game %s: %v", sessionID, err)
		return jsonResponse(404, map[string]string{"error": "game not found or not owned by user"}), nil
	}

	// Clean up all membership records for this session (best-effort)
	members, membErr := dbClient.GetSessionMembers(ctx, sessionID)
	if membErr == nil {
		for _, m := range members {
			if delErr := dbClient.DeleteMembership(ctx, string(m.UserID), sessionID); delErr != nil {
				log.Printf("delete membership for user %s session %s (non-fatal): %v", m.UserID, sessionID, delErr)
			}
		}
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

// isAuthorizedForSession returns true if userID is the owner or a party member.
func isAuthorizedForSession(ss game.SaveState, userID string) bool {
	if ss.UserID == userID || ss.OwnerID == userID {
		return true
	}
	if ss.Players != nil {
		if _, ok := ss.Players[userID]; ok {
			return true
		}
	}
	return false
}

func matchesGamePath(path string) bool {
	// matches /api/games/{uuid} — must have a non-empty segment after /api/games/
	const prefix = "/api/games/"
	return len(path) > len(prefix) && path[:len(prefix)] == prefix
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
