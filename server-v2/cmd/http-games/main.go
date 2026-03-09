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
	SessionID      string                     `json:"session_id"`
	UserID         string                     `json:"user_id"`
	CreationParams game.CharacterCreationData `json:"creation_params"`
	// Legacy fields — preserved for backward-compat with old world-gen code path
	PlayerName        string   `json:"player_name,omitempty"`
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
	reqID := req.RequestContext.RequestID

	log.Printf("http-games: %s %s user=%s req=%s", method, path, userID, reqID)

	var resp events.APIGatewayV2HTTPResponse
	var err error
	switch {
	case method == "GET" && path == "/api/games":
		resp, err = handleListGames(ctx, userID)
	case method == "POST" && path == "/api/games":
		resp, err = handleCreateGame(ctx, req, userID)
	case method == "GET" && matchesGamePath(path) && !matchesJoinCharacterPath(path) && !matchesRetryWorldGenPath(path):
		resp, err = handleGetGame(ctx, req, userID)
	case method == "DELETE" && matchesGamePath(path):
		resp, err = handleDeleteGame(ctx, req, userID)
	case method == "POST" && matchesJoinCharacterPath(path):
		resp, err = handleJoinCharacter(ctx, req, userID)
	case method == "POST" && matchesRetryWorldGenPath(path):
		resp, err = handleRetryWorldGen(ctx, req, userID)
	default:
		resp, err = jsonResponse(404, map[string]string{"error": "not found"}), nil
	}

	log.Printf("http-games: %s %s → %d (req=%s)", method, path, resp.StatusCode, reqID)
	return resp, err
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
		// Determine player name: prefer PlayersData (v3) → Players map (v2) → legacy Player field.
		ownerKey := s.OwnerID
		if ownerKey == "" {
			ownerKey = s.UserID
		}
		playerName := s.Player.Name
		if s.Players != nil {
			if pc, ok := s.Players[ownerKey]; ok && pc.Name != "" {
				playerName = pc.Name
			}
		}
		if s.PlayersData != nil {
			if pd, ok := s.PlayersData[ownerKey]; ok && pd != nil && pd.Name != "" {
				playerName = pd.Name
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
	if ur, err := dbClient.GetUser(ctx, userID); err != nil {
		log.Printf("list games: GetUser error (quota will show restricted): %v", err)
	} else if ur != nil {
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
	var body game.CharacterCreationData
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
	log.Printf("http-games POST: user=%s role=%s ai_enabled=%v preview=%v", userID, userRecord.Role, userRecord.AIEnabled, previewMode)

	sessionID := game.NewSessionID()

	// Build the D&D character from creation data (validates class/race/skills).
	// In preview mode, skip character creation (no AI, no D&D mechanics).
	playerName := body.Name
	if playerName == "" {
		playerName = "Adventurer"
	}

	// Always create the legacy stub for room placement tracking
	player := game.NewCharacter(playerName, "")
	g := game.NewGame(sessionID, userID)
	g.SetPlayerCharacter(userID, player)
	g.CreationParams = body

	// Build the full D&D character if we have enough data
	if body.ClassID != "" && body.RaceID != "" && len(body.AbilityScores) == 6 {
		dndChar, err := game.BuildDnDCharacter(ctx, body)
		if err != nil {
			log.Printf("http-games POST: BuildDnDCharacter error: %v", err)
			return jsonResponse(400, map[string]string{"error": fmt.Sprintf("character creation failed: %v", err)}), nil
		}
		g.SetDnDCharacter(userID, dndChar)
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
		log.Printf("http-games POST: invoking world-gen for session %s", sessionID)
		payload, _ := json.Marshal(worldGenPayload{
			SessionID:      sessionID,
			UserID:         userID,
			CreationParams: body,
			// Legacy fields for backward-compat
			PlayerName:  playerName,
			ThemeHint:   body.ThemeHint,
			Preferences: body.Preferences,
		})
		if err := invokeWorldGen(ctx, payload); err != nil {
			log.Printf("http-games POST: invoke world-gen FAILED for session %s: %v (game still created)", sessionID, err)
		} else {
			log.Printf("http-games POST: world-gen invoked for session %s", sessionID)
		}
	} else {
		log.Printf("http-games POST: skipping world-gen for session %s (preview mode)", sessionID)
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

	// Load DnD characters from SaveState for enriched CharacterView
	if saveState.PlayersData != nil {
		bus, loadErr := g.LoadDnDCharacters(ctx, saveState.PlayersData)
		if loadErr != nil {
			log.Printf("handleGetGame LoadDnDCharacters (non-fatal): %v", loadErr)
		} else {
			_ = bus
		}
	}

	stateView := g.BuildGameStateView(userID, saveState.ChatHistory)
	return jsonResponse(200, map[string]any{
		"session_id":            sessionID,
		"ready":                 saveState.Ready,
		"state":                 stateView,
		"title":                 saveState.Title,
		"theme":                 saveState.Theme,
		"quest_goal":            saveState.QuestGoal,
		"total_tokens":          saveState.TotalTokens,
		"conversation_count":    saveState.ConversationCount,
		"creation_params":       saveState.CreationParams,
		"needs_character_reset": g.NeedsCharacterReset,
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
		log.Printf("invokeWorldGen: WORLD_GEN_ARN not set, skipping")
		return nil
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}
	client := awslambda.NewFromConfig(cfg)
	out, err := client.Invoke(ctx, &awslambda.InvokeInput{
		FunctionName:   aws.String(fnName),
		InvocationType: awslambdatypes.InvocationTypeEvent, // async, no wait
		Payload:        payload,
	})
	if err != nil {
		return fmt.Errorf("lambda invoke %s: %w", fnName, err)
	}
	log.Printf("invokeWorldGen: dispatched to %s status=%d", fnName, out.StatusCode)
	return nil
}

func matchesJoinCharacterPath(path string) bool {
	// matches /api/games/{uuid}/join-character
	const suffix = "/join-character"
	return len(path) > len(suffix) && path[len(path)-len(suffix):] == suffix
}

func matchesRetryWorldGenPath(path string) bool {
	// matches /api/games/{uuid}/retry-world-gen
	const suffix = "/retry-world-gen"
	return len(path) > len(suffix) && path[len(path)-len(suffix):] == suffix
}

// handleRetryWorldGen re-invokes world-gen for a session that is stuck in not-ready state.
// Only the session owner can retry. Only allowed when ready=false.
func handleRetryWorldGen(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	p := req.RequestContext.HTTP.Path
	const suffix = "/retry-world-gen"
	const prefix = "/api/games/"
	sessionID := ""
	if len(p) > len(prefix)+len(suffix) {
		sessionID = p[len(prefix) : len(p)-len(suffix)]
	}
	if sessionID == "" {
		return jsonResponse(400, map[string]string{"error": "missing session id"}), nil
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	saveState, err := dbClient.GetGame(ctx, sessionID)
	if err != nil {
		return jsonResponse(404, map[string]string{"error": "game not found"}), nil
	}

	// Only the owner can trigger a retry.
	ownerID := saveState.OwnerID
	if ownerID == "" {
		ownerID = saveState.UserID
	}
	if ownerID != userID {
		return jsonResponse(403, map[string]string{"error": "only the session owner can retry world generation"}), nil
	}

	// Refuse if the game is already ready — nothing to retry.
	if saveState.Ready {
		return jsonResponse(409, map[string]string{"error": "game is already ready"}), nil
	}

	// Check the user still has AI access (in case their record changed).
	userRecord, err := dbClient.GetUser(ctx, userID)
	if err != nil {
		log.Printf("handleRetryWorldGen: GetUser error user=%s: %v", userID, err)
	}
	if userRecord == nil || !userRecord.AIEnabled {
		log.Printf("handleRetryWorldGen: ai_access_not_enabled for user=%s (record=%v)", userID, userRecord != nil)
		return jsonResponse(403, map[string]string{"error": "ai_access_not_enabled"}), nil
	}

	payload, _ := json.Marshal(worldGenPayload{
		SessionID:      sessionID,
		UserID:         userID,
		CreationParams: saveState.CreationParams,
		PlayerName:     saveState.Player.Name,
		ThemeHint:      saveState.CreationParams.ThemeHint,
		Preferences:    saveState.CreationParams.Preferences,
	})

	log.Printf("handleRetryWorldGen: invoking world-gen for session %s user=%s", sessionID, userID)
	if err := invokeWorldGen(ctx, payload); err != nil {
		log.Printf("handleRetryWorldGen: invoke world-gen FAILED for session %s: %v", sessionID, err)
		return serverError(), nil
	}

	return jsonResponse(202, map[string]string{"status": "world generation restarted"}), nil
}

// handleJoinCharacter updates a member's character stub with real character details.
// Called by party members after they've been added via invite flow.
func handleJoinCharacter(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	sessionID := req.PathParameters["uuid"]
	if sessionID == "" {
		// Try extracting from path: /api/games/{uuid}/join-character
		p := req.RequestContext.HTTP.Path
		const suffix = "/join-character"
		const prefix = "/api/games/"
		if len(p) > len(prefix)+len(suffix) {
			sessionID = p[len(prefix) : len(p)-len(suffix)]
		}
	}
	if sessionID == "" {
		return jsonResponse(400, map[string]string{"error": "missing session id"}), nil
	}

	var body game.CharacterCreationData
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonResponse(400, map[string]string{"error": "invalid body"}), nil
	}

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

	// Load existing DnD players from SaveState so ToSaveState doesn't lose them
	if saveState.PlayersData != nil {
		bus, loadErr := g.LoadDnDCharacters(ctx, saveState.PlayersData)
		if loadErr != nil {
			log.Printf("handleJoinCharacter LoadDnDCharacters (non-fatal): %v", loadErr)
		} else {
			_ = bus // bus scoped to this invocation
		}
	}

	playerName := body.Name
	if playerName == "" {
		playerName = "Adventurer"
	}
	// Legacy stub for room placement
	char := game.NewCharacter(playerName, "")
	g.SetPlayerCharacter(userID, char)

	// Build D&D character if creation data is complete
	if body.ClassID != "" && body.RaceID != "" && len(body.AbilityScores) == 6 {
		dndChar, charErr := game.BuildDnDCharacter(ctx, body)
		if charErr != nil {
			log.Printf("handleJoinCharacter BuildDnDCharacter: %v", charErr)
			return jsonResponse(400, map[string]string{"error": fmt.Sprintf("character creation failed: %v", charErr)}), nil
		}
		g.SetDnDCharacter(userID, dndChar)
	}

	g.Version++

	updated := g.ToSaveState(saveState.Narrative, saveState.ChatHistory)
	if err := dbClient.PutGame(ctx, updated); err != nil {
		log.Printf("handleJoinCharacter PutGame: %v", err)
		return serverError(), nil
	}

	return jsonResponse(200, map[string]string{"session_id": sessionID}), nil
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
