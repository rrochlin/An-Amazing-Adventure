package main

import (
	"context"
	"testing"
)

func TestHandlerWorldGen_MissingSessionID(t *testing.T) {
	// Empty event — session ID is empty string, DB lookup will fail
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("BEDROCK_REGION", "us-west-2")

	evt := worldGenEvent{}
	err := handler(context.Background(), evt)
	// Should error at DB layer since session ID is empty
	if err == nil {
		t.Error("expected error for empty session ID")
	}
}

func TestHandlerWorldGen_EventParsed(t *testing.T) {
	// Verify the event struct is correctly shaped
	evt := worldGenEvent{
		SessionID:  "sess-abc-123",
		UserID:     "user-xyz",
		PlayerName: "Aragorn",
	}
	if evt.SessionID != "sess-abc-123" {
		t.Error("session ID not preserved")
	}
	if evt.PlayerName != "Aragorn" {
		t.Error("player name not preserved")
	}
}
