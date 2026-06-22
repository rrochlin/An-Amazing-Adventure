package db

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func TestMutationEntryItem_UsesBinarySessionID(t *testing.T) {
	item, err := mutationEntryItem(game.MutationEntry{
		SessionID: "session-123",
		Ts:        123,
		Turn:      7,
		Tool:      "damage_character",
		Input:     map[string]any{"amount": 12},
		Result:    "ok",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessionID, ok := item["session_id"].(*types.AttributeValueMemberB)
	if !ok {
		t.Fatalf("expected session_id to marshal as Binary, got %T", item["session_id"])
	}
	if got := string(sessionID.Value); got != "session-123" {
		t.Fatalf("expected binary session_id %q, got %q", "session-123", got)
	}

	timestamp, ok := item["ts"].(*types.AttributeValueMemberN)
	if !ok || timestamp.Value != "123" {
		t.Fatalf("expected ts to marshal as number 123, got %T %v", item["ts"], item["ts"])
	}

	tool, ok := item["tool"].(*types.AttributeValueMemberS)
	if !ok || tool.Value != "damage_character" {
		t.Fatalf("expected tool to marshal as string damage_character, got %T %v", item["tool"], item["tool"])
	}

	input, ok := item["input"].(*types.AttributeValueMemberM)
	if !ok {
		t.Fatalf("expected input to marshal as map, got %T", item["input"])
	}
	inputValue, err := attributevalue.MarshalMap(map[string]any{"amount": 12})
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if _, exists := input.Value["amount"]; !exists {
		t.Fatalf("expected input map to include amount, got %v", input.Value)
	}
	if _, ok := inputValue["amount"].(*types.AttributeValueMemberN); !ok {
		t.Fatalf("expected amount to marshal as number, got %T", inputValue["amount"])
	}
}

func TestPutMutationRequiresMutationsTable(t *testing.T) {
	client := &Client{}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for missing MUTATIONS_TABLE")
		}
		msg := ""
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(msg, "MUTATIONS_TABLE") {
			t.Fatalf("panic message %q does not mention MUTATIONS_TABLE", msg)
		}
	}()

	_ = client.PutMutation(context.Background(), game.MutationEntry{SessionID: "session-123"})
}