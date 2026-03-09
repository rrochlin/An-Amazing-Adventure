// http-invites handles party invite code management.
//
// Routes:
//
//	POST /api/invites          — create a new invite (owner only)
//	GET  /api/invites/{code}   — resolve invite info (no auth required)
//	POST /api/invites/{code}/join — redeem invite and join session (auth required)
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path

	switch {
	case method == "POST" && path == "/api/invites":
		return handleCreateInvite(ctx, req)
	case method == "GET" && strings.HasPrefix(path, "/api/invites/") && !strings.HasSuffix(path, "/join"):
		return handleGetInvite(ctx, req)
	case method == "POST" && strings.HasSuffix(path, "/join"):
		return handleJoinInvite(ctx, req)
	default:
		return jsonResponse(404, map[string]string{"error": "not found"}), nil
	}
}

// -------------------------------------------------------------------
// POST /api/invites
// -------------------------------------------------------------------

type createInviteRequest struct {
	SessionID string `json:"session_id"`
	MaxUses   int    `json:"max_uses"` // 0 = unlimited
	TTLDays   int    `json:"ttl_days"` // default 7
}

type createInviteResponse struct {
	Code    string `json:"code"`
	URL     string `json:"url"`
	Expires int64  `json:"expires"` // Unix ms
}

func handleCreateInvite(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID := req.RequestContext.Authorizer.JWT.Claims["sub"]
	if userID == "" {
		return jsonResponse(401, map[string]string{"error": "unauthorized"}), nil
	}

	var body createInviteRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.SessionID == "" {
		return jsonResponse(400, map[string]string{"error": "session_id is required"}), nil
	}

	ttlDays := body.TTLDays
	if ttlDays <= 0 {
		ttlDays = 7
	}
	maxUses := body.MaxUses
	if maxUses <= 0 {
		maxUses = 10
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	// Verify caller owns the session
	saveState, err := dbClient.GetGame(ctx, body.SessionID)
	if err != nil {
		return jsonResponse(404, map[string]string{"error": "session not found"}), nil
	}
	ownerID := saveState.OwnerID
	if ownerID == "" {
		ownerID = saveState.UserID
	}
	if ownerID != userID {
		return jsonResponse(403, map[string]string{"error": "only the session owner can create invites"}), nil
	}

	code, err := generateInviteCode(6)
	if err != nil {
		log.Printf("http-invites: generate code: %v", err)
		return serverError(), nil
	}

	expiresAt := time.Now().Add(time.Duration(ttlDays) * 24 * time.Hour)

	inv := db.InviteRecord{
		Code:      code,
		SessionID: db.BinaryID(body.SessionID),
		CreatedBy: db.BinaryID(userID),
		ExpiresAt: expiresAt.Unix(), // DynamoDB TTL is in seconds
		MaxUses:   maxUses,
		Uses:      0,
	}
	if err := dbClient.PutInvite(ctx, inv); err != nil {
		log.Printf("http-invites: PutInvite: %v", err)
		return serverError(), nil
	}

	// Denormalize invite code onto the SaveState for quick lookup
	g, _ := game.FromSaveState(saveState)
	if g != nil {
		g.InviteCode = code
		g.Version++
		updated := g.ToSaveState(saveState.Narrative, saveState.ChatHistory)
		if putErr := dbClient.PutGame(ctx, updated); putErr != nil {
			log.Printf("http-invites: update SaveState invite_code (non-fatal): %v", putErr)
		}
	}

	domain := os.Getenv("CLIENT_DOMAIN")
	if domain == "" {
		domain = "https://d1ctll9l3g8cf4.cloudfront.net"
	}
	return jsonResponse(201, createInviteResponse{
		Code:    code,
		URL:     fmt.Sprintf("%s/join/%s", domain, code),
		Expires: expiresAt.UnixMilli(),
	}), nil
}

// -------------------------------------------------------------------
// GET /api/invites/{code}
// -------------------------------------------------------------------

type resolveInviteResponse struct {
	Code         string `json:"code"`
	GameTitle    string `json:"game_title"`
	PartyCurrent int    `json:"party_current"`
	PartyMax     int    `json:"party_max"`
	Expired      bool   `json:"expired"`
}

func handleGetInvite(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	code := req.PathParameters["code"]
	if code == "" {
		return jsonResponse(400, map[string]string{"error": "missing code"}), nil
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	inv, err := dbClient.GetInvite(ctx, code)
	if err != nil || inv == nil {
		return jsonResponse(404, map[string]string{"error": "invite not found"}), nil
	}

	expired := inv.ExpiresAt > 0 && time.Now().Unix() > inv.ExpiresAt
	if inv.MaxUses > 0 && inv.Uses >= inv.MaxUses {
		expired = true
	}

	saveState, err := dbClient.GetGame(ctx, string(inv.SessionID))
	if err != nil {
		return jsonResponse(404, map[string]string{"error": "session not found"}), nil
	}

	partyCurrent := len(saveState.Players)
	partyMax := saveState.PartySize
	if partyMax == 0 {
		partyMax = 4
	}
	if partyMax > 0 && partyCurrent >= partyMax {
		expired = true
	}

	return jsonResponse(200, resolveInviteResponse{
		Code:         code,
		GameTitle:    saveState.Title,
		PartyCurrent: partyCurrent,
		PartyMax:     partyMax,
		Expired:      expired,
	}), nil
}

// -------------------------------------------------------------------
// POST /api/invites/{code}/join
// -------------------------------------------------------------------

type joinInviteResponse struct {
	SessionID string `json:"session_id"`
}

func handleJoinInvite(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID := req.RequestContext.Authorizer.JWT.Claims["sub"]
	if userID == "" {
		return jsonResponse(401, map[string]string{"error": "unauthorized"}), nil
	}

	// Extract code from path: /api/invites/{code}/join
	code := req.PathParameters["code"]
	if code == "" {
		return jsonResponse(400, map[string]string{"error": "missing code"}), nil
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		return serverError(), nil
	}

	inv, err := dbClient.GetInvite(ctx, code)
	if err != nil || inv == nil {
		return jsonResponse(404, map[string]string{"error": "invite not found"}), nil
	}

	// Check expiry
	if inv.ExpiresAt > 0 && time.Now().Unix() > inv.ExpiresAt {
		return jsonResponse(410, map[string]string{"error": "invite expired"}), nil
	}
	if inv.MaxUses > 0 && inv.Uses >= inv.MaxUses {
		return jsonResponse(410, map[string]string{"error": "invite has reached max uses"}), nil
	}

	sessionID := string(inv.SessionID)
	saveState, err := dbClient.GetGame(ctx, sessionID)
	if err != nil {
		return jsonResponse(404, map[string]string{"error": "session not found"}), nil
	}

	// Check party capacity
	partyMax := saveState.PartySize
	if partyMax == 0 {
		partyMax = 4
	}
	if len(saveState.Players) >= partyMax {
		return jsonResponse(409, map[string]string{"error": "party is full"}), nil
	}

	// Check if already a member
	if _, alreadyIn := saveState.Players[userID]; alreadyIn {
		// Idempotent — already joined, just return session_id
		return jsonResponse(200, joinInviteResponse{SessionID: sessionID}), nil
	}

	// Add the new member: create a character stub and write membership
	g, err := game.FromSaveState(saveState)
	if err != nil {
		return serverError(), nil
	}

	// Stub character — will be filled in on the character creation page
	stub := game.NewCharacter("Adventurer", "")
	g.SetPlayerCharacter(userID, stub)
	g.Version++

	updated := g.ToSaveState(saveState.Narrative, saveState.ChatHistory)
	if err := dbClient.PutGame(ctx, updated); err != nil {
		log.Printf("http-invites join: PutGame: %v", err)
		return serverError(), nil
	}

	if err := dbClient.PutMembership(ctx, db.MembershipRecord{
		UserID:    db.BinaryID(userID),
		SessionID: db.BinaryID(sessionID),
		Role:      "member",
		JoinedAt:  time.Now().UnixMilli(),
	}); err != nil {
		log.Printf("http-invites join: PutMembership (non-fatal): %v", err)
	}

	if err := dbClient.IncrementInviteUses(ctx, code); err != nil {
		log.Printf("http-invites join: IncrementInviteUses (non-fatal): %v", err)
	}

	return jsonResponse(200, joinInviteResponse{SessionID: sessionID}), nil
}

// -------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------

func generateInviteCode(length int) (string, error) {
	b := make([]byte, length)
	n := big.NewInt(int64(len(inviteCodeAlphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		b[i] = inviteCodeAlphabet[idx.Int64()]
	}
	return string(b), nil
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
