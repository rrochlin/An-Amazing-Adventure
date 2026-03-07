package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func makeReq(method, path, body, sub string) events.APIGatewayV2HTTPRequest {
	claims := map[string]string{}
	if sub != "" {
		claims["sub"] = sub
	}
	return events.APIGatewayV2HTTPRequest{
		Body: body,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
				Path:   path,
			},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
					Claims: claims,
				},
			},
		},
	}
}

func TestHandlerUnknownMethod_404(t *testing.T) {
	req := makeReq("GET", "/api/users", "", "user-123")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for GET /api/users, got %d", resp.StatusCode)
	}
}

func TestHandlerUpdateUser_NoAuth(t *testing.T) {
	t.Setenv("USER_POOL_ID", "us-west-2_test")
	req := makeReq("PUT", "/api/users", `{"email":"test@example.com"}`, "")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
	}
}

func TestHandlerUpdateUser_InvalidJSON(t *testing.T) {
	t.Setenv("USER_POOL_ID", "us-west-2_test")
	req := makeReq("PUT", "/api/users", `not-json`, "user-123")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestHandlerUpdateUser_ValidRequest_ReachesDB(t *testing.T) {
	t.Setenv("USER_POOL_ID", "us-west-2_test")
	req := makeReq("PUT", "/api/users", `{"email":"newemail@example.com"}`, "user-sub-123")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Will fail at Cognito call with no real credentials — should be 500, not 401/400
	if resp.StatusCode == 401 || resp.StatusCode == 400 {
		t.Errorf("routing/auth failed, got %d — expected to reach Cognito call", resp.StatusCode)
	}
	// Must be valid JSON
	var body map[string]any
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Errorf("response is not valid JSON: %s", resp.Body)
	}
}

func TestJSONResponse_Format(t *testing.T) {
	resp := jsonResponse(201, map[string]string{"status": "created"})
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type json, got %q", resp.Headers["Content-Type"])
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Errorf("body not JSON: %v", err)
	}
}
