package db

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// InviteRecord is a party invite code stored in the invites table.
// The DynamoDB TTL on expires_at causes codes to auto-expire.
type InviteRecord struct {
	Code      string   `dynamodbav:"code"`       // PK — 6-char alphanumeric
	SessionID BinaryID `dynamodbav:"session_id"` // game session this invite is for
	CreatedBy BinaryID `dynamodbav:"created_by"` // user_id of session owner
	ExpiresAt int64    `dynamodbav:"expires_at"` // Unix epoch SECONDS (DynamoDB TTL)
	MaxUses   int      `dynamodbav:"max_uses"`   // 0 = unlimited
	Uses      int      `dynamodbav:"uses"`       // current redemption count
}

// GetInvite loads an invite record by code. Returns nil when not found.
func (c *Client) GetInvite(ctx context.Context, code string) (*InviteRecord, error) {
	c.requireInvitesTable()
	codeAttr, _ := attributevalue.Marshal(code)
	out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.invitesTable),
		Key:       map[string]types.AttributeValue{"code": codeAttr},
	})
	if err != nil {
		return nil, fmt.Errorf("GetInvite: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}
	var r InviteRecord
	if err := attributevalue.UnmarshalMap(out.Item, &r); err != nil {
		return nil, fmt.Errorf("GetInvite unmarshal: %w", err)
	}
	return &r, nil
}

// PutInvite writes an invite record.
func (c *Client) PutInvite(ctx context.Context, inv InviteRecord) error {
	c.requireInvitesTable()
	item, err := attributevalue.MarshalMap(inv)
	if err != nil {
		return fmt.Errorf("PutInvite marshal: %w", err)
	}
	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.invitesTable),
		Item:      item,
	})
	return err
}

// IncrementInviteUses atomically increments the uses counter on an invite record.
func (c *Client) IncrementInviteUses(ctx context.Context, code string) error {
	c.requireInvitesTable()
	codeAttr, _ := attributevalue.Marshal(code)
	_, err := c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        aws.String(c.invitesTable),
		Key:              map[string]types.AttributeValue{"code": codeAttr},
		UpdateExpression: aws.String("ADD #u :one"),
		ExpressionAttributeNames: map[string]string{
			"#u": "uses",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one": &types.AttributeValueMemberN{Value: "1"},
		},
	})
	if err != nil {
		return fmt.Errorf("IncrementInviteUses: %w", err)
	}
	return nil
}
