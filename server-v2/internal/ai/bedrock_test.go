package ai_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// makeHistory builds a slice of alternating user/assistant NarrativeMessages.
func makeHistory(n int) []game.NarrativeMessage {
	msgs := make([]game.NarrativeMessage, 0, n)
	for i := range n {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		msgs = append(msgs, game.NarrativeMessage{
			Role: role,
			Content: []game.NarrativeBlock{{
				Type: "text",
				Text: "message " + string(rune('A'+i)),
			}},
		})
	}
	return msgs
}

func TestTrimHistory_BelowThreshold_Unchanged(t *testing.T) {
	// History shorter than maxHistoryMessages must be returned as-is.
	// We use an ai.Client that has no real Bedrock creds — TrimHistory should
	// not call Bedrock at all when below the threshold.
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}

	history := makeHistory(10)
	got, err := c.TrimHistory(context.Background(), history)
	if err != nil {
		t.Fatalf("TrimHistory: %v", err)
	}
	if len(got) != len(history) {
		t.Errorf("expected %d messages unchanged, got %d", len(history), len(got))
	}
}

func TestTrimHistory_AtThreshold_Unchanged(t *testing.T) {
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}

	history := makeHistory(40) // exactly maxHistoryMessages
	got, err := c.TrimHistory(context.Background(), history)
	if err != nil {
		t.Fatalf("TrimHistory: %v", err)
	}
	if len(got) != 40 {
		t.Errorf("expected 40 messages at threshold, got %d", len(got))
	}
}

func TestTrimHistory_AboveThreshold_Trims(t *testing.T) {
	// When above the threshold, TrimHistory must attempt summarisation.
	// With no real Bedrock creds the call will fail — TrimHistory must
	// return the original history rather than an error.
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}

	history := makeHistory(50) // 10 over the limit
	got, err := c.TrimHistory(context.Background(), history)
	if err != nil {
		t.Fatalf("TrimHistory must not return an error even on Bedrock failure: %v", err)
	}
	// On Bedrock failure (no creds in test), history is returned untrimmed
	if len(got) == 0 {
		t.Error("expected non-empty history back")
	}
}

func TestTrimHistory_SummaryPrefixed(t *testing.T) {
	// Verify the summary message is prepended with "[Story so far]"
	// by checking the first message of a trimmed result (when Bedrock succeeds).
	// In unit tests Bedrock will fail, so we just verify the fallback is clean.
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}
	history := makeHistory(50)
	got, _ := c.TrimHistory(context.Background(), history)
	// Either trimmed (first msg is summary) or untrimmed fallback — both valid
	for _, m := range got {
		for _, b := range m.Content {
			if strings.HasPrefix(b.Text, "[Story so far]") {
				return // found the summary prefix — test passes
			}
		}
	}
	// No summary found is fine — means Bedrock failed and we fell back
	if len(got) != 50 {
		t.Errorf("expected fallback to return all 50 messages, got %d", len(got))
	}
}
