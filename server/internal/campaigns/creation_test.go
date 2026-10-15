package campaigns_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func briarwellRules(t *testing.T) campaigns.CharacterCreationRules {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := reg.Get("briarwell")
	if !ok {
		t.Fatal("expected briarwell")
	}
	return def.CharacterCreation
}

func TestApply_PresetIgnoresEverythingButTheName(t *testing.T) {
	rules := briarwellRules(t)
	out, err := rules.Apply(game.CharacterCreationData{
		Name:          "  Pip  ",
		Backstory:     "A grizzled dwarven mercenary.",
		RaceID:        "dwarf",
		ClassID:       "barbarian",
		AbilityScores: map[string]int{"str": 15},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if out.Name != "Pip" || out.RaceID != "human" || out.ClassID != "fighter" || len(out.AbilityScores) != 6 {
		t.Fatalf("expected the preset human fighter named Pip, got %#v", out)
	}
	if !strings.Contains(out.Backstory, "child of about ten") {
		t.Fatalf("expected the locked background, got %q", out.Backstory)
	}

	// The preset must actually build, or every Briarwell game would fail.
	if _, err := game.BuildDnDCharacter(context.Background(), out); err != nil {
		t.Fatalf("Briarwell preset does not build: %v", err)
	}
}

func TestApply_PresetFallsBackToDefaultName(t *testing.T) {
	out, err := briarwellRules(t).Apply(game.CharacterCreationData{})
	if err != nil || out.Name != "Wren" {
		t.Fatalf("expected default name Wren, got %q (%v)", out.Name, err)
	}
}

func TestApply_StandardRulesEnforceAllowedLists(t *testing.T) {
	rules := campaigns.CharacterCreationRules{
		AllowedRaces:         []string{"human", "dwarf"},
		AllowedSubraces:      []string{"hill-dwarf"},
		AllowedClasses:       []string{"fighter"},
		LockedBackgroundText: "A sellsword.",
	}
	for name, in := range map[string]game.CharacterCreationData{
		"race":    {RaceID: "elf", ClassID: "fighter"},
		"subrace": {RaceID: "dwarf", SubraceID: "mountain-dwarf", ClassID: "fighter"},
		"class":   {RaceID: "human", ClassID: "monk"},
	} {
		if _, err := rules.Apply(in); err == nil {
			t.Errorf("expected a disallowed %s to be rejected", name)
		}
	}
	out, err := rules.Apply(game.CharacterCreationData{RaceID: "dwarf", SubraceID: "hill-dwarf", ClassID: "fighter", Backstory: "mine"})
	if err != nil {
		t.Fatalf("expected an allowed character to pass, got %v", err)
	}
	if out.Backstory != "A sellsword." {
		t.Fatalf("expected the locked background to replace the player's, got %q", out.Backstory)
	}
}

func TestCampaignManifest_ExposesCreationMode(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, m := range reg.List() {
		modes[m.ID] = m.CharacterCreation.Mode
	}
	if modes["briarwell"] != "preset" || modes["test"] != "standard" {
		t.Fatalf("unexpected creation modes %v", modes)
	}
}

func TestValidate_PresetMustBeBuildableAndAllowed(t *testing.T) {
	base := func() *campaigns.CampaignDefinition {
		d := &campaigns.CampaignDefinition{
			ID: "c", Version: "1", Title: "C", Premise: "p",
			Party:       campaigns.PartyTemplate{StartingRoomID: "room"},
			Map:         campaigns.CampaignMap{Rooms: map[string]campaigns.CampaignRoom{"room": {Name: "Room"}}},
			StoryNodes:  map[string]campaigns.StoryNode{"start": {Mode: "ai_scene", Title: "Start"}},
			StartNodeID: "start",
		}
		d.CharacterCreation = campaigns.CharacterCreationRules{
			AllowedRaces:   []string{"human"},
			AllowedClasses: []string{"fighter"},
			Preset: &campaigns.PresetCharacter{
				RaceID: "human", ClassID: "fighter",
				AbilityScores: map[string]int{"str": 15, "dex": 14, "con": 13, "int": 12, "wis": 10, "cha": 8},
			},
		}
		return d
	}
	if err := base().Validate(); err != nil {
		t.Fatalf("expected a valid preset, got %v", err)
	}
	cases := map[string]func(*campaigns.CharacterCreationRules){
		"unbuildable allowed class": func(r *campaigns.CharacterCreationRules) { r.AllowedClasses = append(r.AllowedClasses, "rogue") },
		"preset race not allowed":   func(r *campaigns.CharacterCreationRules) { r.Preset.RaceID = "dwarf" },
		"missing scores":            func(r *campaigns.CharacterCreationRules) { r.Preset.AbilityScores = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := base()
			mutate(&d.CharacterCreation)
			if err := d.Validate(); err == nil {
				t.Fatalf("expected a validation error for %s", name)
			}
		})
	}
}
