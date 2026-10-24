package campaigns_test

import (
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Each scene's intro must appear when that scene starts, and nowhere else.
// Playtest 2 opened with the rescue scene's "shouting" intro because the
// intros were attached to the wrong nodes.
func TestBriarwell_SceneIntrosBelongToTheirScenes(t *testing.T) {
	def, _, history := bootstrapBriarwell(t)
	var opening strings.Builder
	for _, m := range history {
		opening.WriteString(m.Content)
	}
	if strings.Contains(opening.String(), "shouting") {
		t.Fatalf("the opening must not contain the rescue scene's intro:\n%s", opening.String())
	}
	want := map[string]string{
		"morning_errand": "",
		"village_deeds":  "Little Bess is crying",
		"gift_stall":     "Two coins clink",
		"birthday":       "The sun is low and gold",
		"bandit_raid":    "You wake to screaming",
		"rescue":         "When the shouting finally stops",
		"road_to_guild":  "Dawn finds you",
	}
	for nodeID, phrase := range want {
		got := def.StoryNodes[nodeID].IntroText
		if (phrase == "" && got != "") || !strings.Contains(got, phrase) {
			t.Errorf("%s intro = %q, want it to contain %q", nodeID, got, phrase)
		}
	}
}

func objectiveStatus(g *game.Game, id string) string {
	for _, o := range g.Campaign.ActiveObjectives {
		if o.ID == id {
			return o.Status
		}
	}
	return ""
}

func TestBriarwell_DeedObjectivesTickOffAndUntouchedOneIsLeftUnfinished(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "I'll do it", "village_deeds")

	first, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_bully": "stopped"},
	})
	if err != nil {
		t.Fatalf("first deed: %v", err)
	}
	if first.Transition.Transitioned {
		t.Fatal("one deed should not finish the errand")
	}
	if objectiveStatus(g, "meadow_bully") != "completed" {
		t.Fatalf("expected Trouble in the Meadow completed, got %q", objectiveStatus(g, "meadow_bully"))
	}
	if len(first.Events) != 1 || first.Events[0].Message != "Objective complete: Trouble in the Meadow" {
		t.Fatalf("expected a completion event for the bully deed, got %#v", first.Events)
	}

	second, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_miller": "told_wife"},
	})
	if err != nil {
		t.Fatalf("second deed: %v", err)
	}
	if second.Transition.ToNodeID != "gift_stall" {
		t.Fatalf("expected two deeds to move on to the gift stall, got %#v", second.Transition)
	}
	for id, want := range map[string]string{"miller_secret": "completed", "return_doll": "unfinished", "earn_coins": "completed"} {
		if got := objectiveStatus(g, id); got != want {
			t.Errorf("objective %s = %q, want %q", id, got, want)
		}
	}
	var messages []string
	for _, e := range second.Events {
		messages = append(messages, e.Message)
	}
	joined := strings.Join(messages, " | ")
	for _, want := range []string{"Objective complete: The Miller's Secret", "Left unfinished: Bess's Lost Doll", "Objective updated: Choose a gift"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected event %q, got %s", want, joined)
		}
	}
}

// Playtest 6: the player found Bess's doll, then the second deed moved the
// story on. The doll quest is left unfinished (not failed), and the doll,
// which nothing later uses, leaves the player's pack and the world.
func TestBriarwell_UnfinishedQuestTakesItsItems(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "I'll do it", "village_deeds")
	owner, _ := g.OwnerCharacter()
	give := func(id, name string) {
		g.Items[id] = game.Item{ID: id, Name: name}
		owner.Inventory = append(owner.Inventory, id)
	}
	give("doll-1", "Marta the Rag Doll")
	give("bread-1", "Heel of Bread")
	g.Players[g.OwnerID] = owner
	g.Items["doll-2"] = game.Item{ID: "doll-2", Name: "Rag Doll"}
	meadow := g.Rooms["meadow"]
	meadow.Items = append(meadow.Items, "doll-2")
	g.Rooms["meadow"] = meadow

	turn, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_bully": "stopped", "deed_miller": "took_bribe"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if turn.Transition.ToNodeID != "gift_stall" || objectiveStatus(g, "return_doll") != "unfinished" {
		t.Fatalf("node %q, return_doll %q", g.Campaign.ActiveNodeID, objectiveStatus(g, "return_doll"))
	}
	owner, _ = g.OwnerCharacter()
	var carried []string
	for _, id := range owner.Inventory {
		carried = append(carried, g.Items[id].Name)
	}
	if strings.Contains(strings.ToLower(strings.Join(carried, ",")), "doll") || !contains(carried, "Heel of Bread") {
		t.Fatalf("the doll should go and the bread stay, carrying %v", carried)
	}
	if _, ok := g.Items["doll-2"]; ok || len(g.Rooms["meadow"].Items) != 0 {
		t.Fatalf("the doll in the meadow should be gone too, meadow %v", g.Rooms["meadow"].Items)
	}
	var lost bool
	for _, e := range turn.Events {
		lost = lost || (e.Type == "item_lost" && strings.Contains(e.Message, "Marta the Rag Doll"))
	}
	if !lost {
		t.Fatalf("the player should be told the doll is gone, events %+v", turn.Events)
	}
}

// The platform lets a campaign pause before moving on: with player_in_room
// in the completion rules, the stall waits until the player walks to Market
// Lane, and an unfinished deed can still be finished on the way.
func TestFreeRoamBufferBeforeTheNextScene(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	buffered := *def
	buffered.StoryNodes = map[string]campaigns.StoryNode{}
	for id, n := range def.StoryNodes {
		buffered.StoryNodes[id] = n
	}
	deeds := buffered.StoryNodes["village_deeds"]
	deeds.CompletionRules = append([]campaigns.TransitionRule(nil), deeds.CompletionRules...)
	for i := range deeds.CompletionRules {
		r := deeds.CompletionRules[i]
		r.Conditions = append(append([]campaigns.Condition(nil), r.Conditions...), campaigns.Condition{Type: "player_in_room", RoomID: "market_lane"})
		deeds.CompletionRules[i] = r
	}
	buffered.StoryNodes["village_deeds"] = deeds
	if err := buffered.Validate(); err != nil {
		t.Fatalf("a buffered campaign should validate: %v", err)
	}
	def = &buffered

	chooseOption(t, def, g, "I'll do it", "village_deeds")
	turn, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_bully": "stopped", "deed_miller": "took_bribe"},
	})
	if err != nil || turn.Transition.Transitioned {
		t.Fatalf("two deeds alone shouldn't move on yet: %+v %v", turn.Transition, err)
	}
	// Free roam: the doll can still be returned, and pays.
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{Labels: map[string]string{"deed_doll": "returned"}}); err != nil {
		t.Fatal(err)
	}
	if objectiveStatus(g, "return_doll") != "completed" || g.Campaign.ActiveNodeID != "village_deeds" {
		t.Fatalf("the doll should still count during the buffer: %q in %s", objectiveStatus(g, "return_doll"), g.Campaign.ActiveNodeID)
	}
	if err := g.PlaceCharacter(g.OwnerID, "market_lane"); err != nil {
		t.Fatal(err)
	}
	turn, err = campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{})
	if err != nil || turn.Transition.ToNodeID != "gift_stall" {
		t.Fatalf("arriving at Market Lane should open the stall: %+v %v", turn.Transition, err)
	}
}
