package main

import (
	"context"
	"strings"
	"testing"
)

func assertPanicsWithEnvAbsent(t *testing.T, envVar string, fn func()) {
	t.Helper()
	t.Setenv(envVar, "")
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

func TestHandlerWorldGen_MissingSessionID(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("BEDROCK_REGION", "us-west-2")
	// WEBSOCKET_API_ENDPOINT intentionally absent — WS push is best-effort

	evt := worldGenEvent{}
	err := handler(context.Background(), evt)
	// Should error at DB layer since session ID is empty
	if err == nil {
		t.Error("expected error for empty session ID")
	}
}

func TestHandlerWorldGen_EventParsed(t *testing.T) {
	evt := worldGenEvent{
		SessionID:  "sess-abc-123",
		UserID:     "user-xyz",
		PlayerName: "Aragorn",
	}
	if evt.SessionID != "sess-abc-123" {
		t.Errorf("session ID not preserved: %q", evt.SessionID)
	}
	if evt.PlayerName != "Aragorn" {
		t.Errorf("player name not preserved: %q", evt.PlayerName)
	}
	if evt.UserID != "user-xyz" {
		t.Errorf("user ID not preserved: %q", evt.UserID)
	}
}

// ---- Required env var tests ----
// world-gen requires: SESSIONS_TABLE
// CONNECTIONS_TABLE and WEBSOCKET_API_ENDPOINT are optional (WS push is best-effort)

func TestHandlerWorldGen_MissingSESSIONS_TABLE_Panics(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("BEDROCK_REGION", "us-west-2")
	evt := worldGenEvent{SessionID: "sess-1", UserID: "user-1", PlayerName: "Frodo"}
	assertPanicsWithEnvAbsent(t, "SESSIONS_TABLE", func() {
		handler(context.Background(), evt) //nolint:errcheck
	})
}

func TestHandlerWorldGen_NoWSEndpoint_DoesNotPanic(t *testing.T) {
	// When WEBSOCKET_API_ENDPOINT is absent, world-gen must still reach
	// the DB layer gracefully (fail on DynamoDB, not on WS setup).
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("BEDROCK_REGION", "us-west-2")
	// No WEBSOCKET_API_ENDPOINT set

	evt := worldGenEvent{SessionID: "no-ws-session", UserID: "user-1", PlayerName: "Gimli"}
	err := handler(context.Background(), evt)
	// DB lookup will fail (no real DynamoDB), but we must not panic
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}
