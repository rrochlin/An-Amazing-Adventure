package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

func TestRenderShowsTurnStateChanges(t *testing.T) {
	g := &game.Game{
		Version: 4,
		Rooms:   map[string]game.Area{"cottage": {ID: "cottage", Name: "Family Cottage"}},
		Campaign: &game.CampaignRuntimeState{
			CampaignID:       "briarwell",
			ActiveNodeID:     "morning_errand",
			CurrentObjective: "Listen to Father",
			ActiveObjectives: []game.ObjectiveState{{ID: "listen", Status: "active"}},
			ActiveDialogue:   &game.DialogueRuntimeState{AssetID: "errand", CurrentNode: "Start", AwaitingChoice: true, PendingChoices: []game.DialogueChoice{{ID: 0}, {ID: 1}}},
		},
	}
	tr := trace.Start(trace.KindDialogueChoice, "ws-game-action", "s1", g)
	tr.Input = "dialogue_choice 0"
	g.Campaign.ActiveNodeID = "village_deeds"
	g.Campaign.CurrentObjective = "Earn two coins"
	g.Campaign.ActiveObjectives = []game.ObjectiveState{{ID: "listen", Status: "completed"}, {ID: "coins", Status: "active"}}
	g.Campaign.Labels = map[string]string{"attitude": "eager"}
	g.Campaign.ActiveDialogue = nil
	tr.Call(trace.ModelCall{Role: "resolver", Model: "haiku", Output: `{"labels":{}}`, System: "SYSTEM PROMPT"})
	tr.Chunk("narrative_chunk")
	tr.Chunk("narrative_chunk")
	tr.Frame("state_delta", "conn-1", nil)
	tr.Errorf("put game attempt 1: conflict")
	tr.Finish(g)

	save := game.SaveState{
		SessionID:   "s1",
		Title:       "Briarwell",
		Campaign:    g.Campaign,
		ChatHistory: []game.ChatMessage{{Type: "player", Content: "I'll do it!"}, {Type: "narrative", Content: "Father ruffles\nyour hair."}},
	}
	var out bytes.Buffer
	render(&out, save, []*trace.Turn{tr}, []game.MutationEntry{{Turn: 1, Tool: "move_player", Input: map[string]any{"room": "square"}, Result: "ok"}}, false)
	report := out.String()

	for _, want := range []string{
		"# Session s1 — Briarwell",
		"1. **NARRATIVE** Father ruffles your hair.",
		"### 1. dialogue_choice `dialogue_choice 0`",
		"- player room: -",
		"- node: morning_errand → **village_deeds**",
		`- objective: "Listen to Father" → "Earn two coins"`,
		"- objective listen: active → completed",
		"- objective coins added: active",
		"- label attitude set: eager",
		"- dialogue: errand/Start (awaiting choice, 2 options) → none",
		"- resolver call (haiku)",
		"    - verdict: `{\"labels\":{}}`",
		"- frames: narrative_chunk×2, state_delta→conn-1",
		"- **error**: put game attempt 1: conflict",
		"- turn 1 `move_player` {\"room\":\"square\"} → ok",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q\n\n%s", want, report)
		}
	}
	if strings.Contains(report, "SYSTEM PROMPT") {
		t.Errorf("prompts should only appear with -full")
	}

	out.Reset()
	render(&out, save, []*trace.Turn{tr}, nil, true)
	if !strings.Contains(out.String(), "SYSTEM PROMPT") {
		t.Errorf("-full should include the system prompt")
	}
}
