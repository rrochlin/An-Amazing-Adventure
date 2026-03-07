package main

import (
	"context"
	"encoding/json"
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

func makeActionReq(connID, body string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		Body: body,
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connID,
		},
	}
}

func TestHandlerAction_InvalidJSON(t *testing.T) {
	req := makeActionReq("conn-1", "bad-json")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestHandlerAction_ValidBody_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")

	body, _ := json.Marshal(actionRequest{
		Action:    "game_action",
		SubAction: "move",
		Payload:   "north",
	})
	req := makeActionReq("conn-abc", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Should fail at DB layer (410 Gone or 500), not at parse layer (400)
	if resp.StatusCode == 400 {
		t.Errorf("routing/parse failure — expected to reach DB layer, got 400")
	}
}

func TestActionRequest_SubActions(t *testing.T) {
	cases := []struct {
		subAction string
		payload   string
	}{
		{"move", "north"},
		{"pick_up", "Rusty Dagger"},
		{"drop", "Heavy Shield"},
		{"equip", "Iron Helm"},
		{"unequip", "head"},
	}
	for _, c := range cases {
		body, _ := json.Marshal(actionRequest{
			Action:    "game_action",
			SubAction: c.subAction,
			Payload:   c.payload,
		})
		var parsed actionRequest
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("failed to parse action %q: %v", c.subAction, err)
		}
		if parsed.SubAction != c.subAction {
			t.Errorf("expected sub_action=%q, got %q", c.subAction, parsed.SubAction)
		}
		if parsed.Payload != c.payload {
			t.Errorf("expected payload=%q, got %q", c.payload, parsed.Payload)
		}
	}
}

// ---- Required env var tests ----
// ws-game-action calls GetConnection first, so CONNECTIONS_TABLE panics immediately.
// SESSIONS_TABLE is required later (GetGame) but unreachable without real DynamoDB.
// Both vars are set in Terraform — see modules/lambdas/main.tf.

func TestHandlerAction_MissingCONNECTIONS_TABLE_Panics(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")
	body, _ := json.Marshal(actionRequest{Action: "game_action", SubAction: "move", Payload: "north"})
	assertPanicsWithEnvAbsent(t, "CONNECTIONS_TABLE", func() {
		handler(context.Background(), makeActionReq("conn-1", string(body))) //nolint:errcheck
	})
}

func TestHandlerAction_Equip_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")

	body, _ := json.Marshal(actionRequest{
		Action:    "game_action",
		SubAction: "equip",
		Payload:   "Iron Helm",
	})
	req := makeActionReq("conn-equip", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode == 400 {
		t.Errorf("parse failure — expected to reach DB layer, got 400")
	}
}

func TestHandlerAction_Unequip_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")

	body, _ := json.Marshal(actionRequest{
		Action:    "game_action",
		SubAction: "unequip",
		Payload:   "head",
	})
	req := makeActionReq("conn-unequip", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode == 400 {
		t.Errorf("parse failure — expected to reach DB layer, got 400")
	}
}
