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

// GetMemberSessions returns all session IDs the user belongs to (owned + joined).
func (c *Client) GetMemberSessions(ctx context.Context, userID string) ([]string, error) {
	records, err := c.GetMembershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(records))
	for _, r := range records {
		ids = append(ids, string(r.SessionID))
	}
	return ids, nil
}

// GetSessionMembers returns all membership records for a session.
// Uses the session-members-index GSI on the memberships table.
func (c *Client) GetSessionMembers(ctx context.Context, sessionID string) ([]MembershipRecord, error) {
	c.requireMembershipsTable()
	out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(c.membershipsTable),
		IndexName:              aws.String("session-members-index"),
		KeyConditionExpression: aws.String("session_id = :sid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sid": binaryIDVal(sessionID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GetSessionMembers: %w", err)
	}
	members := make([]MembershipRecord, 0, len(out.Items))
	for _, item := range out.Items {
		var m MembershipRecord
		if err := attributevalue.UnmarshalMap(item, &m); err != nil {
			continue
		}
		members = append(members, m)
	}
	return members, nil
}

// DeleteMembership removes a single membership record.
func (c *Client) DeleteMembership(ctx context.Context, userID, sessionID string) error {
	c.requireMembershipsTable()
	uk := marshalBinaryKey("user_id", userID)
	sk := marshalBinaryKey("session_id", sessionID)
	_, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(c.membershipsTable),
		Key: map[string]types.AttributeValue{
			"user_id":    uk["user_id"],
			"session_id": sk["session_id"],
		},
	})
	return err
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
