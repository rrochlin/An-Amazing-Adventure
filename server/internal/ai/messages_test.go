package ai

import (
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func TestToBedrockMessages_MergesAdjacentSameRoleTurns(t *testing.T) {
	text := func(role, s string) game.NarrativeMessage {
		return game.NarrativeMessage{Role: role, Content: []game.NarrativeBlock{{Type: "text", Text: s}}}
	}
	msgs := toBedrockMessages([]game.NarrativeMessage{
		text("assistant", "Father speaks."),
		text("user", "I'll do it!"),
		text("assistant", "Father ruffles your hair."),
		text("assistant", "The village square bustles."),
		text("user", "I look for Bess."),
	})
	if len(msgs) != 4 {
		t.Fatalf("expected 4 alternating turns, got %d", len(msgs))
	}
	for i := 1; i < len(msgs); i++ {
		if msgs[i].Role == msgs[i-1].Role {
			t.Fatalf("turns %d and %d share role %s", i-1, i, msgs[i].Role)
		}
	}
	if len(msgs[2].Content) != 2 {
		t.Fatalf("expected the two assistant turns merged, got %d blocks", len(msgs[2].Content))
	}
}
