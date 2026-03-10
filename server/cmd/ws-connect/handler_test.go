package main

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// assertPanicsWithEnvAbsent runs fn with the given env var unset and asserts
// that it panics with a message containing the var name. This catches missing
// env var configuration before deployment.
func assertPanicsWithEnvAbsent(t *testing.T, envVar string, fn func()) {
	t.Helper()
	t.Setenv(envVar, "") // unset for this test; restored after
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic for missing %s, but handler did not panic", envVar)
			return
		}
		msg := ""
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(msg, envVar) {
			t.Errorf("panic message %q does not mention %s", msg, envVar)
		}
	}()
	fn()
}

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
	req := makeWSReq("conn-1", map[string]string{"token": "not-a-jwt", "gameId": "game-uuid"})
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
	req := makeWSReq("conn-1", map[string]string{"token": expiredToken, "gameId": "game-uuid"})
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

// ---- Required env var tests ----
// ws-connect now reads the sessions table (for auth) before writing to connections.
// Both SESSIONS_TABLE and CONNECTIONS_TABLE must be set.

func TestHandlerConnect_MissingSESSIONS_TABLE_Panics(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("USER_POOL_ID", "us-west-2_test")
	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLWFiYyIsImV4cCI6OTk5OTk5OTk5OX0.signature"
	req := makeWSReq("conn-1", map[string]string{"token": validToken, "gameId": "g"})
	assertPanicsWithEnvAbsent(t, "SESSIONS_TABLE", func() {
		handler(context.Background(), req) //nolint:errcheck
	})
}

// TestHandlerConnect_MissingGameId verifies we reject connections without a gameId.
func TestHandlerConnect_MissingGameId(t *testing.T) {
	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLWFiYyIsImV4cCI6OTk5OTk5OTk5OX0.signature"
	req := makeWSReq("conn-1", map[string]string{"token": validToken}) // no gameId
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for missing gameId, got %d", resp.StatusCode)
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
