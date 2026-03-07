package main

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

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
