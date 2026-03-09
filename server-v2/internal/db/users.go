package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// UserRecord is the per-user RBAC and quota record stored in the users table.
type UserRecord struct {
	UserID      BinaryID `dynamodbav:"user_id"`
	Role        string   `dynamodbav:"role"` // "admin" | "user" | "restricted"
	AIEnabled   bool     `dynamodbav:"ai_enabled"`
	TokenLimit  int      `dynamodbav:"token_limit"` // 0 = unlimited
	TokensUsed  int      `dynamodbav:"tokens_used"`
	GamesLimit  int      `dynamodbav:"games_limit"`  // 0 = unlimited
	BillingMode string   `dynamodbav:"billing_mode"` // "admin_granted" | "own_key" | "subscription"
	APIKeyHash  string   `dynamodbav:"api_key_hash,omitempty"`
	CreatedAt   int64    `dynamodbav:"created_at"`
	UpdatedAt   int64    `dynamodbav:"updated_at"`
	Notes       string   `dynamodbav:"notes,omitempty"`
}

// GetUser loads a UserRecord by Cognito sub. Returns nil (not an error) when
// the record does not exist — callers should treat a nil result as restricted.
func (c *Client) GetUser(ctx context.Context, userID string) (*UserRecord, error) {
	c.requireUsersTable()
	key := marshalBinaryKey("user_id", userID)
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.usersTable),
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("GetUser: %w", err)
	}
	if out.Item == nil {
		return nil, nil // user not found — treat as restricted
	}
	var record UserRecord
	if err := attributevalue.UnmarshalMap(out.Item, &record); err != nil {
		return nil, fmt.Errorf("GetUser unmarshal: %w", err)
	}
	return &record, nil
}

// PutUser writes a UserRecord unconditionally (used by cognito-post-confirm).
func (c *Client) PutUser(ctx context.Context, u UserRecord) error {
	c.requireUsersTable()
	item, err := attributevalue.MarshalMap(u)
	if err != nil {
		return fmt.Errorf("PutUser marshal: %w", err)
	}
	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.usersTable),
		Item:      item,
	})
	return err
}

// UpdateUser replaces a UserRecord, requiring the record already exists.
func (c *Client) UpdateUser(ctx context.Context, userID string, u UserRecord) error {
	c.requireUsersTable()
	u.UpdatedAt = time.Now().UnixMilli()
	item, err := attributevalue.MarshalMap(u)
	if err != nil {
		return fmt.Errorf("UpdateUser marshal: %w", err)
	}
	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(c.usersTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(user_id)"),
	})
	if err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}
	return nil
}

// UpdateUserTokens atomically increments tokens_used by delta.
// This is a post-hoc accounting write — limit enforcement happens before
// calling Bedrock, not here. The write is unconditional and should always
// succeed as long as the user record exists.
func (c *Client) UpdateUserTokens(ctx context.Context, userID string, delta int) error {
	c.requireUsersTable()
	key := marshalBinaryKey("user_id", userID)
	now := time.Now().UnixMilli()
	_, err := c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        aws.String(c.usersTable),
		Key:              key,
		UpdateExpression: aws.String("ADD tokens_used :delta SET updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":delta": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", delta)},
			":now":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now)},
		},
	})
	if err != nil {
		return fmt.Errorf("UpdateUserTokens: %w", err)
	}
	return nil
}

// ListUsers returns all UserRecords via a full table scan (admin use only).
// Handles pagination automatically.
func (c *Client) ListUsers(ctx context.Context) ([]UserRecord, error) {
	c.requireUsersTable()
	var records []UserRecord
	var lastKey map[string]types.AttributeValue

	for {
		out, err := c.ddb.Scan(ctx, &dynamodb.ScanInput{
			TableName:         aws.String(c.usersTable),
			ExclusiveStartKey: lastKey,
		})
		if err != nil {
			return nil, fmt.Errorf("ListUsers scan: %w", err)
		}
		for _, item := range out.Items {
			var r UserRecord
			if err := attributevalue.UnmarshalMap(item, &r); err != nil {
				continue // skip malformed records
			}
			records = append(records, r)
		}
		if out.LastEvaluatedKey == nil {
			break
		}
		lastKey = out.LastEvaluatedKey
	}
	return records, nil
}
