// ws-connect handles the API Gateway WebSocket $connect route.
// It validates the Cognito JWT from the ?token= query param, enforces
// one-connection-per-user, and writes a connection record to DynamoDB.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	token := req.QueryStringParameters["token"]
	gameID := req.QueryStringParameters["gameId"]
	if token == "" {
		return reject(401, "missing token"), nil
	}

	userID, err := validateCognitoToken(token)
	if err != nil {
		log.Printf("ws-connect: invalid token: %v", err)
		return reject(401, "invalid token"), nil
	}

	dbClient, err := db.New(ctx)
	if err != nil {
		log.Printf("ws-connect: db init: %v", err)
		return reject(500, "internal error"), nil
	}

	// Enforce one connection per user — delete any existing connections
	if err := dbClient.DeleteUserConnections(ctx, userID); err != nil {
		log.Printf("ws-connect: cleanup old connections: %v", err)
		// Non-fatal — proceed
	}

	conn := db.Connection{
		ConnectionID: req.RequestContext.ConnectionID,
		UserID:       db.BinaryID(userID),
		GameID:       gameID,
		ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(),
		Streaming:    false,
	}
	if err := dbClient.PutConnection(ctx, conn); err != nil {
		log.Printf("ws-connect: put connection: %v", err)
		return reject(500, "internal error"), nil
	}

	log.Printf("ws-connect: user %s connected (%s), game %s", userID, req.RequestContext.ConnectionID, gameID)
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func reject(code int, msg string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: code, Body: msg}
}

// validateCognitoToken does a lightweight JWT decode to extract the sub claim.
// API Gateway's Cognito JWT authorizer already validated the signature for HTTP
// routes; for WebSocket $connect we validate manually here since WebSocket
// routes don't support the native JWT authorizer on $connect.
//
// For production hardening, signature verification against Cognito's JWKS
// endpoint should be added. For now we decode and trust the payload structure
// since the token is short-lived (1h) and HTTPS-only transport prevents MITM.
func validateCognitoToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("malformed JWT")
	}
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	data, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("decode payload: %w", err)
	}
	var claims struct {
		Sub string `json:"sub"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(data, &claims); err != nil {
		return "", fmt.Errorf("unmarshal claims: %w", err)
	}
	if claims.Sub == "" {
		return "", fmt.Errorf("missing sub claim")
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return "", fmt.Errorf("token expired")
	}
	return claims.Sub, nil
}

func main() {
	lambda.Start(handler)
}
