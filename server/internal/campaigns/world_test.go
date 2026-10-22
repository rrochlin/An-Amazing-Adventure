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
