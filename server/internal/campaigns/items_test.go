package campaigns_test

import (
	"strings"
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
	// Coins come from the deeds themselves (see DeedsPayAndTakeTheirItems).
	giveCoin(t, g, "smooth stone")

	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_bully": "stopped", "deed_miller": "took_bribe"},
	}); err != nil {
		t.Fatal(err)
	}
	walkToStall(t, def, g)
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
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
	walkToStall(t, def, g)
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
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

// Playtest 5: the engineer handed over a coin twice for one deed, kept the
// returned doll in the pack, and the rescue teleported a hidden child to
// the woods. Deeds now pay and take things themselves.
func TestBriarwell_DeedsPayAndTakeTheirItems(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "I'll do it", "village_deeds")
	doll := game.NewItem("rag doll", "Bess's doll")
	if err := g.AddItem(doll); err != nil {
		t.Fatal(err)
	}
	if err := g.GiveItemToPlayer(doll.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{Labels: map[string]string{"deed_bully": "stopped"}}); err != nil {
		t.Fatal(err)
	}
	if got := inventoryNames(g); len(got) != 2 || got[1] != "copper coin" {
		t.Fatalf("stopping Dunstan pays one coin, got %v", got)
	}
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{Labels: map[string]string{"deed_doll": "returned"}}); err != nil {
		t.Fatal(err)
	}
	if got := inventoryNames(g); len(got) != 2 || got[0] != "copper coin" || got[1] != "copper coin" {
		t.Fatalf("returning the doll hands it over and pays a coin, got %v", got)
	}
	// Two deeds done: walk to the stall, where both coins are spent.
	walkToStall(t, def, g)
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
		t.Fatal(err)
	}
	if g.Campaign.ActiveNodeID != "gift_stall" {
		t.Fatalf("expected gift_stall, got %q", g.Campaign.ActiveNodeID)
	}
	chooseOption(t, def, g, "sweets", "birthday")
	if got := inventoryNames(g); len(got) != 1 || got[0] != "honey sweets" {
		t.Fatalf("the stall takes both coins, got %v", got)
	}
}

func TestBriarwell_RescueHappensWhereYouAre(t *testing.T) {
	reg, _ := campaigns.LoadEmbeddedRegistry()
	def, _ := reg.Get("briarwell")
	for _, id := range []string{"rescue", "road_to_guild"} {
		if loc := def.StoryNodes[id].LocationID; loc != "" {
			t.Errorf("%s should not move the player (it did, to %q): a child hiding under the bed is rescued there", id, loc)
		}
	}
	if !strings.Contains(def.StoryNodes["rescue"].IntroText, "what is left of Briarwell") {
		t.Error("the rescue intro should not name a place")
	}
}
