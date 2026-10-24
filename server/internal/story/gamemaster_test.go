package story_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story"
)

type fakeGM struct {
	narrative string
	decision  string
	got       ai.GameMasterRequest
}

func (f *fakeGM) GameMasterTurn(_ context.Context, req ai.GameMasterRequest) (ai.GameMasterResult, error) {
	f.got = req
	if req.OnChunk != nil {
		req.OnChunk(f.narrative)
	}
	return ai.GameMasterResult{Narrative: f.narrative, Decision: f.decision, NewMessages: append(req.History, game.NarrativeMessage{Role: "assistant"})}, nil
}

func briarwellAt(t *testing.T, nodeID string) (*campaigns.CampaignDefinition, *game.Game) {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := reg.Get("briarwell")
	g, _, err := campaigns.BootstrapGame(def, "s", "u", game.NewCharacter("Wren", ""), game.CharacterCreationData{Name: "Wren", RaceID: "human", ClassID: "fighter"})
	if err != nil {
		t.Fatal(err)
	}
	if nodeID == "village_deeds" {
		// The real path: answer Father, which activates the deeds.
		if _, err := campaigns.ResumeDialogueWithChoice(def, g, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
			t.Fatal(err)
		}
		if g.Campaign.ActiveNodeID != "village_deeds" {
			t.Fatalf("expected village_deeds, got %q", g.Campaign.ActiveNodeID)
		}
	}
	return def, g
}

func TestGameMaster_AppliesWhatTheSceneAllowsAndAdvancesTheStory(t *testing.T) {
	def, g := briarwellAt(t, "village_deeds")
	// Playtest 5's "confront dunstan", as the game master should play it:
	// moving is allowed, settling the deed is allowed, killing Tomas and
	// inventing Pip are not.
	fake := &fakeGM{
		narrative: "You plant your feet and call out across the meadow.",
		decision: `{"world_events":[
			{"type":"move_player","input":{"room_name":"Meadow"}},
			{"type":"set_character_alive","input":{"character_name":"Tomas","alive":false}},
			{"type":"create_character","input":{"name":"Pip","description":"-","backstory":"-","room_name":"Meadow"}}],
		  "beats":{"deed_bully":"stopped","gift":"ribbon"},
		  "off_script":{"kind":"improvised","reason":"a mud-splatter ambush"}}`,
	}
	var chunks []string
	narrationDone := false
	out, err := story.GameMaster{Models: fake}.RunTurn(context.Background(), story.TurnInput{
		Game: g, Campaign: def, PlayerInput: "confront dunstan",
		OnChunk:         func(s string) { chunks = append(chunks, s) },
		OnNarrationDone: func() { narrationDone = true },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !narrationDone || strings.Join(chunks, "") != fake.narrative || out.Narrative != fake.narrative {
		t.Fatalf("narration should stream and complete: %v %q", chunks, out.Narrative)
	}
	if !strings.Contains(fake.got.System, "This scene is BOUNDED") || fake.got.ToolName != "end_turn" || fake.got.PlayerInput != "confront dunstan" {
		t.Fatalf("request: tool %q, player %q", fake.got.ToolName, fake.got.PlayerInput)
	}
	owner, _ := g.OwnerCharacter()
	if owner.LocationID != "meadow" {
		t.Fatalf("the allowed move should apply, player at %q", owner.LocationID)
	}
	if !g.NPCs["father"].Alive {
		t.Fatal("a death the scene doesn't allow must not apply")
	}
	for _, npc := range g.NPCs {
		if npc.Name == "Pip" {
			t.Fatal("an invented character must not be created")
		}
	}
	joined := strings.Join(out.Errors, "\n")
	for _, want := range []string{`"set_character_alive" is not allowed`, `"create_character" is not allowed`, `beat "gift" is not one of this scene's beats`} {
		if !strings.Contains(joined, want) {
			t.Errorf("errors should report %q:\n%s", want, joined)
		}
	}
	if g.Campaign.Labels["deed_bully"] != "stopped" {
		t.Fatalf("the settled beat should advance the campaign, labels %v", g.Campaign.Labels)
	}
	owner, _ = g.OwnerCharacter()
	coins := 0
	for _, id := range owner.Inventory {
		if g.Items[id].Name == "copper coin" {
			coins++
		}
	}
	if coins != 1 {
		t.Fatalf("the deed's on_success should pay one coin, got %d", coins)
	}
	if out.Decision == nil || len(out.Decision.WorldEvents) != 1 || len(out.Mutations) != 1 || out.Mutations[0].Tool != "move_player" {
		t.Fatalf("decision/mutations: %+v %+v", out.Decision, out.Mutations)
	}
	var types []string
	for _, e := range out.NarrativeEvents {
		types = append(types, e.Type)
	}
	if strings.Join(types, ",") != "player_moved,objective_completed,item_gained" { // the deed's coin
		t.Fatalf("events %v", types)
	}
}

func TestGameMaster_NoDecisionIsReportedNotFatal(t *testing.T) {
	def, g := briarwellAt(t, "village_deeds")
	out, err := story.GameMaster{Models: &fakeGM{narrative: "Quiet."}}.RunTurn(context.Background(), story.TurnInput{Game: g, Campaign: def, PlayerInput: "wait"})
	if err != nil || out.Narrative != "Quiet." || len(out.Errors) != 1 || !strings.Contains(out.Errors[0], "no decision") {
		t.Fatalf("out %+v err %v", out.Errors, err)
	}
}

func TestPipelineForFollowsTheCampaignSwitch(t *testing.T) {
	type both interface {
		story.Models
		story.GMModels
	}
	var models both
	if _, ok := story.PipelineFor(&campaigns.CampaignDefinition{}, models).(story.Current); !ok {
		t.Fatal("default is the current pipeline")
	}
	if _, ok := story.PipelineFor(&campaigns.CampaignDefinition{StoryPipeline: campaigns.StoryPipelineGameMaster}, models).(story.GameMaster); !ok {
		t.Fatal("game_master selects the game master")
	}
	if _, ok := story.PipelineFor(nil, models).(story.Current); !ok {
		t.Fatal("no campaign: current")
	}
}

func TestGameMaster_StagesTheSceneBeforeAnyoneLeaves(t *testing.T) {
	// Playtest 9: Dunstan walks off into the long grass, then leaves the
	// meadow. His mark belongs to the meadow, and the stage gets the cues.
	def, g := briarwellAt(t, "village_deeds")
	if err := g.PlaceCharacter(g.OwnerID, "meadow"); err != nil {
		t.Fatal(err)
	}
	fake := &fakeGM{
		narrative: "You plant your feet.\n\nDunstan: Not worth my time.\n\nHe stalks off through the grass.",
		decision: `{"world_events":[{"type":"move_character","input":{"character_name":"Dunstan","room_name":"Village Square"}}],
		  "beats":{},
		  "staging":[{"line":0,"who":"player","move_to":"old_fence_post"},
		             {"line":2,"who":"Dunstan","move_to":"long_grass"},
		             {"line":7,"who":"Dunstan","emote":"shake"}]}`,
	}
	out, err := story.GameMaster{Models: fake}.RunTurn(context.Background(), story.TurnInput{Game: g, Campaign: def, PlayerInput: "I face Dunstan"})
	if err != nil {
		t.Fatal(err)
	}
	dunstan := g.NPCs["bully"]
	if dunstan.Mark != "long_grass" || dunstan.MarkRoom != "meadow" || dunstan.MarkHere() != "" {
		t.Errorf("his mark stays in the meadow he left: %+v", dunstan)
	}
	if owner, _ := g.OwnerCharacter(); owner.MarkHere() != "old_fence_post" {
		t.Errorf("player mark %q", owner.MarkHere())
	}
	var cues []game.WorldEvent
	for _, e := range out.NarrativeEvents {
		if e.Type == "cue" {
			cues = append(cues, e)
		}
	}
	if len(cues) != 3 || cues[1].Character != "bully" || cues[2].Line != 2 {
		t.Errorf("cues for the stage, the late one on the last paragraph: %+v", cues)
	}
}
