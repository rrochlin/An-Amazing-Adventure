package campaigns_test

import (
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
)

// scripted returns Briarwell with the village deeds scene written in the
// script format: bounded, a cast with tiers, and beats with meanings.
func scripted(t *testing.T) *campaigns.CampaignDefinition {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	src, _ := reg.Get("briarwell")
	def := *src
	def.StoryNodes = map[string]campaigns.StoryNode{}
	for id, n := range src.StoryNodes {
		def.StoryNodes[id] = n
	}
	node := def.StoryNodes["village_deeds"]
	node.Control = campaigns.ControlBounded
	node.StageDirections.Boundaries = []string{"Stay in Briarwell", "No one is seriously hurt"}
	node.Cast = []campaigns.SceneCastMember{
		{ActorID: "bully", Tier: campaigns.TierPlayer, Part: "Bullying the smaller children in the meadow", Wants: "To look tough and keep his coin"},
		{ActorID: "bess", Part: "Crying by the well"},
	}
	deed := func(key string, outcomes ...campaigns.BeatOutcome) campaigns.Beat {
		return campaigns.Beat{Key: key, Kind: "label", Description: "A deed in Briarwell", Outcomes: outcomes}
	}
	node.Beats = []campaigns.Beat{
		deed("deed_doll",
			campaigns.BeatOutcome{Value: "returned", Meaning: "The player gives Bess her doll back. Learning where it is doesn't count."},
			campaigns.BeatOutcome{Value: "kept", Meaning: "The player finds the doll and keeps or sells it."}),
		deed("deed_miller",
			campaigns.BeatOutcome{Value: "told_wife", Meaning: "The player tells the miller's wife."},
			campaigns.BeatOutcome{Value: "took_bribe", Meaning: "The player takes the miller's coin to keep quiet."}),
		deed("deed_bully",
			campaigns.BeatOutcome{Value: "stopped", Meaning: "The player stops Dunstan."},
			campaigns.BeatOutcome{Value: "joined", Meaning: "The player joins in."}),
	}
	def.StoryNodes["village_deeds"] = node
	return &def
}

func TestScript_ValidSceneAndMeaningsReachTheResolver(t *testing.T) {
	def := scripted(t)
	if err := def.Validate(); err != nil {
		t.Fatalf("scripted scene should validate: %v", err)
	}
	specs := campaigns.AllowedConditionSpecsForNode(def.StoryNodes["village_deeds"])
	var doll *campaigns.AllowedConditionSpec
	for i := range specs {
		if specs[i].Key == "deed_doll" {
			doll = &specs[i]
		}
	}
	if doll == nil || !strings.Contains(doll.Meanings["returned"], "doesn't count") || doll.Description == "" {
		t.Fatalf("resolver specs should carry beat meanings, got %+v", doll)
	}
}

func TestScript_TierDefaultsAndOverrides(t *testing.T) {
	def := scripted(t)
	node := def.StoryNodes["village_deeds"]
	if got := def.Tier(node, "bully"); got != campaigns.TierPlayer {
		t.Fatalf("scene override: got %q", got)
	}
	if got := def.Tier(node, "bess"); got != campaigns.TierActor {
		t.Fatalf("default tier: got %q", got)
	}
}

func TestScript_NormalizeDefaultsAndLegacyContract(t *testing.T) {
	reg, _ := campaigns.LoadEmbeddedRegistry()
	def, _ := reg.Get("briarwell")
	if c := def.StoryNodes["morning_errand"].Control; c != campaigns.ControlControlled {
		t.Fatalf("yarn scenes default to controlled, got %q", c)
	}
	deeds := def.StoryNodes["village_deeds"]
	if deeds.Control != campaigns.ControlBounded {
		t.Fatalf("ai scenes default to bounded, got %q", deeds.Control)
	}
	if len(deeds.StageDirections.MustMention) == 0 || deeds.StageDirections.Focus == "" {
		t.Fatalf("narrator_contract should carry over to stage directions, got %+v", deeds.StageDirections)
	}
}

func TestScript_ValidationErrors(t *testing.T) {
	cases := map[string]func(n *campaigns.StoryNode, d *campaigns.CampaignDefinition){
		"control \"loose\"":               func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) { n.Control = "loose" },
		"no stage_directions.must_happen": func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) { n.Control = campaigns.ControlControlled },
		"unknown actor":                   func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) { n.Cast[0].ActorID = "nobody" },
		"tier \"boss\"":                   func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) { n.Cast[0].Tier = "boss" },
		"appears_in":                      func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) { n.Cast[0].ActorID = "bandit_leader" },
		"without a value or meaning":      func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) { n.Beats[0].Outcomes[0].Meaning = "" },
		"isn't one of its label beats":    func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) { n.Beats = n.Beats[1:] },
		"doesn't define": func(n *campaigns.StoryNode, _ *campaigns.CampaignDefinition) {
			n.Beats[0].Outcomes = n.Beats[0].Outcomes[:1]
		},
		"unknown actor \"ghost\"": func(_ *campaigns.StoryNode, d *campaigns.CampaignDefinition) {
			a := d.Actors["father"]
			a.Relationships = map[string]string{"ghost": "fears it"}
			actors := map[string]campaigns.ActorTemplate{}
			for k, v := range d.Actors {
				actors[k] = v
			}
			actors["father"] = a
			d.Actors = actors
		},
	}
	for want, mutate := range cases {
		t.Run(want, func(t *testing.T) {
			def := scripted(t)
			node := def.StoryNodes["village_deeds"]
			node.Cast = append([]campaigns.SceneCastMember{}, node.Cast...)
			node.Beats = append([]campaigns.Beat{}, node.Beats...)
			node.Beats[0].Outcomes = append([]campaigns.BeatOutcome{}, node.Beats[0].Outcomes...)
			mutate(&node, def)
			def.StoryNodes["village_deeds"] = node
			err := def.Validate()
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want error containing %q, got %v", want, err)
			}
		})
	}
}

func TestNarratorBrief_StageDirectionsAndSceneCast(t *testing.T) {
	def := scripted(t)
	_, g, _ := bootstrapBriarwell(t)
	g.Campaign.ActiveNodeID = "village_deeds"
	brief := campaigns.NarratorBrief(def, g)
	for _, want := range []string{
		"This scene is bounded",
		"Boundaries (never go outside these): Stay in Briarwell; No one is seriously hurt",
		"In this scene, Dunstan: Bullying the smaller children in the meadow Wants: To look tough and keep his coin",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("brief missing %q\n%s", want, brief)
		}
	}
}
