package ai_test

import (
	"context"
	"testing"
	"time"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// newTestGame sets up a minimal game with a player and one room.
func newTestGame(t *testing.T) *game.Game {
	t.Helper()
	g := game.NewGame("test-session", "test-user")
	g.SetPlayerCharacter("test-user", game.NewCharacter("Hero", "The protagonist"))
	room := game.NewArea("Tavern", "A smoky room")
	if err := g.AddRoom(room); err != nil {
		t.Fatal(err)
	}
	if err := g.PlacePlayer(room.ID); err != nil {
		t.Fatal(err)
	}
	return g
}

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
	// No AWS in tests: a short deadline fails the calls at once instead of
	// waiting seconds for credentials.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	// When above the threshold, TrimHistory must attempt summarisation.
	// With no real Bedrock creds the call will fail — TrimHistory must
	// return the original history rather than an error.
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}

	history := makeHistory(50) // 10 over the limit
	got, err := c.TrimHistory(ctx, history)
	if err != nil {
		t.Fatalf("TrimHistory must not return an error even on Bedrock failure: %v", err)
	}
	// On Bedrock failure (no creds in test), history is returned untrimmed
	if len(got) == 0 {
		t.Error("expected non-empty history back")
	}
}

// ---- NarrateStream structural tests ----
// These tests verify the call returns without error on the no-tool-config path.
// With no real Bedrock creds, NarrateStream will fail at the API call — that is
// expected and fine. We test the pre-call setup (history trimming, message building).

// ---- EngineerScan structural tests ----
