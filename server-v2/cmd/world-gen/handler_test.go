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
		SessionID:         "sess-abc-123",
		UserID:            "user-xyz",
		PlayerName:        "Aragorn",
		PlayerDescription: "Tall ranger from the north",
		PlayerAge:         "late 30s",
		PlayerBackstory:   "Heir to the throne of Gondor",
		ThemeHint:         "high fantasy epic",
		Preferences:       []string{"combat", "exploration"},
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
	if evt.PlayerDescription != "Tall ranger from the north" {
		t.Errorf("player description not preserved: %q", evt.PlayerDescription)
	}
	if len(evt.Preferences) != 2 || evt.Preferences[0] != "combat" {
		t.Errorf("preferences not preserved: %v", evt.Preferences)
	}
}

func TestHandlerWorldGen_EventParsed_EmptyPlayerName(t *testing.T) {
	// player_name is now optional — verify empty name is preserved
	evt := worldGenEvent{
		SessionID:  "sess-abc-456",
		UserID:     "user-xyz",
		PlayerName: "", // intentionally empty — AI will generate
		ThemeHint:  "cosmic horror",
	}
	if evt.PlayerName != "" {
		t.Errorf("expected empty player name, got: %q", evt.PlayerName)
	}
	if evt.ThemeHint != "cosmic horror" {
		t.Errorf("theme hint not preserved: %q", evt.ThemeHint)
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
