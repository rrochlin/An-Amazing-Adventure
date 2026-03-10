// http-admin handles admin management API routes:
//
//	GET  /api/admin/users           — list all users with Cognito email enrichment
//	PUT  /api/admin/users/{userId}  — update role, AI access, limits, notes
//	GET  /api/admin/stats           — aggregate user and token stats
//
// Auth is enforced at two layers:
//  1. API Gateway JWT authorizer — requires valid Cognito token
//  2. Lambda-level admin group check — requires cognito:groups to contain "admin"
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cognitoidp "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
)

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// Enforce admin group membership — defense in depth beyond API Gateway authorizer
	if !hasGroup(req.RequestContext.Authorizer.JWT.Claims, "admin") {
		return jsonResponse(403, map[string]string{"error": "forbidden"}), nil
	}

	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path

	dbClient, err := db.New(ctx)
	if err != nil {
		log.Printf("http-admin: db init: %v", err)
		return serverError(), nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("http-admin: aws config: %v", err)
		return serverError(), nil
	}
	cognitoClient := cognitoidp.NewFromConfig(cfg)
	userPoolID := os.Getenv("USER_POOL_ID")

	switch {
	case method == "GET" && path == "/api/admin/users":
		return handleListUsers(ctx, dbClient, cognitoClient, userPoolID)
	case method == "PUT" && strings.HasPrefix(path, "/api/admin/users/"):
		userID := req.PathParameters["userId"]
		return handleUpdateUser(ctx, req, dbClient, cognitoClient, userPoolID, userID)
	case method == "GET" && path == "/api/admin/stats":
		return handleStats(ctx, dbClient)
	default:
		return jsonResponse(404, map[string]string{"error": "not_found"}), nil
	}
}

// AdminUserView is the JSON shape returned to the admin panel per user.
type AdminUserView struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	AIEnabled   bool   `json:"ai_enabled"`
	TokenLimit  int    `json:"token_limit"`
	TokensUsed  int    `json:"tokens_used"`
	GamesLimit  int    `json:"games_limit"`
	BillingMode string `json:"billing_mode"`
	Notes       string `json:"notes,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

func handleListUsers(
	ctx context.Context,
	dbClient *db.Client,
	cognitoClient *cognitoidp.Client,
	userPoolID string,
) (events.APIGatewayV2HTTPResponse, error) {
	records, err := dbClient.ListUsers(ctx)
	if err != nil {
		log.Printf("http-admin ListUsers: %v", err)
		return serverError(), nil
	}

	views := make([]AdminUserView, 0, len(records))
	for _, r := range records {
		email := fetchEmail(ctx, cognitoClient, userPoolID, string(r.UserID))
		views = append(views, AdminUserView{
			UserID:      string(r.UserID),
			Email:       email,
			Role:        r.Role,
			AIEnabled:   r.AIEnabled,
			TokenLimit:  r.TokenLimit,
			TokensUsed:  r.TokensUsed,
			GamesLimit:  r.GamesLimit,
			BillingMode: r.BillingMode,
			Notes:       r.Notes,
			CreatedAt:   r.CreatedAt,
		})
	}
	return jsonResponse(200, views), nil
}

type updateUserRequest struct {
	Role       string `json:"role"` // "admin" | "user" | "restricted"
	AIEnabled  bool   `json:"ai_enabled"`
	TokenLimit int    `json:"token_limit"` // 0 = unlimited
	GamesLimit int    `json:"games_limit"` // 0 = unlimited
	Notes      string `json:"notes"`
}

func handleUpdateUser(
	ctx context.Context,
	req events.APIGatewayV2HTTPRequest,
	dbClient *db.Client,
	cognitoClient *cognitoidp.Client,
	userPoolID string,
	userID string,
) (events.APIGatewayV2HTTPResponse, error) {
	if userID == "" {
		return jsonResponse(400, map[string]string{"error": "missing userId"}), nil
	}

	var body updateUserRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonResponse(400, map[string]string{"error": "invalid_body"}), nil
	}

	existing, err := dbClient.GetUser(ctx, userID)
	if err != nil || existing == nil {
		return jsonResponse(404, map[string]string{"error": "user_not_found"}), nil
	}

	existing.Role = body.Role
	existing.AIEnabled = body.AIEnabled
	existing.TokenLimit = body.TokenLimit
	existing.GamesLimit = body.GamesLimit
	existing.Notes = body.Notes

	if err := dbClient.UpdateUser(ctx, userID, *existing); err != nil {
		log.Printf("http-admin UpdateUser %s: %v", userID, err)
		return serverError(), nil
	}

	// Sync Cognito group membership to match the new role
	if err := syncCognitoGroups(ctx, cognitoClient, userPoolID, userID, body.Role); err != nil {
		// Non-fatal: DB is source of truth; Cognito groups are informational
		log.Printf("http-admin syncCognitoGroups %s (non-fatal): %v", userID, err)
	}

	return jsonResponse(200, map[string]string{"status": "ok"}), nil
}

// syncCognitoGroups ensures the user is in the correct Cognito group for their role:
//
//	admin      → [admin, user]
//	user       → [user]
//	restricted → [restricted]
func syncCognitoGroups(
	ctx context.Context,
	cognitoClient *cognitoidp.Client,
	userPoolID, userID, role string,
) error {
	allGroups := []string{"admin", "user", "restricted"}
	targetGroups := map[string]bool{}
	switch role {
	case "admin":
		targetGroups["admin"] = true
		targetGroups["user"] = true
	case "user":
		targetGroups["user"] = true
	default:
		targetGroups["restricted"] = true
	}

	for _, g := range allGroups {
		g := g
		if targetGroups[g] {
			if _, err := cognitoClient.AdminAddUserToGroup(ctx, &cognitoidp.AdminAddUserToGroupInput{
				UserPoolId: aws.String(userPoolID),
				Username:   aws.String(userID),
				GroupName:  aws.String(g),
			}); err != nil {
				log.Printf("AdminAddUserToGroup %s → %s: %v", userID, g, err)
			}
		} else {
			if _, err := cognitoClient.AdminRemoveUserFromGroup(ctx, &cognitoidp.AdminRemoveUserFromGroupInput{
				UserPoolId: aws.String(userPoolID),
				Username:   aws.String(userID),
				GroupName:  aws.String(g),
			}); err != nil {
				// Non-fatal: user may not be in the group
				log.Printf("AdminRemoveUserFromGroup %s ← %s (non-fatal): %v", userID, g, err)
			}
		}
	}
	return nil
}

type adminStats struct {
	TotalUsers      int `json:"total_users"`
	AdminUsers      int `json:"admin_users"`
	ApprovedUsers   int `json:"approved_users"`
	RestrictedUsers int `json:"restricted_users"`
	TotalTokensUsed int `json:"total_tokens_used"`
}

func handleStats(ctx context.Context, dbClient *db.Client) (events.APIGatewayV2HTTPResponse, error) {
	records, err := dbClient.ListUsers(ctx)
	if err != nil {
		return serverError(), nil
	}
	stats := adminStats{}
	for _, r := range records {
		stats.TotalUsers++
		stats.TotalTokensUsed += r.TokensUsed
		switch r.Role {
		case "admin":
			stats.AdminUsers++
		case "user":
			stats.ApprovedUsers++
		default:
			stats.RestrictedUsers++
		}
	}
	return jsonResponse(200, stats), nil
}

func fetchEmail(ctx context.Context, cognitoClient *cognitoidp.Client, userPoolID, userID string) string {
	out, err := cognitoClient.AdminGetUser(ctx, &cognitoidp.AdminGetUserInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(userID),
	})
	if err != nil {
		return ""
	}
	for _, attr := range out.UserAttributes {
		if aws.ToString(attr.Name) == "email" {
			return aws.ToString(attr.Value)
		}
	}
	return ""
}

func hasGroup(claims map[string]string, group string) bool {
	return strings.Contains(claims["cognito:groups"], group)
}

func jsonResponse(status int, body any) events.APIGatewayV2HTTPResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func serverError() events.APIGatewayV2HTTPResponse {
	return jsonResponse(500, map[string]string{"error": "internal_server_error"})
}

func main() {
	lambda.Start(handler)
}
