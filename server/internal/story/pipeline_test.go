package story

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// fakeModels records the order of calls and returns canned results.
type fakeModels struct {
	order       []string
	narrator    ai.NarratorResult
	narratorErr error
	engineer    ai.EngineerResult
	engineerErr error
	resolver    ai.CampaignConditionResolutionResult
	resolverErr error
}

func (f *fakeModels) NarrateStream(_ context.Context, _ *game.Game, _ []game.NarrativeMessage, _ string, _ *campaigns.CampaignDefinition, onChunk func(string)) (ai.NarratorResult, error) {
	f.order = append(f.order, "narrator")
	if onChunk != nil {
		onChunk("chunk")
	}
	return f.narrator, f.narratorErr
}

func (f *fakeModels) EngineerScan(context.Context, *game.Game, string) (ai.EngineerResult, error) {
	f.order = append(f.order, "engineer")
	return f.engineer, f.engineerErr
}

func (f *fakeModels) ResolveCampaignConditions(context.Context, *game.Game, *campaigns.CampaignDefinition, string, string) (ai.CampaignConditionResolutionResult, error) {
	f.order = append(f.order, "resolver")
	return f.resolver, f.resolverErr
}

func campaignGame(t *testing.T, id string) (*game.Game, *campaigns.CampaignDefinition) {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := reg.Get(id)
	if !ok {
		t.Fatalf("campaign %q missing", id)
	}
	g, _, err := campaigns.BootstrapGame(def, "s-"+id, "user-1", game.NewCharacter("Hero", ""),
		game.CharacterCreationData{Name: "Hero", RaceID: "human", ClassID: "fighter"})
	if err != nil {
		t.Fatal(err)
	}
	return g, def
}

func TestCurrentRunsTheTurnInOrder(t *testing.T) {
	g, def := campaignGame(t, "lichs-labyrinth")
	models := &fakeModels{
		narrator: ai.NarratorResult{
			Narrative:   "The briefing settles into a plan.",
			NewMessages: []game.NarrativeMessage{{Role: "assistant"}},
			Tokens:      ai.TokenUsage{InputTokens: 10, OutputTokens: 5},
			Calls:       []trace.ModelCall{{Role: "narrator"}},
		},
		engineer: ai.EngineerResult{
			Events:    []game.WorldEvent{{Type: "item_gained", Message: "You gain a map."}},
			Mutations: []game.MutationEntry{{Tool: "give_item_to_player"}},
			Tokens:    ai.TokenUsage{InputTokens: 3, OutputTokens: 1},
			Calls:     []trace.ModelCall{{Role: "engineer"}},
		},
		resolver: ai.CampaignConditionResolutionResult{
			Resolution: campaigns.ConditionResolution{BoolFlags: map[string]bool{"briefing_complete": true}},
			Tokens:     ai.TokenUsage{InputTokens: 2, OutputTokens: 2},
			Calls:      []trace.ModelCall{{Role: "resolver"}},
		},
	}
	var events []string
	out, err := Current{Models: models}.RunTurn(context.Background(), TurnInput{
		Game:            g,
		Campaign:        def,
		PlayerInput:     "we are ready",
		OnChunk:         func(string) { events = append(events, "chunk") },
		OnNarrationDone: func() { events = append(events, "narration done") },
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.Join(models.order, ","); got != "narrator,engineer,resolver" {
		t.Fatalf("call order %s", got)
	}
	if strings.Join(events, ",") != "chunk,narration done" {
		t.Fatalf("callbacks %v: narration must finish before post-narration work", events)
	}
	if g.Campaign.ActiveNodeID != "choose_entry_plan" {
		t.Fatalf("expected the transition to apply to the game, node is %q", g.Campaign.ActiveNodeID)
	}
	if out.Transition == nil || !out.Transition.Transitioned || out.Resolution == nil {
		t.Fatalf("expected resolution and transition, got %+v %+v", out.Resolution, out.Transition)
	}
	if out.AuthoredDialogue != "Choose your route into the labyrinth." {
		t.Fatalf("authored dialogue %q", out.AuthoredDialogue)
	}
	last := out.NewMessages[len(out.NewMessages)-1]
	if last.Content[0].Text != out.AuthoredDialogue {
		t.Fatalf("authored dialogue should close the narrator context, got %+v", last)
	}
	// The scene change goes with the authored text; the engineer's event
	// stays with the narration that caused it.
	if len(out.SceneEvents) == 0 || out.SceneEvents[0].Type != "scene_started" {
		t.Fatalf("scene events %+v", out.SceneEvents)
	}
	if out.NarrativeEvents[0].Type != "item_gained" {
		t.Fatalf("narrative events %+v", out.NarrativeEvents)
	}
	if out.Tokens.Total() != 23 {
		t.Fatalf("tokens %+v, want all three calls summed", out.Tokens)
	}
	if len(out.Calls) != 3 || len(out.Mutations) != 1 || len(out.Errors) != 0 {
		t.Fatalf("calls %d mutations %d errors %v", len(out.Calls), len(out.Mutations), out.Errors)
	}
}

func TestCurrentFailsWhenTheNarratorFails(t *testing.T) {
	g, def := campaignGame(t, "lichs-labyrinth")
	models := &fakeModels{
		narratorErr: errors.New("throttled"),
		narrator:    ai.NarratorResult{Calls: []trace.ModelCall{{Role: "narrator", Error: "throttled"}}},
	}
	done := false
	out, err := Current{Models: models}.RunTurn(context.Background(), TurnInput{
		Game: g, Campaign: def, OnNarrationDone: func() { done = true },
	})
	if err == nil || done {
		t.Fatalf("expected a failed turn with no narration-done signal, err=%v done=%v", err, done)
	}
	if strings.Join(models.order, ",") != "narrator" {
		t.Fatalf("nothing should run after a failed narration: %v", models.order)
	}
	if len(out.Calls) != 1 || len(out.Errors) != 1 || out.Errors[0] != "narrator: throttled" {
		t.Fatalf("the failed call and error should be reported, got %+v %v", out.Calls, out.Errors)
	}
}

func TestCurrentContinuesPastPostNarrationErrors(t *testing.T) {
	g, def := campaignGame(t, "lichs-labyrinth")
	models := &fakeModels{
		narrator:    ai.NarratorResult{Narrative: "Words."},
		engineerErr: errors.New("engineer down"),
		resolverErr: errors.New("bad json"),
	}
	out, err := Current{Models: models}.RunTurn(context.Background(), TurnInput{Game: g, Campaign: def})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(out.Errors, "|") != "engineer: engineer down|resolver: bad json" {
		t.Fatalf("errors %v", out.Errors)
	}
	if out.Narrative != "Words." || out.Resolution != nil || out.Transition != nil {
		t.Fatalf("unexpected output %+v", out)
	}
}

func TestCurrentSkipsTheResolverOutsideCampaigns(t *testing.T) {
	g := game.NewGame("s1", "user-1")
	models := &fakeModels{narrator: ai.NarratorResult{Narrative: "A cave."}}
	out, err := Current{Models: models}.RunTurn(context.Background(), TurnInput{Game: g})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(models.order, ",") != "narrator,engineer" || out.Transition != nil {
		t.Fatalf("order %v transition %+v", models.order, out.Transition)
	}
}
