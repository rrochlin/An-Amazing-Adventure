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

// makeHTTPReq builds a minimal APIGatewayV2HTTPRequest with Cognito sub claim.
func makeHTTPReq(method, path string, body string, sub string, pathParams map[string]string) events.APIGatewayV2HTTPRequest {
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
		PathParameters: pathParams,
	}
}

// ---- Auth guard ----

func TestHandlerRejects_NoSub(t *testing.T) {
	req := makeHTTPReq("GET", "/api/games", "", "", nil)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 with no sub, got %d", resp.StatusCode)
	}
}

// ---- Route dispatch ----

func TestHandlerUnknownRoute_404(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-table")
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("WORLD_GEN_ARN", "")
	req := makeHTTPReq("GET", "/api/unknown", "", "user-123", nil)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for unknown route, got %d", resp.StatusCode)
	}
}

// ---- GET /api/games ----

func TestHandlerListGames_EmptyList(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-table")
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("USERS_TABLE", "test-users")
	t.Setenv("WORLD_GEN_ARN", "")
	// Without a real DynamoDB table this will error at the DB layer —
	// we assert the handler routes correctly and returns a structured error.
	req := makeHTTPReq("GET", "/api/games", "", "user-123", nil)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// 500 expected because env is missing real DB credentials, not 401/404
	if resp.StatusCode == 401 || resp.StatusCode == 404 {
		t.Errorf("expected non-auth error, got %d — routing failed", resp.StatusCode)
	}
	// Response must be valid JSON
	var body map[string]any
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Errorf("response body is not valid JSON: %s", resp.Body)
	}
}

// ---- POST /api/games ----

func TestHandlerCreateGame_EmptyBody_AcceptsAIGenerated(t *testing.T) {
	// player_name is now optional — empty body should reach the DB layer (not 400)
	t.Setenv("SESSIONS_TABLE", "test-table")
	t.Setenv("USERS_TABLE", "test-users")
	t.Setenv("WORLD_GEN_ARN", "")
	req := makeHTTPReq("POST", "/api/games", `{}`, "user-123", nil)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Should NOT be a 400 (bad request) — will fail at DB layer (500) without real credentials
	if resp.StatusCode == 400 {
		t.Errorf("empty body should not return 400 now that player_name is optional, got %d\nbody: %s", resp.StatusCode, resp.Body)
	}
}

func TestHandlerCreateGame_WithAllParams(t *testing.T) {
	// Verify the handler accepts all new optional creation params
	t.Setenv("SESSIONS_TABLE", "test-table")
	t.Setenv("USERS_TABLE", "test-users")
	t.Setenv("WORLD_GEN_ARN", "")
	body := `{
		"player_name": "Aria",
		"player_age": "mid 20s",
		"player_description": "A nimble rogue with sharp eyes",
		"player_backstory": "Raised by thieves, seeking redemption",
		"theme_hint": "gritty noir",
		"preferences": ["stealth", "mystery"]
	}`
	req := makeHTTPReq("POST", "/api/games", body, "user-123", nil)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Should reach DB layer, not return 400
	if resp.StatusCode == 400 {
		t.Errorf("valid body with all params should not return 400, got %d\nbody: %s", resp.StatusCode, resp.Body)
	}
}

func TestHandlerCreateGame_InvalidJSON(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-table")
	t.Setenv("USERS_TABLE", "test-users")
	t.Setenv("WORLD_GEN_ARN", "")
	req := makeHTTPReq("POST", "/api/games", `not-json`, "user-123", nil)
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

// ---- DELETE /api/games/{uuid} ----

func TestHandlerDeleteGame_MissingUUID(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-table")
	req := makeHTTPReq("DELETE", "/api/games/", "user-123", "user-123", map[string]string{})
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Without uuid in path params, routing should still attempt delete and fail gracefully
	if resp.StatusCode == 0 {
		t.Error("expected a non-zero status code")
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Errorf("response is not valid JSON: %s", resp.Body)
	}
}

// ---- Response format helpers ----

func TestJSONResponse_ContentType(t *testing.T) {
	resp := jsonResponse(200, map[string]string{"foo": "bar"})
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", resp.Headers["Content-Type"])
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Errorf("body is not valid JSON: %s", resp.Body)
	}
	if body["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", body)
	}
}

func TestServerError_Returns500(t *testing.T) {
	resp := serverError()
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestMatchesGamePath(t *testing.T) {
	cases := []struct {
		path  string
		match bool
	}{
		{"/api/games/abc-123", true},       // UUID segment present — match
		{"/api/games/abc-123/extra", true}, // deeper path — match
		{"/api/games/", false},             // trailing slash only — no UUID, no match
		{"/api/games", false},              // base path — no UUID, no match
		{"/api/other/uuid", false},         // wrong prefix — no match
	}
	for _, c := range cases {
		got := matchesGamePath(c.path)
		if got != c.match {
			t.Errorf("matchesGamePath(%q) = %v, want %v", c.path, got, c.match)
		}
	}
}

// ---- Required env var tests ----
// http-games requires: SESSIONS_TABLE, USERS_TABLE
// (CONNECTIONS_TABLE is NOT required — http-games never touches connections)

func TestHandlerGames_MissingSESSIONS_TABLE_Panics(t *testing.T) {
	req := makeHTTPReq("GET", "/api/games", "", "user-sub-123", nil)
	assertPanicsWithEnvAbsent(t, "SESSIONS_TABLE", func() {
		handler(context.Background(), req) //nolint:errcheck
	})
}

func TestHandlerGames_NoCONNECTIONS_TABLE_DoesNotPanic(t *testing.T) {
	// http-games must NOT panic when CONNECTIONS_TABLE is absent —
	// it never uses the connections table.
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("CONNECTIONS_TABLE", "") // explicitly absent
	req := makeHTTPReq("GET", "/api/games", "", "user-sub-123", nil)
	// Should reach DynamoDB (and fail with a credential/network error), not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("http-games panicked with CONNECTIONS_TABLE absent: %v", r)
		}
	}()
	handler(context.Background(), req) //nolint:errcheck
}
