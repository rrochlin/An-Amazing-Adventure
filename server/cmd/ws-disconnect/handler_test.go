package main

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
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

func makeWSReq(connID string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connID,
		},
	}
}

func TestHandlerDisconnect_AlwaysReturns200(t *testing.T) {
	// Disconnect must always return 200 — API GW ignores the response
	// but a non-200 would cause unnecessary retries.
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	req := makeWSReq("conn-to-clean-up")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Even with no real DB, the handler swallows errors and returns 200
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 (disconnect always returns 200), got %d", resp.StatusCode)
	}
}

// ---- Required env var tests ----
// ws-disconnect requires: CONNECTIONS_TABLE

func TestHandlerDisconnect_MissingCONNECTIONS_TABLE_Panics(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	assertPanicsWithEnvAbsent(t, "CONNECTIONS_TABLE", func() {
		handler(context.Background(), makeWSReq("conn-1")) //nolint:errcheck
	})
}

func TestHandlerDisconnect_EmptyConnID(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	req := makeWSReq("")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 even for empty connID, got %d", resp.StatusCode)
	}
}
