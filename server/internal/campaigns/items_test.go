package campaigns_test

import (
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func inventoryNames(g *game.Game) []string {
	owner, _ := g.OwnerCharacter()
	names := []string{}
	for _, id := range owner.Inventory {
		names = append(names, g.Items[id].Name)
	}
	return names
}

func giveCoin(t *testing.T, g *game.Game, name string) {
	t.Helper()
	item := game.NewItem(name, "")
	if err := g.AddItem(item); err != nil {
		t.Fatal(err)
	}
	if err := g.GiveItemToPlayer(item.ID); err != nil {
		t.Fatal(err)
	}
}

// Playtest 4 (session a2d428dc): choosing the ribbon never gave the ribbon,
// and the map left the player in the Village Square through the stall and
// the birthday.
func TestBriarwell_GiftIsAnItemAndScenesMoveThePlayer(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	if loc(g) != "family_cottage" {
		t.Fatalf("the errand starts at home, got %q", loc(g))
	}
	chooseOption(t, def, g, "I'll do it", "village_deeds")
	if err := g.PlaceCharacter(g.OwnerID, "meadow"); err != nil {
		t.Fatal(err)
	}
	giveCoin(t, g, "copper coin")
	giveCoin(t, g, "Hob's silver coin")
	giveCoin(t, g, "smooth stone")

	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_bully": "stopped", "deed_miller": "took_bribe"},
	}); err != nil {
		t.Fatal(err)
	}
	if g.Campaign.ActiveNodeID != "gift_stall" || loc(g) != "market_lane" {
		t.Fatalf("the stall scene should put the player on Market Lane, got node %q at %q", g.Campaign.ActiveNodeID, loc(g))
	}

	chooseOption(t, def, g, "ribbon", "birthday")
	if got := inventoryNames(g); len(got) != 2 || got[0] != "smooth stone" || got[1] != "cornflower ribbon" {
		t.Fatalf("Garrick takes both coins and hands over the ribbon, got %v", got)
	}
	if loc(g) != "family_cottage" {
		t.Fatalf("the birthday happens at home, got %q", loc(g))
	}
	if !g.Discovered("market_lane") {
		t.Fatal("being taken somewhere by the story discovers it")
	}

	chooseAndStay(t, def, g, "Give Ros")
	if got := inventoryNames(g); len(got) != 1 || got[0] != "smooth stone" {
		t.Fatalf("giving Ros the gift takes it, got %v", got)
	}
}

func TestTakeItemIsHarmlessWhenTheItemIsMissing(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "I'll do it", "village_deeds")
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_bully": "stopped", "deed_doll": "returned"},
	}); err != nil {
		t.Fatal(err)
	}
	// No coins in the pack: the purchase still goes through.
	chooseOption(t, def, g, "wooden bird", "birthday")
	if got := inventoryNames(g); len(got) != 1 || got[0] != "carved wooden bird" {
		t.Fatalf("got %v", got)
	}
}

func loc(g *game.Game) string {
	owner, _ := g.OwnerCharacter()
	return owner.LocationID
}
