package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// candidate is a fake pipeline: it records what it was given, moves the
// campaign on, and reports one call.
type candidate struct {
	got []story.TurnInput
}

func (c *candidate) RunTurn(_ context.Context, in story.TurnInput) (story.TurnOutput, error) {
	c.got = append(c.got, in)
	if in.Game.Campaign.Labels == nil {
		in.Game.Campaign.Labels = map[string]string{}
	}
	in.Game.Campaign.Labels["deed_bully"] = "stopped"
	return story.TurnOutput{
		Narrative: "Dunstan lands in the clover.",
		Calls:     []trace.ModelCall{{Role: "game_master", InputTok: 900, OutputTok: 300, DurationMs: 12000}},
		Errors:    []string{"engineer: slow"},
	}, nil
}

func briarwell(t *testing.T) (*campaigns.Registry, *game.Game, game.SaveState) {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := reg.Get("briarwell")
	g, history, err := campaigns.BootstrapGame(def, "s1", "user-1", game.NewCharacter("Wren", ""),
		game.CharacterCreationData{Name: "Wren", RaceID: "human", ClassID: "fighter"})
	if err != nil {
		t.Fatal(err)
	}
	narrative := []game.NarrativeMessage{{Role: "assistant", Content: []game.NarrativeBlock{{Type: "text", Text: "Sunlight spills in."}}}}
	return reg, g, g.ToSaveState(narrative, history)
}

// stepClock advances 12s every time it's read, so each replayed turn takes
// 12s: over the 10s budget.
func stepClock() func() time.Time {
	now := time.Unix(1_700_000_000, 0)
	return func() time.Time {
		now = now.Add(12 * time.Second)
		return now
	}
}

func chatTrace(g *game.Game, save game.SaveState, input string) *trace.Turn {
	tr := trace.Start(trace.KindChat, "ws-chat", "s1", g)
	tr.Before.World = trace.WorldOf(save)
	tr.Input = input
	tr.Call(trace.ModelCall{Role: "narrator", Output: "You burst from the grass.", InputTok: 1000, OutputTok: 200, DurationMs: 6000},
		trace.ModelCall{Role: "resolver", Output: `{"labels":{"deed_bully":"stopped"}}`, DurationMs: 500})
	tr.Finish(g)
	return tr
}

func TestReplayRebuildsTheTurnAndReports(t *testing.T) {
	reg, g, save := briarwell(t)
	tr := chatTrace(g, save, "I ambush Dunstan")
	// The session moved on after this turn; the replay must start from the
	// traced world, not the final one.
	final := save
	final.Campaign = &game.CampaignRuntimeState{CampaignID: "briarwell", ActiveNodeID: "road_to_guild"}
	choice := trace.Start(trace.KindDialogueChoice, "ws-game-action", "s1", g)

	cand := &candidate{}
	results := runner{Registry: reg, Pipeline: func(*trace.Turn) story.Pipeline { return cand }, Now: stepClock()}.
		replay(context.Background(), final, []*trace.Turn{tr, choice})

	if len(cand.got) != 1 {
		t.Fatalf("expected one chat turn replayed, got %d", len(cand.got))
	}
	in := cand.got[0]
	if in.PlayerInput != "I ambush Dunstan" || in.Campaign == nil || in.Campaign.ID != "briarwell" {
		t.Fatalf("unexpected input %+v", in)
	}
	if in.Game.Campaign.ActiveNodeID != "morning_errand" {
		t.Fatalf("replay should start from the traced world, got node %q", in.Game.Campaign.ActiveNodeID)
	}
	if len(in.History) != 1 || in.History[0].Content[0].Text != "Sunlight spills in." {
		t.Fatalf("narrator context should come from the traced world, got %+v", in.History)
	}
	if len(results[0].Missing) != 0 {
		t.Fatalf("a full trace needs no approximation, got %v", results[0].Missing)
	}
	if !strings.HasPrefix(results[1].Skipped, "dialogue_choice turn") {
		t.Fatalf("non-chat turns are skipped, got %q", results[1].Skipped)
	}

	var out bytes.Buffer
	render(&out, save, "fake", results)
	report := out.String()
	for _, want := range []string{
		"Candidate pipeline: **fake**. Latency budget: 10s per turn.",
		"- turns replayed: 1, skipped: 1, rebuilt with missing data: 0",
		"- over budget: original 0, candidate 1",
		"| Latency | 6.5s (within budget) | 12.0s (**over budget**) |",
		"| Tokens in / out | 1000 / 200 | 900 / 300 |",
		"| Model calls | narrator 6.0s, resolver 0.5s | game_master 12.0s |",
		"| Errors | none | engineer: slow |",
		"- label deed_bully set: stopped",
		"> You burst from the grass.",
		"> Dunstan lands in the clover.",
		"_Skipped: dialogue_choice turn",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q\n\n%s", want, report)
		}
	}
}

func TestReplayApproximatesOlderTraces(t *testing.T) {
	reg, g, save := briarwell(t)
	// An older trace: campaign state and room only, no starting world.
	tr := trace.Start(trace.KindChat, "ws-chat", "s1", g)
	tr.Input = "Go to the meadow"
	tr.Before.PlayerRoomID = "meadow"
	tr.Call(trace.ModelCall{Role: "narrator", Messages: []game.NarrativeMessage{
		{Role: "assistant", Content: []game.NarrativeBlock{{Type: "text", Text: "[Story so far] ..."}}},
		{Role: "user", Content: []game.NarrativeBlock{{Type: "text", Text: "Go to the meadow"}}},
	}})
	final := save
	final.Campaign = &game.CampaignRuntimeState{CampaignID: "briarwell", ActiveNodeID: "road_to_guild"}

	cand := &candidate{}
	results := runner{Registry: reg, Pipeline: func(*trace.Turn) story.Pipeline { return cand }}.
		replay(context.Background(), final, []*trace.Turn{tr})

	in := cand.got[0]
	if in.Game.Campaign.ActiveNodeID != "morning_errand" {
		t.Fatalf("the traced campaign state should be laid over the final world, got %q", in.Game.Campaign.ActiveNodeID)
	}
	if owner, _ := in.Game.OwnerCharacter(); owner.LocationID != "meadow" {
		t.Fatalf("the traced player room should be restored, got %q", owner.LocationID)
	}
	if len(in.History) != 1 || in.History[0].Content[0].Text != "[Story so far] ..." {
		t.Fatalf("history should be the traced narrator request minus this turn's input, got %+v", in.History)
	}
	if len(results[0].Missing) != 2 {
		t.Fatalf("expected the world and narrator context to be reported missing, got %v", results[0].Missing)
	}
	var out bytes.Buffer
	render(&out, save, "fake", results)
	if !strings.Contains(out.String(), "> **Not in the trace:** starting world") {
		t.Errorf("report should say what the trace lacked:\n%s", out.String())
	}
}

func TestRecordedPipelineRepeatsTheOriginal(t *testing.T) {
	reg, g, save := briarwell(t)
	tr := chatTrace(g, save, "hello")
	results := runner{Registry: reg, Pipeline: func(t *trace.Turn) story.Pipeline { return recorded{t} }}.
		replay(context.Background(), save, []*trace.Turn{tr})
	if results[0].Output.Narrative != "You burst from the grass." || results[0].Err != nil {
		t.Fatalf("recorded pipeline should return the original narration, got %+v", results[0])
	}
}

func TestReferenceTurnsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	world, _ := json.Marshal(game.SaveState{SessionID: "abcdef12-session", Title: "T", OwnerID: "cognito-sub", UserID: "cognito-sub",
		Players: map[string]game.Character{"cognito-sub": {ID: "c1", Name: "Hero"}}})
	chat := &trace.Turn{SessionID: "abcdef12-session", Kind: trace.KindChat, Input: "look", UserID: "cognito-sub", ConnectionID: "conn",
		Before: &trace.Snapshot{World: world}, Frames: []trace.Frame{{Type: "state_delta", To: "conn"}}}
	choice := &trace.Turn{SessionID: "abcdef12-session", Kind: trace.KindDialogueChoice, Before: &trace.Snapshot{World: world}}
	bare := &trace.Turn{SessionID: "abcdef12-session", Kind: trace.KindChat, Before: &trace.Snapshot{}}
	save := game.SaveState{SessionID: "abcdef12-session", Title: "T"}

	written, err := saveReference(dir, save, []*trace.Turn{chat, choice, bare}, []int{1}, "baseline")
	if err != nil || len(written) != 1 || !strings.HasSuffix(written[0], "abcdef12-01.json") {
		t.Fatalf("save: %v %v", written, err)
	}
	if _, err := saveReference(dir, save, []*trace.Turn{chat, choice, bare}, []int{2}, ""); err == nil {
		t.Fatal("a dialogue_choice turn has nothing to replay and should be refused")
	}
	if _, err := saveReference(dir, save, []*trace.Turn{chat, choice, bare}, []int{3}, ""); err == nil {
		t.Fatal("a turn without a starting world should be refused")
	}

	refs, err := loadReference(dir)
	if err != nil || len(refs) != 1 {
		t.Fatalf("load: %v %v", refs, err)
	}
	r := refs[0]
	if r.Note != "baseline" || r.Turn != 1 || r.Trace.Input != "look" || r.saveFor().Title != "T" {
		t.Fatalf("round trip lost data: %+v", r)
	}
	if r.Trace.UserID != "" || r.Trace.ConnectionID != "" || r.Trace.Frames[0].To != "" {
		t.Fatalf("saved turns must not carry who played or on which socket: %+v", r.Trace)
	}
	if strings.Contains(string(r.Trace.Before.World), "cognito-sub") {
		t.Fatal("the player's id must be replaced inside the starting world too")
	}
	var w game.SaveState
	if err := json.Unmarshal(r.Trace.Before.World, &w); err != nil || w.OwnerID != "player" || w.Players["player"].Name != "Hero" {
		t.Fatalf("the world should still rebuild with the placeholder id: %+v %v", w, err)
	}
	if got, err := parseTurns("2, 3,5", 9); err != nil || len(got) != 3 || got[2] != 5 {
		t.Fatalf("parseTurns: %v %v", got, err)
	}
	if _, err := parseTurns("7", 3); err == nil {
		t.Fatal("out-of-range turn should fail")
	}
}
