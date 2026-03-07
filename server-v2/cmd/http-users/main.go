// http-users handles /api/users REST routes.
// Sign-up is handled entirely by the Cognito client in the browser (SRP flow).
// This Lambda only handles profile updates that require backend involvement.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cognitoidp "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method

	switch method {
	case "PUT":
		return handleUpdateUser(ctx, req)
	default:
		return jsonResponse(404, map[string]string{"error": "not found"}), nil
	}
}

func handleUpdateUser(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID := req.RequestContext.Authorizer.JWT.Claims["sub"]
	if userID == "" {
		return jsonResponse(401, map[string]string{"error": "unauthorized"}), nil
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonResponse(400, map[string]string{"error": "invalid body"}), nil
	}

	userPoolID := os.Getenv("USER_POOL_ID")
	if userPoolID == "" {
		log.Println("USER_POOL_ID not set")
		return serverError(), nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return serverError(), nil
	}
	client := cognitoidp.NewFromConfig(cfg)

	attrs := []cognitotypes.AttributeType{}
	if body.Email != "" {
		attrs = append(attrs, cognitotypes.AttributeType{
			Name:  aws.String("email"),
			Value: aws.String(body.Email),
		})
	}

	if len(attrs) > 0 {
		_, err = client.AdminUpdateUserAttributes(ctx, &cognitoidp.AdminUpdateUserAttributesInput{
			UserPoolId:     aws.String(userPoolID),
			Username:       aws.String(userID),
			UserAttributes: attrs,
		})
		if err != nil {
			log.Printf("update user %s: %v", userID, err)
			return serverError(), nil
		}
	}

	return jsonResponse(200, map[string]string{"status": "ok"}), nil
}

func jsonResponse(code int, body any) events.APIGatewayV2HTTPResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func serverError() events.APIGatewayV2HTTPResponse {
	return jsonResponse(500, map[string]string{"error": "internal server error"})
}

func main() {
	lambda.Start(handler)
}
