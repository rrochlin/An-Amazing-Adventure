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
	kept, dropped := gm.Validate(node(def, "village_deeds"), nil, d)
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
	s := gm.Schema(node(def, "village_deeds"), nil)
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

func TestSettledBeatsAreFinal(t *testing.T) {
	def, g := briarwell(t)
	g.Campaign.ActiveObjectives = append(g.Campaign.ActiveObjectives,
		game.ObjectiveState{ID: "miller_secret", Status: "active"},
		game.ObjectiveState{ID: "return_doll", Status: "active"})

	// Open: offered, with what it pays.
	brief := gm.Brief(def, g)
	if !strings.Contains(brief, "deed_miller (") || !strings.Contains(brief, "Settling it pays the player one copper coin") {
		t.Fatalf("an open beat should say what it pays:\n%s", brief)
	}

	// Playtest 6: the bribe was taken, then the player told the wife too,
	// and the beat was rewritten (with a second coin in the prose).
	g.Campaign.Labels["deed_miller"] = "took_bribe"
	kept, dropped := gm.Validate(node(def, "village_deeds"), g, gm.Decision{Beats: map[string]any{"deed_miller": "told_wife", "deed_doll": "returned"}})
	if _, ok := kept.Beats["deed_miller"]; ok || kept.Beats["deed_doll"] != "returned" {
		t.Fatalf("a settled beat must not be settled again, got %v", kept.Beats)
	}
	if !strings.Contains(strings.Join(dropped, "\n"), `"deed_miller" was already settled as took_bribe`) {
		t.Fatalf("dropped %v", dropped)
	}
	brief = gm.Brief(def, g)
	if !strings.Contains(brief, "Already settled, and final") || !strings.Contains(brief, "deed_miller = took_bribe") {
		t.Fatalf("the brief should list settled beats:\n%s", brief)
	}
	raw, _ := json.Marshal(gm.Schema(node(def, "village_deeds"), g))
	if strings.Contains(string(raw), `"deed_miller"`) || !strings.Contains(string(raw), `"deed_doll"`) {
		t.Fatalf("the schema should offer only open beats: %s", raw)
	}
}

func TestValidateDropsAMoveToWhereThePlayerIs(t *testing.T) {
	def, g := briarwell(t)
	owner, _ := g.OwnerCharacter()
	here, _ := g.GetRoom(owner.LocationID)
	kept, dropped := gm.Validate(node(def, "village_deeds"), g, gm.Decision{WorldEvents: []gm.WorldEvent{
		{Type: "move_player", Input: map[string]any{"room_name": strings.ToUpper(here.Name)}},
		{Type: "move_player", Input: map[string]any{"room_name": "Meadow"}},
	}})
	if len(kept.WorldEvents) != 1 || kept.WorldEvents[0].Input["room_name"] != "Meadow" {
		t.Fatalf("kept %+v", kept.WorldEvents)
	}
	if len(dropped) != 1 || !strings.Contains(dropped[0], "already in "+here.Name) {
		t.Fatalf("dropped %v", dropped)
	}
}

func TestParseDecisionReadsQuotedValues(t *testing.T) {
	// Playtest 6, turn 8: beats came back as "" and the decision half-failed.
	d, notes, err := gm.ParseDecision(`{"beats":"","world_events":[{"type":"move_player","input":{"room_name":"Meadow"}}]}`)
	if err != nil || len(d.WorldEvents) != 1 || len(d.Beats) != 0 || len(notes) != 1 {
		t.Fatalf("empty string beats: %+v %v %v", d, notes, err)
	}
	d, notes, err = gm.ParseDecision(`{"beats":"{\"deed_doll\":\"kept\"}","world_events":"[]"}`)
	if err != nil || d.Beats["deed_doll"] != "kept" || len(d.WorldEvents) != 0 || len(notes) != 2 {
		t.Fatalf("quoted JSON: %+v %v %v", d, notes, err)
	}
	if _, _, err := gm.ParseDecision(`{"beats":"deed_doll=kept"}`); err == nil {
		t.Fatal("an unreadable value should be an error")
	}
	if _, _, err := gm.ParseDecision(`{"beats":{"deed_doll":["kept"]},"world_events":[{"type":"move_player","input":"Meadow"}]}`); err == nil {
		t.Fatal("a wrongly shaped event should be an error, not half applied")
	}
}

func TestValidateDropsCreatingWhatIsAlreadyHere(t *testing.T) {
	def, g := briarwell(t)
	owner, _ := g.OwnerCharacter()
	doll := game.Item{ID: "doll-1", Name: "Rag Doll"}
	g.Items[doll.ID] = doll
	room := g.Rooms[owner.LocationID]
	room.Items = append(room.Items, doll.ID)
	g.Rooms[owner.LocationID] = room
	kept, dropped := gm.Validate(node(def, "village_deeds"), g, gm.Decision{WorldEvents: []gm.WorldEvent{
		{Type: "create_item", Input: map[string]any{"name": "rag doll", "description": "again"}},
		{Type: "give_item_to_player", Input: map[string]any{"item_name": "Rag Doll"}},
	}})
	if len(kept.WorldEvents) != 1 || kept.WorldEvents[0].Type != "give_item_to_player" {
		t.Fatalf("kept %+v", kept.WorldEvents)
	}
	if len(dropped) != 1 || !strings.Contains(dropped[0], "already exists") {
		t.Fatalf("dropped %v", dropped)
	}
}
