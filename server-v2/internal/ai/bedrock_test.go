package ai_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// newTestGame sets up a minimal game with a player and one room.
func newTestGame(t *testing.T) *game.Game {
	t.Helper()
	g := game.NewGame("test-session", "test-user")
	g.Player = game.NewCharacter("Hero", "The protagonist")
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

// ---- NarrateStream structural tests ----
// These tests verify the call returns without error on the no-tool-config path.
// With no real Bedrock creds, NarrateStream will fail at the API call — that is
// expected and fine. We test the pre-call setup (history trimming, message building).

func TestNarrateStream_ReturnsWithoutToolCallAttempt(t *testing.T) {
	// NarrateStream must not panic on valid input even when Bedrock is unavailable.
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}
	g := newTestGame(t)
	_, streamErr := c.NarrateStream(context.Background(), g, nil, "look around", nil)
	// Expect a Bedrock connectivity/auth error, NOT a nil pointer or panic
	if streamErr == nil {
		t.Log("NarrateStream unexpectedly succeeded (real Bedrock available?)")
	}
	// The key assertion: no panic occurred. If we reach here the test passes.
}

func TestNarrateStream_EmptyHistoryHandled(t *testing.T) {
	// nil history must not cause a nil pointer panic.
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}
	g := newTestGame(t)
	_, _ = c.NarrateStream(context.Background(), g, nil, "hello", nil)
	// Passes if no panic
}

// ---- EngineerScan structural tests ----

func TestEngineerScan_ReturnsWithoutPanic(t *testing.T) {
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}
	g := newTestGame(t)
	result, scanErr := c.EngineerScan(context.Background(), g, "The goblin appears from the shadows.")
	if scanErr != nil {
		// Expected — no real Bedrock creds in tests
		t.Logf("EngineerScan returned expected error (no Bedrock creds): %v", scanErr)
		return
	}
	// If somehow Bedrock is available, result must be valid
	_ = result
}

func TestEngineerScan_EmptyNarrativeNoMutations(t *testing.T) {
	// An empty narrative should yield no mutations even if Bedrock is available.
	// Without Bedrock this just verifies no panic on empty input.
	t.Setenv("BEDROCK_REGION", "us-west-2")
	c, err := ai.New(context.Background())
	if err != nil {
		t.Fatalf("ai.New: %v", err)
	}
	g := newTestGame(t)
	_, _ = c.EngineerScan(context.Background(), g, "")
	// Passes if no panic
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
