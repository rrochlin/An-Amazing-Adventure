package gm_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story/gm"
)

func briarwell(t *testing.T) (*campaigns.CampaignDefinition, *game.Game) {
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
	g.Campaign.ActiveNodeID = "village_deeds"
	return def, g
}

func node(def *campaigns.CampaignDefinition, id string) *campaigns.StoryNode {
	n := def.StoryNodes[id]
	return &n
}

func TestAllowedWorldEventsByControlLevel(t *testing.T) {
	def, _ := briarwell(t)
	bounded := gm.AllowedWorldEvents(node(def, "village_deeds"))
	controlled := gm.AllowedWorldEvents(node(def, "morning_errand"))
	for _, want := range []string{"move_player", "create_item", "damage_character"} {
		if !has(bounded, want) {
			t.Errorf("bounded scene should allow %s, got %v", want, bounded)
		}
	}
	for _, never := range []string{"create_character", "set_character_alive"} {
		if has(bounded, never) {
			t.Errorf("%s needs the scene's say-so, got %v", never, bounded)
		}
	}
	if has(controlled, "create_item") || !has(controlled, "move_player") {
		t.Errorf("controlled scene: only movement and handing items, got %v", controlled)
	}
	raid := *node(def, "bandit_raid")
	raid.StageDirections.AllowedWorldEvents = []string{"set_character_alive"}
	if !has(gm.AllowedWorldEvents(&raid), "set_character_alive") {
		t.Error("a scene can allow a death explicitly")
	}
	if n := len(gm.AllowedWorldEvents(nil)); n != len(campaigns.WorldEventTypes) {
		t.Errorf("no scene: everything allowed, got %d", n)
	}
}

func TestValidateDropsWhatTheSceneForbids(t *testing.T) {
	// Playtest 5 again: the engineer handed over the boy's horse (allowed:
	// items move in a bounded scene) and killed Tomas from prose (never
	// allowed unless the scene says so).
	def, _ := briarwell(t)
	d := gm.Decision{
		WorldEvents: []gm.WorldEvent{
			{Type: "move_player", Input: map[string]any{"room_name": "Meadow"}},
			{Type: "set_character_alive", Input: map[string]any{"character_name": "Tomas", "alive": false}},
			{Type: "create_character", Input: map[string]any{"name": "Pip"}},
		},
		Beats:     map[string]any{"deed_bully": "stopped", "deed_doll": "found", "gift": "ribbon", "stayed_hidden": true},
		OffScript: &gm.OffScript{Kind: "improvised"},
	}
	kept, dropped := gm.Validate(node(def, "village_deeds"), d)
	if len(kept.WorldEvents) != 1 || kept.WorldEvents[0].Type != "move_player" {
		t.Fatalf("kept %+v", kept.WorldEvents)
	}
	if len(kept.Beats) != 1 || kept.Beats["deed_bully"] != "stopped" {
		t.Fatalf("kept beats %v", kept.Beats)
	}
	joined := strings.Join(dropped, "\n")
	for _, want := range []string{`"set_character_alive" is not allowed`, `"create_character" is not allowed`, `beat "deed_doll" has no outcome found`, `beat "gift" is not one of this scene's beats`, `beat "stayed_hidden" is not one of this scene's beats`} {
		if !strings.Contains(joined, want) {
			t.Errorf("dropped should mention %q:\n%s", want, joined)
		}
	}
	res := gm.Resolution(kept)
	if res.Labels["deed_bully"] != "stopped" {
		t.Fatalf("resolution %+v", res)
	}
}

func TestSchemaIsNarrowedToTheScene(t *testing.T) {
	def, _ := briarwell(t)
	s := gm.Schema(node(def, "village_deeds"))
	raw, _ := json.Marshal(s)
	for _, want := range []string{`"deed_doll"`, `"returned"`, `"kept"`, `"const":"move_player"`, `"room_name"`, `is not enough`, `"narration"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("schema should contain %s", want)
		}
	}
	if strings.Contains(string(raw), `"create_character"`) {
		t.Error("schema must not offer create_character in a bounded scene")
	}
}

func TestBriefSaysPolicyWorldAndBeats(t *testing.T) {
	def, g := briarwell(t)
	if err := g.PlaceCharacter(g.OwnerID, "meadow"); err != nil {
		t.Fatal(err)
	}
	b := gm.Brief(def, g)
	for _, want := range []string{
		"This scene is BOUNDED", "Place: Meadow", "Also here: Dunstan", "Stop at the next decision point",
		"Allowed world changes in this scene: create_item", "deed_bully (Dunstan bullying", `"stopped" when The player makes Dunstan stop`,
		"never create or give coins",
	} {
		if !strings.Contains(b, want) {
			t.Errorf("brief should say %q", want)
		}
	}
	// The cast list says who exists and where (so no one is invented); the
	// world slice lists only who is actually here.
	here := b[strings.Index(b, "Also here:"):]
	here = here[:strings.Index(here, "\n")]
	if strings.Contains(here, "Hob") || !strings.Contains(here, "Dunstan") {
		t.Errorf("world slice should list only the room's occupants, got %q", here)
	}
	g.Campaign.ActiveNodeID = "morning_errand"
	if !strings.Contains(gm.Brief(def, g), "This scene is CONTROLLED") {
		t.Error("controlled policy")
	}
}

func has(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
