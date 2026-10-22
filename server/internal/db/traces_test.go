package db

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

func TestTraceItemRoundTrip(t *testing.T) {
	in := &trace.Turn{
		SessionID: "s1",
		Kind:      trace.KindChat,
		StartedAt: 1_700_000_000_000,
		Turn:      3,
		Input:     "I look around",
		Calls:     []trace.ModelCall{{Role: "narrator", System: "prompt", Output: "You see a well."}},
	}
	item, err := traceItem(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := item["session_id"].(*types.AttributeValueMemberB); !ok {
		t.Fatalf("session_id must be Binary to match the table key, got %T", item["session_id"])
	}
	if n := item["expires_at"].(*types.AttributeValueMemberN).Value; n != "1702592000" {
		t.Fatalf("expires_at = %s, want start + 30 days", n)
	}
	out, err := decodeTraceItem(item)
	if err != nil {
		t.Fatal(err)
	}
	if out.Input != in.Input || len(out.Calls) != 1 || out.Calls[0].Output != "You see a well." {
		t.Fatalf("round trip lost data: %+v", out)
	}
}

func TestPutTraceWithoutTableIsDisabled(t *testing.T) {
	c := &Client{}
	if err := c.PutTrace(context.Background(), &trace.Turn{}); !errors.Is(err, trace.ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}
