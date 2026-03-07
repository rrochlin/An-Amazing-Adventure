package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func makeWSChatReq(connID, body string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		Body: body,
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connID,
		},
	}
}

func TestHandlerChat_InvalidJSON(t *testing.T) {
	req := makeWSChatReq("conn-1", "not-json")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestHandlerChat_EmptyContent(t *testing.T) {
	body, _ := json.Marshal(chatRequest{Action: "chat", Content: ""})
	req := makeWSChatReq("conn-1", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for empty content, got %d", resp.StatusCode)
	}
}

func TestHandlerChat_ValidMessage_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")
	t.Setenv("BEDROCK_REGION", "us-west-2")

	body, _ := json.Marshal(chatRequest{Action: "chat", Content: "Go north"})
	req := makeWSChatReq("conn-abc", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Will fail at DynamoDB GetConnection — should be 410 (Gone/not found) or 500, not 400
	if resp.StatusCode == 400 {
		t.Errorf("routing/parse failure (400) — expected to reach DB layer")
	}
}

func TestChatRequest_Parsed(t *testing.T) {
	var req chatRequest
	body := `{"action":"chat","content":"Hello world"}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Content != "Hello world" {
		t.Errorf("expected content 'Hello world', got %q", req.Content)
	}
	if req.Action != "chat" {
		t.Errorf("expected action 'chat', got %q", req.Action)
	}
}
