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

func TestDamageHurtsButDoesNotKillUnlessTheSceneAllowsIt(t *testing.T) {
	def, g := briarwell(t)
	dmg := func(who string, amount float64) gm.WorldEvent {
		return gm.WorldEvent{Type: "damage_character", Input: map[string]any{"character_name": who, "amount": amount}}
	}
	kept, dropped := gm.Validate(node(def, "bandit_raid"), g, gm.Decision{WorldEvents: []gm.WorldEvent{
		dmg("player", 150), dmg("Tomas", 30), dmg("Tomas", 500),
	}})
	if len(kept.WorldEvents) != 3 || kept.WorldEvents[0].Input["amount"] != float64(99) ||
		kept.WorldEvents[1].Input["amount"] != float64(30) || kept.WorldEvents[2].Input["amount"] != float64(99) {
		t.Fatalf("lethal damage should be lowered to leave 1, kept %+v", kept.WorldEvents)
	}
	if len(dropped) != 2 || !strings.Contains(dropped[0], "doesn't allow deaths") {
		t.Fatalf("dropped %v", dropped)
	}

	// A scene that allows deaths keeps the full amount.
	deadly := *node(def, "bandit_raid")
	deadly.StageDirections.AllowedWorldEvents = []string{"set_character_alive"}
	kept, _ = gm.Validate(&deadly, g, gm.Decision{WorldEvents: []gm.WorldEvent{dmg("Tomas", 500)}})
	if kept.WorldEvents[0].Input["amount"] != float64(500) {
		t.Fatalf("an authored death should stand, kept %+v", kept.WorldEvents)
	}
}

func TestBriefSaysWhatTheCharactersHereKnow(t *testing.T) {
	def, g := briarwell(t)
	if err := g.PlaceCharacter(g.OwnerID, "village_square"); err != nil {
		t.Fatal(err)
	}
	brief := gm.Brief(def, g)
	if !strings.Contains(brief, "Little Bess knows: She dropped her rag doll in the Meadow") {
		t.Fatalf("Bess is here, so the brief should say what she knows:\n%s", brief)
	}
	if err := g.PlaceCharacter(g.OwnerID, "old_mill"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(gm.Brief(def, g), "Little Bess knows") {
		t.Fatal("need to know: not when Bess isn't here")
	}
}

func TestStagingKeepsWhatTheRoomAllows(t *testing.T) {
	def, g := briarwell(t)
	if err := g.PlaceCharacter(g.OwnerID, "village_square"); err != nil {
		t.Fatal(err)
	}
	d := gm.Decision{Staging: []gm.Cue{
		{Line: 1, Who: "Little Bess", MoveTo: "stone_well"},
		{Line: 2, Who: "player", Emote: "bow"},
		{Line: 0, Who: "Dunstan", Emote: "laugh"},         // in the meadow, not here
		{Line: 0, Who: "Little Bess", MoveTo: "hearth"},   // a mark of another room
		{Line: 0, Who: "Little Bess", Emote: "pirouette"}, // no such emote
		{Line: -3, Who: "little bess", Emote: "hop"},      // any case; line clamped
	}}
	kept, dropped := gm.Validate(node(def, "village_deeds"), g, d)
	if len(kept.Staging) != 3 {
		t.Fatalf("kept %+v (dropped %v)", kept.Staging, dropped)
	}
	if kept.Staging[0].Who != "bess" || kept.Staging[0].MoveTo != "stone_well" {
		t.Errorf("Bess's cue should name her ID: %+v", kept.Staging[0])
	}
	if kept.Staging[1].Who != gm.PlayerCue || kept.Staging[2].Line != 0 {
		t.Errorf("player cue and clamped line: %+v", kept.Staging[1:])
	}
	if len(dropped) != 3 {
		t.Errorf("dropped %v", dropped)
	}

	events := gm.ApplyStaging(g, kept.Staging, 2)
	if bess := g.NPCs["bess"]; bess.MarkHere() != "stone_well" {
		t.Errorf("Bess should stand at the well, got %q", bess.MarkHere())
	}
	if len(events) != 3 || events[0].Type != "cue" || events[0].Character != "bess" || events[0].Mark != "stone_well" {
		t.Fatalf("cue events %+v", events)
	}
	if events[1].Line != 1 {
		t.Errorf("a cue past the last paragraph plays on the last: %+v", events[1])
	}
	if !strings.Contains(gm.Brief(def, g), "Little Bess at stone_well") {
		t.Error("the brief should say where people stand")
	}

	// Moving her anywhere leaves the mark behind.
	if err := g.MoveNPC("bess", "meadow"); err != nil {
		t.Fatal(err)
	}
	if m := g.NPCs["bess"].MarkHere(); m != "" {
		t.Errorf("a mark only holds in its room, got %q", m)
	}
}

func TestSchemaStagesOnlyWhoIsHere(t *testing.T) {
	def, g := briarwell(t)
	if err := g.PlaceCharacter(g.OwnerID, "village_square"); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(gm.Schema(node(def, "village_deeds"), g))
	s := string(raw)
	for _, want := range []string{`"staging"`, `"Little Bess"`, `"stone_well"`, `"startle"`} {
		if !strings.Contains(s, want) {
			t.Errorf("schema should offer %s", want)
		}
	}
	if strings.Contains(s, `"Dunstan"`) || strings.Contains(s, `"hearth"`) {
		t.Error("only who and what is here")
	}
}

func TestValidateDropsMovingSomeoneWhereTheyAre(t *testing.T) {
	def, g := briarwell(t)
	d := gm.Decision{WorldEvents: []gm.WorldEvent{
		{Type: "move_character", Input: map[string]any{"character_name": "Little Bess", "room_name": "Village Square"}},
		{Type: "move_character", Input: map[string]any{"character_name": "Little Bess", "room_name": "Meadow"}},
	}}
	kept, dropped := gm.Validate(node(def, "village_deeds"), g, d)
	if len(kept.WorldEvents) != 1 || kept.WorldEvents[0].Input["room_name"] != "Meadow" {
		t.Fatalf("kept %+v", kept.WorldEvents)
	}
	if len(dropped) != 1 || !strings.Contains(dropped[0], "already in Village Square") {
		t.Errorf("dropped %v", dropped)
	}
}

func TestStagingCanWalkSomeoneUpToAnother(t *testing.T) {
	def, g := briarwell(t)
	if err := g.PlaceCharacter(g.OwnerID, "village_square"); err != nil {
		t.Fatal(err)
	}
	d := gm.Decision{Staging: []gm.Cue{
		{Line: 0, Who: "player", Toward: "Little Bess"},
		{Line: 1, Who: "Little Bess", Toward: "player", Emote: "hop"},
		{Line: 1, Who: "Little Bess", Toward: "Dunstan"}, // not here
		{Line: 1, Who: "player", Toward: "player"},       // nobody to walk to
	}}
	kept, dropped := gm.Validate(node(def, "village_deeds"), g, d)
	if len(kept.Staging) != 2 || kept.Staging[0].Toward != "bess" || kept.Staging[1].Toward != gm.PlayerCue {
		t.Fatalf("kept %+v dropped %v", kept.Staging, dropped)
	}
	events := gm.ApplyStaging(g, kept.Staging, 3)
	if events[0].Toward != "bess" || events[0].Character != gm.PlayerCue {
		t.Errorf("cue event %+v", events[0])
	}
}

func TestIndoorsTheInteriorsMarksAreOffered(t *testing.T) {
	// Playtest 11: the player hid behind the barrel by the door, and the only
	// marks the game master could name were the hearth, door and gate.
	def, g := briarwell(t)
	g.Campaign.ActiveNodeID = "bandit_raid"
	if g.Campaign.ActiveSceneMap == nil || g.Campaign.ActiveSceneMap.ID != "cottage_interior" {
		t.Fatalf("setup: the opening is played indoors, got %+v", g.Campaign.ActiveSceneMap)
	}
	raw, _ := json.Marshal(gm.Schema(node(def, "bandit_raid"), g))
	if !strings.Contains(string(raw), `"barrel"`) || strings.Contains(string(raw), `"garden_gate"`) {
		t.Errorf("indoors: the interior's marks, not the room's: %s", raw)
	}
	kept, _ := gm.Validate(node(def, "bandit_raid"), g, gm.Decision{Staging: []gm.Cue{{Who: "player", MoveTo: "barrel"}}})
	if len(kept.Staging) != 1 {
		t.Errorf("the barrel is a mark indoors")
	}
	if !strings.Contains(gm.Brief(def, g), "barrel (Barrel)") {
		t.Error("the brief lists the interior's marks")
	}

	if err := g.PlaceCharacter(g.OwnerID, "village_square"); err != nil {
		t.Fatal(err)
	}
	if raw, _ := json.Marshal(gm.Schema(node(def, "bandit_raid"), g)); strings.Contains(string(raw), `"barrel"`) {
		t.Error("outside, the room's own marks")
	}
}

func TestTheGameMasterCantMoveThePlayerWhereTheSceneForbids(t *testing.T) {
	def, g := briarwell(t)
	g.Campaign.BlockedRooms = def.StoryNodes["village_deeds"].BlockedRooms
	kept, dropped := gm.Validate(node(def, "village_deeds"), g, gm.Decision{WorldEvents: []gm.WorldEvent{
		{Type: "move_player", Input: map[string]any{"room_name": "Woods' Edge"}},
	}})
	if len(kept.WorldEvents) != 0 || len(dropped) != 1 || !strings.Contains(dropped[0], "Mother told you") {
		t.Fatalf("kept %+v dropped %v", kept.WorldEvents, dropped)
	}
	if !strings.Contains(gm.Brief(def, g), "can't go to these places in this scene: Woods' Edge") {
		t.Error("the brief says where the player can't go")
	}
}

func TestTheBriefNamesPeopleAsThePlayerDoes(t *testing.T) {
	def, g := briarwell(t)
	if err := g.PlaceCharacter(g.OwnerID, "family_cottage"); err != nil {
		t.Fatal(err)
	}
	brief := gm.Brief(def, g)
	if !strings.Contains(brief, "whom the player calls Father: write their lines as Father:") {
		t.Errorf("Tomas's lines should be Father's (the panel saw FATHER and TOMAS):\n%s", brief)
	}
	if strings.Contains(brief, "handled by the story") {
		t.Error("the narration shouldn't learn to say 'the story' pays")
	}
}
