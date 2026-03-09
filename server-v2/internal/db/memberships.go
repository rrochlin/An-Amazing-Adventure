package db

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// MembershipRecord links a user to a game session they belong to.
// PK: user_id (B), SK: session_id (B).
// GSI "session-members-index" on session_id enables reverse lookup.
type MembershipRecord struct {
	UserID    BinaryID `dynamodbav:"user_id"`    // PK
	SessionID BinaryID `dynamodbav:"session_id"` // SK
	Role      string   `dynamodbav:"role"`       // "owner" | "member"
	JoinedAt  int64    `dynamodbav:"joined_at"`  // Unix ms
}

// PutMembership writes a membership record.
func (c *Client) PutMembership(ctx context.Context, m MembershipRecord) error {
	c.requireMembershipsTable()
	item, err := attributevalue.MarshalMap(m)
	if err != nil {
		return fmt.Errorf("PutMembership marshal: %w", err)
	}
	_, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.membershipsTable),
		Item:      item,
	})
	return err
}

// GetMembershipsByUser returns all membership records for a user.
func (c *Client) GetMembershipsByUser(ctx context.Context, userID string) ([]MembershipRecord, error) {
	c.requireMembershipsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.membershipsTable),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GetMembershipsByUser: %w", err)
	}
	records := make([]MembershipRecord, 0, len(out.Items))
	for _, item := range out.Items {
		var r MembershipRecord
		if err := attributevalue.UnmarshalMap(item, &r); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, nil
}

// CountUserGames returns the number of sessions a user owns (via user-sessions-index
// on the sessions table). Used to enforce the games_limit quota.
func (c *Client) CountUserGames(ctx context.Context, userID string) (int, error) {
	c.requireSessionsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.sessionsTable),
		IndexName:              aws.String("user-sessions-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": binaryIDVal(userID),
		},
		Select: types.SelectCount,
	})
	if err != nil {
		return 0, fmt.Errorf("CountUserGames: %w", err)
	}
	return int(out.Count), nil
}
