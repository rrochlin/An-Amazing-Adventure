package main

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func makeWSReq(connID string, queryParams map[string]string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		QueryStringParameters: queryParams,
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connID,
		},
	}
}

func TestHandlerConnect_MissingToken(t *testing.T) {
	req := makeWSReq("conn-1", map[string]string{})
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for missing token, got %d", resp.StatusCode)
	}
}

func TestHandlerConnect_MalformedToken(t *testing.T) {
	req := makeWSReq("conn-1", map[string]string{"token": "not-a-jwt"})
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for malformed token, got %d", resp.StatusCode)
	}
}

func TestHandlerConnect_ExpiredToken(t *testing.T) {
	// A real JWT structure but with exp in the past
	// Header: {"alg":"HS256","typ":"JWT"}
	// Payload: {"sub":"user-123","exp":1000000000}  (year 2001 - definitely expired)
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEyMyIsImV4cCI6MTAwMDAwMDAwMH0.signature"
	req := makeWSReq("conn-1", map[string]string{"token": expiredToken})
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for expired token, got %d", resp.StatusCode)
	}
}

func TestHandlerConnect_ValidTokenFormat_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("USER_POOL_ID", "us-west-2_test")
	// Valid JWT structure with future exp
	// Payload: {"sub":"user-abc","exp":9999999999}
	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLWFiYyIsImV4cCI6OTk5OTk5OTk5OX0.signature"
	req := makeWSReq("conn-123", map[string]string{
		"token":  validToken,
		"gameId": "game-uuid",
	})
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Token validates structurally; will fail at DynamoDB with no real credentials
	// Should be 500 (DB failure), not 401 (auth failure)
	if resp.StatusCode == 401 {
		t.Errorf("expected to pass JWT validation, got 401 — JWT parsing failed")
	}
}

// ---- validateCognitoToken unit tests ----

func TestValidateCognitoToken_Empty(t *testing.T) {
	_, err := validateCognitoToken("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestValidateCognitoToken_NotThreeParts(t *testing.T) {
	_, err := validateCognitoToken("only.two")
	if err == nil {
		t.Error("expected error for token with <3 parts")
	}
}

func TestValidateCognitoToken_InvalidBase64Payload(t *testing.T) {
	_, err := validateCognitoToken("header.!!!not-base64!!!.sig")
	if err == nil {
		t.Error("expected error for invalid base64 payload")
	}
}

func TestValidateCognitoToken_MissingSub(t *testing.T) {
	// Payload: {"exp":9999999999} — no sub
	token := "eyJhbGciOiJIUzI1NiJ9.eyJleHAiOjk5OTk5OTk5OTl9.sig"
	_, err := validateCognitoToken(token)
	if err == nil {
		t.Error("expected error for missing sub claim")
	}
}

func TestValidateCognitoToken_ValidStructure(t *testing.T) {
	// Payload: {"sub":"user-abc","exp":9999999999}
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLWFiYyIsImV4cCI6OTk5OTk5OTk5OX0.sig"
	sub, err := validateCognitoToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub != "user-abc" {
		t.Errorf("expected sub=user-abc, got %q", sub)
	}
}
