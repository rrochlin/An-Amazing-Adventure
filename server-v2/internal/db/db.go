// Package db provides DynamoDB access for the game.
// All table names come from environment variables injected by Lambda.
package db

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Client wraps the DynamoDB client with table name config.
type Client struct {
	ddb              *dynamodb.Client
	sessionsTable    string
	connectionsTable string
}

// New creates a Client from the current AWS environment.
// SESSIONS_TABLE is required. CONNECTIONS_TABLE is optional at construction
// time — it is only needed by WebSocket Lambdas; HTTP-only Lambdas omit it.
// A panic is deferred until a connections method is actually called without it.
func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &Client{
		ddb:              dynamodb.NewFromConfig(cfg),
		sessionsTable:    mustEnv("SESSIONS_TABLE"),
		connectionsTable: os.Getenv("CONNECTIONS_TABLE"), // optional; checked at use
	}, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}

// requireConnectionsTable panics with a clear message if CONNECTIONS_TABLE was
// not set. Called at the top of every method that touches that table.
func (c *Client) requireConnectionsTable() {
	if c.connectionsTable == "" {
		panic("required env var CONNECTIONS_TABLE is not set")
	}
}

// -------------------------------------------------------------------
// Game sessions
// -------------------------------------------------------------------

// PutGame writes a SaveState to DynamoDB using optimistic locking.
// If the current version in the DB doesn't match state.Version, the write
// is rejected with a ConditionalCheckFailedException — caller should retry.
func (c *Client) PutGame(ctx context.Context, state game.SaveState) error {
	item, err := attributevalue.MarshalMap(state)
	if err != nil {
		return fmt.Errorf("marshal save state: %w", err)
	}

	var condExpr *string
	var exprAttrVals map[string]types.AttributeValue

	if state.Version > 0 {
		// Optimistic lock: only write if the stored version matches
		prevVersion, _ := attributevalue.Marshal(state.Version - 1)
		condExpr = aws.String("version = :prev")
		exprAttrVals = map[string]types.AttributeValue{":prev": prevVersion}
	} else {
		// First write: item must not exist yet
		condExpr = aws.String("attribute_not_exists(session_id)")
	}

	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(c.sessionsTable),
		Item:                      item,
		ConditionExpression:       condExpr,
		ExpressionAttributeValues: exprAttrVals,
	})
	if err != nil {
		return fmt.Errorf("put game: %w", err)
	}
	return nil
}

// GetGame loads and deserialises a full SaveState by session UUID string.
func (c *Client) GetGame(ctx context.Context, sessionID string) (game.SaveState, error) {
	key, err := sessionKey(sessionID)
	if err != nil {
		return game.SaveState{}, err
	}
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.sessionsTable),
		Key:       key,
	})
	if err != nil {
		return game.SaveState{}, fmt.Errorf("get game: %w", err)
	}
	if out.Item == nil {
		return game.SaveState{}, fmt.Errorf("game not found: %s", sessionID)
	}
	var state game.SaveState
	if err := attributevalue.UnmarshalMap(out.Item, &state); err != nil {
		return game.SaveState{}, fmt.Errorf("unmarshal game: %w", err)
	}
	return state, nil
}

// GetGameReady does a projection-only read to check the ready flag cheaply.
func (c *Client) GetGameReady(ctx context.Context, sessionID string) (bool, error) {
	key, err := sessionKey(sessionID)
	if err != nil {
		return false, err
	}
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:            aws.String(c.sessionsTable),
		Key:                  key,
		ProjectionExpression: aws.String("#r"),
		ExpressionAttributeNames: map[string]string{
			"#r": "ready",
		},
	})
	if err != nil {
		return false, fmt.Errorf("get game ready: %w", err)
	}
	if out.Item == nil {
		return false, fmt.Errorf("game not found: %s", sessionID)
	}
	var partial struct {
		Ready bool `dynamodbav:"ready"`
	}
	if err := attributevalue.UnmarshalMap(out.Item, &partial); err != nil {
		return false, err
	}
	return partial.Ready, nil
}

// ListGames returns all SaveState summaries for a given user ID.
func (c *Client) ListGames(ctx context.Context, userID string) ([]game.SaveState, error) {
	keyEx := expression.Key("user_id").Equal(expression.Value(userID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, err
	}
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(c.sessionsTable),
		IndexName:                 aws.String("user-sessions-index"),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	var saves []game.SaveState
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &saves); err != nil {
		return nil, err
	}
	return saves, nil
}

// DeleteGame removes a session record. Caller must verify ownership first.
func (c *Client) DeleteGame(ctx context.Context, sessionID, userID string) error {
	key, err := sessionKey(sessionID)
	if err != nil {
		return err
	}
	// Condition: only delete if the user_id matches (ownership check)
	userVal, _ := attributevalue.Marshal(userID)
	_, err = c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:           aws.String(c.sessionsTable),
		Key:                 key,
		ConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": userVal,
		},
	})
	if err != nil {
		return fmt.Errorf("delete game: %w", err)
	}
	return nil
}

func sessionKey(sessionID string) (map[string]types.AttributeValue, error) {
	v, err := attributevalue.Marshal(sessionID)
	if err != nil {
		return nil, err
	}
	return map[string]types.AttributeValue{"session_id": v}, nil
}

// -------------------------------------------------------------------
// WebSocket connections
// -------------------------------------------------------------------

// Connection represents a live WebSocket connection record.
type Connection struct {
	ConnectionID string `dynamodbav:"connection_id"`
	UserID       string `dynamodbav:"user_id"`
	GameID       string `dynamodbav:"game_id"`
	ExpiresAt    int64  `dynamodbav:"expires_at"` // Unix epoch seconds; TTL field
	Streaming    bool   `dynamodbav:"streaming"`  // true while AI is generating
}

// PutConnection writes or replaces a connection record.
// ExpiresAt should be ~24h from now so DynamoDB TTL auto-cleans stale records.
func (c *Client) PutConnection(ctx context.Context, conn Connection) error {
	c.requireConnectionsTable()
	item, err := attributevalue.MarshalMap(conn)
	if err != nil {
		return err
	}
	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.connectionsTable),
		Item:      item,
	})
	return err
}

// GetConnection retrieves a connection by its API Gateway connection ID.
func (c *Client) GetConnection(ctx context.Context, connectionID string) (Connection, error) {
	c.requireConnectionsTable()
	v, _ := attributevalue.Marshal(connectionID)
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.connectionsTable),
		Key:       map[string]types.AttributeValue{"connection_id": v},
	})
	if err != nil {
		return Connection{}, err
	}
	if out.Item == nil {
		return Connection{}, fmt.Errorf("connection not found: %s", connectionID)
	}
	var conn Connection
	if err := attributevalue.UnmarshalMap(out.Item, &conn); err != nil {
		return Connection{}, err
	}
	return conn, nil
}

// DeleteConnection removes a connection record on disconnect.
func (c *Client) DeleteConnection(ctx context.Context, connectionID string) error {
	c.requireConnectionsTable()
	v, _ := attributevalue.Marshal(connectionID)
	_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(c.connectionsTable),
		Key:       map[string]types.AttributeValue{"connection_id": v},
	})
	return err
}

// DeleteUserConnections removes all connection records for a given user ID.
// Called on new $connect to enforce one-connection-per-user.
func (c *Client) DeleteUserConnections(ctx context.Context, userID string) error {
	c.requireConnectionsTable()
	userVal, _ := attributevalue.Marshal(userID)
	keyEx := expression.Key("user_id").Equal(expression.Value(userID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return err
	}
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(c.connectionsTable),
		IndexName:                 aws.String("user-connections-index"),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})
	if err != nil {
		return err
	}
	for _, item := range out.Items {
		connIDAttr, ok := item["connection_id"]
		if !ok {
			continue
		}
		_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(c.connectionsTable),
			Key: map[string]types.AttributeValue{
				"connection_id": connIDAttr,
			},
		})
		if err != nil {
			return err
		}
	}
	_ = userVal
	return nil
}

// SetStreaming atomically sets the streaming flag on a connection record.
func (c *Client) SetStreaming(ctx context.Context, connectionID string, streaming bool) error {
	c.requireConnectionsTable()
	v, _ := attributevalue.Marshal(connectionID)
	sv, _ := attributevalue.Marshal(streaming)
	update := expression.Set(expression.Name("streaming"), expression.Value(streaming))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return err
	}
	_, err = c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(c.connectionsTable),
		Key:                       map[string]types.AttributeValue{"connection_id": v},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	_ = sv
	return err
}
