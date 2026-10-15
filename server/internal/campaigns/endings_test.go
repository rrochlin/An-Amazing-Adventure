package campaigns_test

import (
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// endingsCampaign is a minimal two-node campaign whose second node ends the
// run, with two conditional endings and a catch-all.
func endingsCampaign() *campaigns.CampaignDefinition {
	return &campaigns.CampaignDefinition{
		ID: "endings", Version: "1", Title: "Endings", Premise: "p",
		Party: campaigns.PartyTemplate{StartingRoomID: "room"},
		Map:   campaigns.CampaignMap{Rooms: map[string]campaigns.CampaignRoom{"room": {Name: "Room"}}},
		StoryNodes: map[string]campaigns.StoryNode{
			"start": {
				Mode: "ai_scene", Title: "Start",
				CompletionRules: []campaigns.TransitionRule{{
					Conditions: []campaigns.Condition{{Type: "flag_true", Key: "done"}},
					NextNodeID: "finale",
				}},
			},
			"finale": {Mode: "ai_scene", Title: "Finale", EndsRun: true},
		},
		StartNodeID: "start",
		Endings: []campaigns.Ending{
			{ID: "clean", Title: "Clean Success", Conditions: []campaigns.Condition{
				{Type: "flag_true", Key: "artifact_taken"},
				{Type: "flag_false", Key: "anyone_died"},
			}},
			{ID: "costly", Title: "Costly Success", Conditions: []campaigns.Condition{
				{Type: "flag_true", Key: "artifact_taken"},
			}},
			{ID: "failure", Title: "Failure", Summary: "It all went wrong."},
		},
	}
}

func endingsGame(flags map[string]bool) *game.Game {
	g := game.NewGame("session-endings", "user-1")
	g.Campaign = &game.CampaignRuntimeState{CampaignID: "endings", ActiveNodeID: "start", BoolFlags: flags}
	return g
}

func TestSelectEnding_FirstMatchWinsWithCatchAll(t *testing.T) {
	def := endingsCampaign()
	cases := []struct {
		flags map[string]bool
		want  string
	}{
		{map[string]bool{"artifact_taken": true}, "clean"},
		{map[string]bool{"artifact_taken": true, "anyone_died": true}, "costly"},
		{map[string]bool{}, "failure"},
	}
	for _, tc := range cases {
		ending, err := campaigns.SelectEnding(def, endingsGame(tc.flags))
		if err != nil {
			t.Fatalf("select ending: %v", err)
		}
		if ending == nil || ending.ID != tc.want {
			t.Fatalf("flags %v: expected %q, got %#v", tc.flags, tc.want, ending)
		}
	}
}

func TestAdvanceCampaign_EnteringEndsRunNodeEndsTheRun(t *testing.T) {
	def := endingsCampaign()
	g := endingsGame(map[string]bool{"done": true, "artifact_taken": true, "anyone_died": true})

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if !result.RunEnded || result.ToNodeID != "finale" {
		t.Fatalf("expected the run to end on entering finale, got %#v", result)
	}
	c := g.Campaign
	if !c.RunEnded() || c.EndingID != "costly" || c.EndingTitle != "Costly Success" || c.EndedAt == 0 {
		t.Fatalf("expected ended run with costly ending, got %#v", c)
	}

	view := g.BuildCampaignStateView()
	if view.RunStatus != game.RunStatusEnded || view.Ending == nil || view.Ending.ID != "costly" {
		t.Fatalf("expected ending in the client view, got %#v", view)
	}
}

func TestEndedRunIgnoresFurtherTurns(t *testing.T) {
	def := endingsCampaign()
	g := endingsGame(map[string]bool{"done": true})
	if _, err := campaigns.AdvanceCampaign(def, g); err != nil {
		t.Fatalf("advance: %v", err)
	}
	endedAt := g.Campaign.EndedAt

	result, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{BoolFlags: map[string]bool{"done": true}})
	if err != nil {
		t.Fatalf("orchestrate: %v", err)
	}
	if result.Transition.Transitioned || len(result.Events) != 0 {
		t.Fatalf("expected an ended run to ignore turns, got %#v", result)
	}
	if text, err := campaigns.EndRun(def, g); err != nil || text != "" || g.Campaign.EndedAt != endedAt {
		t.Fatalf("expected ending twice to be a no-op, got %q %v %#v", text, err, g.Campaign)
	}
}

func TestOrchestrateTurn_PartyWipeEndsRun(t *testing.T) {
	def := endingsCampaign()
	g := endingsGame(map[string]bool{})
	hero := game.NewCharacter("Hero", "")
	hero.Alive = false
	g.Players = map[string]game.Character{"user-1": hero}

	result, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{BoolFlags: map[string]bool{"done": true}})
	if err != nil {
		t.Fatalf("orchestrate: %v", err)
	}
	if !result.Transition.RunEnded || g.Campaign.EndingID != "failure" {
		t.Fatalf("expected the wipe to end the run with the catch-all, got %#v / %#v", result, g.Campaign)
	}
	if g.Campaign.ActiveNodeID != "start" {
		t.Fatalf("a wipe should not apply the turn's story transition, got node %q", g.Campaign.ActiveNodeID)
	}
	if len(result.Events) != 1 || !strings.Contains(result.Events[0].Message, "Failure") {
		t.Fatalf("expected a campaign_ended event naming the ending, got %#v", result.Events)
	}
}

func TestPartyWiped_LivingPlayerOrNoPlayers(t *testing.T) {
	g := game.NewGame("session-wipe", "user-1")
	g.Players = map[string]game.Character{}
	if g.PartyWiped() {
		t.Fatal("a game with no players is not wiped")
	}
	dead := game.NewCharacter("Dead", "")
	dead.Alive = false
	g.Players = map[string]game.Character{"a": dead, "b": game.NewCharacter("Alive", "")}
	if g.PartyWiped() {
		t.Fatal("a party with a living member is not wiped")
	}
}

func TestBriarwell_ReachingTheRoadEndsTheRun(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "I'll do it", "village_deeds")
	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_doll": "kept", "deed_miller": "told_wife"},
	})
	advanceTo(t, def, g, "gift_stall")
	chooseOption(t, def, g, "ribbon", "birthday")
	chooseOption(t, def, g, "Give Ros", "bandit_raid")
	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{
		Labels: map[string]string{"raid_response": "hid"},
	})
	advanceTo(t, def, g, "rescue")
	chooseOption(t, def, g, "No! I have to find my family", "road_to_guild")

	if !g.Campaign.RunEnded() || g.Campaign.EndingID != "road_to_guild" {
		t.Fatalf("expected Briarwell to end on the road to the guild, got %#v", g.Campaign)
	}
}

func TestValidate_Endings(t *testing.T) {
	cases := map[string]func(d *campaigns.CampaignDefinition){
		"last ending has conditions": func(d *campaigns.CampaignDefinition) {
			d.Endings = d.Endings[:2]
		},
		"ends_run without endings": func(d *campaigns.CampaignDefinition) {
			d.Endings = nil
		},
		"duplicate ending id": func(d *campaigns.CampaignDefinition) {
			d.Endings[1].ID = "clean"
		},
		"ending without title": func(d *campaigns.CampaignDefinition) {
			d.Endings[2].Title = ""
		},
		"unknown ending dialogue": func(d *campaigns.CampaignDefinition) {
			d.Endings[2].DialogueAssetID = "missing"
			d.Endings[2].DialogueStartNode = "Start"
		},
	}
	if err := endingsCampaign().Validate(); err != nil {
		t.Fatalf("expected the base campaign to validate, got %v", err)
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := endingsCampaign()
			mutate(d)
			if err := d.Validate(); err == nil {
				t.Fatalf("expected a validation error for %s", name)
			}
		})
	}
}
