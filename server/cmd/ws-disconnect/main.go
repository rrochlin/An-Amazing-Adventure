// ws-disconnect handles the API Gateway WebSocket $disconnect route.
// It deletes the connection record from DynamoDB.
package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
)

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connID := req.RequestContext.ConnectionID
	log.Printf("ws-disconnect: %s", connID)

	dbClient, err := db.New(ctx)
	if err != nil {
		log.Printf("ws-disconnect: db init: %v", err)
		// Always return 200 — API GW ignores disconnect errors
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	if err := dbClient.DeleteConnection(ctx, connID); err != nil {
		log.Printf("ws-disconnect: delete connection %s: %v", connID, err)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(handler)
}
