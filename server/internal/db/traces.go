package db

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// ErrTracingDisabled is returned by PutTrace when TRACES_TABLE is not set.
// Tracing is diagnostic, so a Lambda without the table keeps working.
var ErrTracingDisabled = errors.New("TRACES_TABLE is not set; turn tracing is disabled")

// traceItem builds the traces-table item for a turn. The full trace is
// gzipped JSON in `data` (prompts and history are large); the plain
// attributes are there for filtering without decoding.
func traceItem(t *trace.Turn) (map[string]types.AttributeValue, error) {
	raw, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("marshal trace: %w", err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return nil, fmt.Errorf("compress trace: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("compress trace: %w", err)
	}
	return map[string]types.AttributeValue{
		"session_id": binaryIDVal(t.SessionID),
		"ts":         &types.AttributeValueMemberN{Value: strconv.FormatInt(t.StartedAt, 10)},
		"kind":       &types.AttributeValueMemberS{Value: t.Kind},
		"turn":       &types.AttributeValueMemberN{Value: strconv.Itoa(t.Turn)},
		"expires_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(t.ExpiresAt(), 10)},
		"data":       &types.AttributeValueMemberB{Value: buf.Bytes()},
	}, nil
}

func decodeTraceItem(item map[string]types.AttributeValue) (*trace.Turn, error) {
	data, ok := item["data"].(*types.AttributeValueMemberB)
	if !ok {
		return nil, fmt.Errorf("trace item has no data")
	}
	zr, err := gzip.NewReader(bytes.NewReader(data.Value))
	if err != nil {
		return nil, fmt.Errorf("decompress trace: %w", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("decompress trace: %w", err)
	}
	t := &trace.Turn{}
	if err := json.Unmarshal(raw, t); err != nil {
		return nil, fmt.Errorf("unmarshal trace: %w", err)
	}
	return t, nil
}

// PutTrace stores a turn's trace. It returns ErrTracingDisabled when
// TRACES_TABLE is not set.
func (c *Client) PutTrace(ctx context.Context, t *trace.Turn) error {
	if c.tracesTable == "" {
		return ErrTracingDisabled
	}
	item, err := traceItem(t)
	if err != nil {
		return err
	}
	if _, err := c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.tracesTable),
		Item:      item,
	}); err != nil {
		return fmt.Errorf("put trace: %w", err)
	}
	return nil
}

// querySession returns every item for sessionID in table, oldest first.
func (c *Client) querySession(ctx context.Context, table, sessionID string) ([]map[string]types.AttributeValue, error) {
	var items []map[string]types.AttributeValue
	var start map[string]types.AttributeValue
	for {
		out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
			TableName:                 aws.String(table),
			KeyConditionExpression:    aws.String("session_id = :sid"),
			ExpressionAttributeValues: map[string]types.AttributeValue{":sid": binaryIDVal(sessionID)},
			ExclusiveStartKey:         start,
		})
		if err != nil {
			return nil, fmt.Errorf("query %s: %w", table, err)
		}
		items = append(items, out.Items...)
		if out.LastEvaluatedKey == nil {
			return items, nil
		}
		start = out.LastEvaluatedKey
	}
}

// ListTraces returns a session's turn traces, oldest first.
func (c *Client) ListTraces(ctx context.Context, sessionID string) ([]*trace.Turn, error) {
	if c.tracesTable == "" {
		return nil, ErrTracingDisabled
	}
	items, err := c.querySession(ctx, c.tracesTable, sessionID)
	if err != nil {
		return nil, err
	}
	turns := make([]*trace.Turn, 0, len(items))
	for _, item := range items {
		t, err := decodeTraceItem(item)
		if err != nil {
			return nil, err
		}
		turns = append(turns, t)
	}
	return turns, nil
}

// ListMutations returns a session's engineer mutation log, oldest first.
func (c *Client) ListMutations(ctx context.Context, sessionID string) ([]game.MutationEntry, error) {
	c.requireMutationsTable()
	items, err := c.querySession(ctx, c.mutationsTable, sessionID)
	if err != nil {
		return nil, err
	}
	entries := make([]game.MutationEntry, 0, len(items))
	for _, item := range items {
		delete(item, "session_id") // B-typed; not decodable into a string
		var e game.MutationEntry
		if err := attributevalue.UnmarshalMap(item, &e); err != nil {
			return nil, fmt.Errorf("unmarshal mutation: %w", err)
		}
		e.SessionID = sessionID
		entries = append(entries, e)
	}
	return entries, nil
}
