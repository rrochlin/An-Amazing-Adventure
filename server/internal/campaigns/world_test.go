package campaigns_test

import (
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func TestBriarwell_CastExistsAsCharactersAtTheirStart(t *testing.T) {
	_, g, _ := bootstrapBriarwell(t)
	want := map[string]string{
		"father": "family_cottage", "mother": "family_cottage", "sister": "family_cottage",
		"trader": "market_lane", "bess": "village_square", "miller": "old_mill", "bully": "meadow",
	}
	for id, room := range want {
		npc, ok := g.NPCs[id]
		if !ok {
			t.Fatalf("%s should exist as a character", id)
		}
		if npc.LocationID != room {
			t.Errorf("%s at %q, want %q", id, npc.LocationID, room)
		}
		if !contains(g.Rooms[room].Occupants, id) {
			t.Errorf("%s missing from %s's occupants", id, room)
		}
	}
	// Characters who arrive later aren't in the world yet.
	for _, id := range []string{"bandit_leader", "guild_master"} {
		if _, ok := g.NPCs[id]; ok {
			t.Errorf("%s should not exist before their scene", id)
		}
	}
	if !g.NPCs["father"].Friendly {
		t.Error("the father should be friendly")
	}
}

func TestBriarwell_PointsOfInterestAndDiscovery(t *testing.T) {
	_, g, _ := bootstrapBriarwell(t)
	if pois := g.Rooms["village_square"].PointsOfInterest; len(pois) != 2 || pois[0].ID != "stone_well" {
		t.Fatalf("village square points of interest = %#v", pois)
	}
	if !g.Discovered("family_cottage") || g.Discovered("meadow") {
		t.Fatalf("only the start room should be discovered, got %v", g.Campaign.DiscoveredRoomIDs)
	}
	if _, err := g.MovePlayer("east"); err != nil {
		t.Fatal(err)
	}
	if !g.Discovered("village_square") {
		t.Fatal("moving should discover the destination")
	}
	if err := g.PlaceCharacter(g.OwnerID, "woods_edge"); err != nil {
		t.Fatal(err)
	}
	if !g.Discovered("woods_edge") {
		t.Fatal("being placed (moved by the story) should discover the room")
	}

	view := g.BuildGameStateView(g.OwnerID, nil)
	if !view.Rooms["woods_edge"].Discovered || view.Rooms["meadow"].Discovered {
		t.Fatalf("room views should carry discovery, got %+v", view.Rooms["meadow"])
	}
	if len(view.Rooms["old_mill"].Occupants) != 1 || view.Rooms["old_mill"].Occupants[0].Name != "Hob the Miller" {
		t.Fatalf("old mill should show Hob, got %+v", view.Rooms["old_mill"].Occupants)
	}
}

func TestNarratorBriefSaysWhoTheCastAre(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	brief := campaigns.NarratorBrief(def, g)
	if !strings.Contains(brief, "Tomas (npc): The player's father.") {
		t.Fatalf("brief should say Tomas is the player's father:\n%s", brief)
	}
	if !strings.Contains(brief, "Dunstan (npc): A village boy") || !strings.Contains(brief, "Currently at: Meadow.") {
		t.Fatalf("brief should place Dunstan in the Meadow:\n%s", brief)
	}
}

func TestValidateRejectsBadStartLocationAndDuplicatePOI(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := reg.Get("briarwell")

	bad := *def
	bad.Actors = map[string]campaigns.ActorTemplate{"ghost": {Name: "Ghost", Kind: "npc", StartLocationID: "nowhere"}}
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "nowhere") {
		t.Fatalf("expected an unknown start location error, got %v", err)
	}

	dup := *def
	dup.Map.Rooms = map[string]campaigns.CampaignRoom{}
	for id, r := range def.Map.Rooms {
		dup.Map.Rooms[id] = r
	}
	room := dup.Map.Rooms["meadow"]
	room.PointsOfInterest = []game.PointOfInterest{{ID: "x", Name: "X"}, {ID: "x", Name: "X again"}}
	dup.Map.Rooms["meadow"] = room
	if err := dup.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate point of interest") {
		t.Fatalf("expected a duplicate point of interest error, got %v", err)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func TestBriarwell_CharactersHaveTheirSizes(t *testing.T) {
	_, g, _ := bootstrapBriarwell(t)
	owner, _ := g.OwnerCharacter()
	if owner.Scale != 0.8 {
		t.Fatalf("Wren is about ten: want 0.8, got %v", owner.Scale)
	}
	for id, want := range map[string]float64{"sister": 0.65, "bess": 0.6, "bully": 0.85, "father": 0} {
		if got := g.NPCs[id].Scale; got != want {
			t.Errorf("%s scale = %v, want %v", id, got, want)
		}
	}
	view := g.BuildGameStateView(g.OwnerID, nil)
	if view.Self.Scale != 0.8 {
		t.Fatalf("the view should carry the player's scale, got %v", view.Self.Scale)
	}
	reg, _ := campaigns.LoadEmbeddedRegistry()
	def, _ := reg.Get("briarwell")
	bad := *def
	bad.Actors = map[string]campaigns.ActorTemplate{"giant": {Name: "Giant", Kind: "npc", Scale: 9}}
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "scale") {
		t.Fatalf("an out-of-range scale should fail validation, got %v", err)
	}
}

func TestBriarwell_CottageScenesArePlayedIndoors(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	sm := g.Campaign.ActiveSceneMap
	if sm == nil || sm.ID != "cottage_interior" || sm.RoomID != "family_cottage" || sm.Name == "" {
		t.Fatalf("the morning errand should be played inside the cottage, got %+v", sm)
	}
	if view := g.BuildCampaignStateView(); view.ActiveSceneMap == nil || view.ActiveSceneMap.ID != "cottage_interior" {
		t.Fatalf("the client should be told the scene map, got %+v", view.ActiveSceneMap)
	}

	// Out of the door: the deeds are played on the village map.
	if _, err := campaigns.ResumeDialogueWithChoice(def, g, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
		t.Fatal(err)
	}
	if g.Campaign.ActiveNodeID != "village_deeds" || g.Campaign.ActiveSceneMap != nil {
		t.Fatalf("node %q should have no scene map, got %+v", g.Campaign.ActiveNodeID, g.Campaign.ActiveSceneMap)
	}
}

func TestValidateTiesSceneMapsToRooms(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := reg.Get("briarwell")
	withMaps := func(maps map[string]campaigns.SceneMap, nodeID, sceneMap, location string) campaigns.CampaignDefinition {
		d := *def
		d.Map.SceneMaps = maps
		d.StoryNodes = map[string]campaigns.StoryNode{}
		for id, n := range def.StoryNodes {
			d.StoryNodes[id] = n
		}
		n := d.StoryNodes[nodeID]
		n.SceneMap, n.LocationID = sceneMap, location
		d.StoryNodes[nodeID] = n
		return d
	}
	cottage := map[string]campaigns.SceneMap{"cottage_interior": {RoomID: "family_cottage", Name: "Inside"}}
	cases := map[string]struct {
		def  campaigns.CampaignDefinition
		want string
	}{
		"unknown scene map":   {withMaps(cottage, "gift_stall", "stall_tent", "market_lane"), `"stall_tent" does not exist`},
		"scene map elsewhere": {withMaps(cottage, "gift_stall", "cottage_interior", "market_lane"), `inside room "family_cottage"`},
		"scene map, no room":  {withMaps(map[string]campaigns.SceneMap{"cellar": {RoomID: "nowhere", Name: "Cellar"}}, "morning_errand", "", "family_cottage"), `"nowhere" does not exist`},
		"scene map, no name":  {withMaps(map[string]campaigns.SceneMap{"cellar": {RoomID: "family_cottage"}}, "morning_errand", "", "family_cottage"), "needs a name"},
	}
	for name, tc := range cases {
		if err := tc.def.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: expected error containing %q, got %v", name, tc.want, err)
		}
	}
}

func TestBriarwell_TheRaidChangesWhoIsWhere(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	advance := func(want string) {
		t.Helper()
		if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
			t.Fatal(err)
		}
		if g.Campaign.ActiveNodeID != want {
			t.Fatalf("expected %s, got %s", want, g.Campaign.ActiveNodeID)
		}
	}
	g.Campaign.ActiveNodeID = "birthday"
	g.Campaign.ActiveDialogue = nil
	g.Campaign.BoolFlags["gift_given"] = true
	advance("bandit_raid")
	// Red Corwin arrives in the world with the raid (playtest 6: rail moves
	// changed only his state, so no one could see him).
	if c, ok := g.NPCs["bandit_leader"]; !ok || c.LocationID != "village_square" || !contains(g.Rooms["village_square"].Occupants, "bandit_leader") {
		t.Fatalf("Red Corwin should be raiding the square, got %+v", g.NPCs["bandit_leader"])
	}

	g.Campaign.ActiveDialogue = nil
	g.Campaign.Labels["raid_response"] = "hid"
	advance("rescue")
	owner, _ := g.OwnerCharacter()
	here := g.Rooms[owner.LocationID]
	// Holloway kneels beside the player wherever they are; the family is gone
	// (the rescue must not reveal whether they live), and so are the bandits.
	if !contains(here.Occupants, "guild_master") || g.NPCs["guild_master"].LocationID != owner.LocationID {
		t.Fatalf("Holloway should be with the player in %s, occupants %v", owner.LocationID, here.Occupants)
	}
	for _, id := range []string{"father", "mother", "sister", "bandit_leader"} {
		if loc := g.NPCs[id].LocationID; loc != "" {
			t.Errorf("%s should be off-stage after the raid, is in %s", id, loc)
		}
		for roomID, r := range g.Rooms {
			if contains(r.Occupants, id) {
				t.Errorf("%s still listed in %s", id, roomID)
			}
		}
	}
}

func TestBriarwell_ScenesSetTheirLight(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	if g.Campaign.ActiveMood != "" {
		t.Fatalf("the morning is daylight, got %q", g.Campaign.ActiveMood)
	}
	g.Campaign.ActiveNodeID = "birthday"
	g.Campaign.ActiveDialogue = nil
	g.Campaign.BoolFlags["gift_given"] = true
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
		t.Fatal(err)
	}
	if g.Campaign.ActiveNodeID != "bandit_raid" || g.BuildCampaignStateView().ActiveMood != "fire" {
		t.Fatalf("the raid burns: node %s mood %q", g.Campaign.ActiveNodeID, g.Campaign.ActiveMood)
	}

	bad := *def
	bad.StoryNodes = map[string]campaigns.StoryNode{}
	for id, n := range def.StoryNodes {
		bad.StoryNodes[id] = n
	}
	n := bad.StoryNodes["rescue"]
	n.Mood = "disco"
	bad.StoryNodes["rescue"] = n
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), `mood "disco"`) {
		t.Fatalf("expected a mood error, got %v", err)
	}
}
