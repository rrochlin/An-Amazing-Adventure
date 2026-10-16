package campaigns_test

import (
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func bootstrapBriarwell(t *testing.T) (*campaigns.CampaignDefinition, *game.Game, []game.ChatMessage) {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("briarwell")
	if !ok {
		t.Fatal("expected briarwell campaign")
	}
	g, history, err := campaigns.BootstrapGame(
		def,
		"session-briarwell",
		"user-1",
		game.NewCharacter("Wren", ""),
		game.CharacterCreationData{Name: "Wren", RaceID: "human", ClassID: "rogue"},
	)
	if err != nil {
		t.Fatalf("bootstrap game: %v", err)
	}
	return def, g, history
}

// chooseOption picks the pending dialogue choice whose text contains want,
// then advances the campaign and asserts it lands on wantNode.
func chooseOption(t *testing.T, def *campaigns.CampaignDefinition, g *game.Game, want, wantNode string) string {
	t.Helper()
	d := g.Campaign.ActiveDialogue
	if d == nil || !d.AwaitingChoice {
		t.Fatalf("node %q: expected dialogue awaiting a choice, got %#v", g.Campaign.ActiveNodeID, d)
	}
	choiceID := -1
	for _, c := range d.PendingChoices {
		if strings.Contains(c.Text, want) {
			choiceID = c.ID
		}
	}
	if choiceID < 0 {
		t.Fatalf("node %q: no choice containing %q in %#v", g.Campaign.ActiveNodeID, want, d.PendingChoices)
	}
	text, err := campaigns.ResumeDialogueWithChoice(def, g, choiceID)
	if err != nil {
		t.Fatalf("resume with %q: %v", want, err)
	}
	advanceTo(t, def, g, wantNode)
	return text
}

// chooseAndStay picks a choice that should not end the scene: the dialogue
// continues to another choice and the campaign stays on the same node.
func chooseAndStay(t *testing.T, def *campaigns.CampaignDefinition, g *game.Game, want string) string {
	t.Helper()
	node := g.Campaign.ActiveNodeID
	d := g.Campaign.ActiveDialogue
	choiceID := -1
	for _, c := range d.PendingChoices {
		if strings.Contains(c.Text, want) {
			choiceID = c.ID
		}
	}
	if choiceID < 0 {
		t.Fatalf("node %q: no choice containing %q in %#v", node, want, d.PendingChoices)
	}
	text, err := campaigns.ResumeDialogueWithChoice(def, g, choiceID)
	if err != nil {
		t.Fatalf("resume with %q: %v", want, err)
	}
	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if result.Transitioned || g.Campaign.ActiveNodeID != node || !g.Campaign.ActiveDialogue.AwaitingChoice {
		t.Fatalf("expected to stay in %q awaiting another choice, got %#v", node, result)
	}
	return text
}

func advanceTo(t *testing.T, def *campaigns.CampaignDefinition, g *game.Game, wantNode string) {
	t.Helper()
	from := g.Campaign.ActiveNodeID
	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance from %q: %v", from, err)
	}
	if !result.Transitioned || result.ToNodeID != wantNode {
		t.Fatalf("advance from %q: expected %q, got %#v", from, wantNode, result)
	}
}

func objectiveStatuses(g *game.Game) map[string]string {
	out := map[string]string{}
	for _, o := range g.Campaign.ActiveObjectives {
		out[o.ID] = o.Status
	}
	return out
}

func TestBriarwell_BootstrapOpensFatherErrand(t *testing.T) {
	_, g, history := bootstrapBriarwell(t)
	if g.Campaign.ActiveNodeID != "morning_errand" {
		t.Fatalf("expected morning_errand, got %q", g.Campaign.ActiveNodeID)
	}
	if len(g.Campaign.ActiveDialogue.PendingChoices) != 2 {
		t.Fatalf("expected two errand choices, got %#v", g.Campaign.ActiveDialogue)
	}
	joined := ""
	for _, m := range history {
		joined += m.Content + "\n"
	}
	// Quoted dialogue must survive the string table round trip intact.
	if !strings.Contains(joined, "Father: It's your sister's birthday, and I've no coin to spare for a present.") {
		t.Fatalf("expected father's quoted line in opening history, got:\n%s", joined)
	}
	if strings.Contains(joined, `""`) {
		t.Fatalf("doubled CSV quotes leaked into dialogue:\n%s", joined)
	}
}

func TestBriarwell_FullPlaythrough(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)

	chooseOption(t, def, g, "I'll do it", "village_deeds")
	if g.Campaign.Labels["attitude"] != "eager" {
		t.Fatalf("expected attitude=eager, got %#v", g.Campaign.Labels)
	}

	// One deed is not enough coin.
	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_doll": "returned"},
	})
	if result, err := campaigns.AdvanceCampaign(def, g); err != nil || result.Transitioned {
		t.Fatalf("expected no transition after one deed, got %#v, %v", result, err)
	}
	// A second deed, good or bad, completes the errand.
	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_bully": "joined"},
	})
	advanceTo(t, def, g, "gift_stall")

	chooseOption(t, def, g, "wooden bird", "birthday")
	if g.Campaign.Labels["gift"] != "carved_bird" {
		t.Fatalf("expected gift=carved_bird, got %#v", g.Campaign.Labels)
	}

	// Giving the gift doesn't end the night; going to bed does.
	text := chooseAndStay(t, def, g, "Give Ros")
	if !strings.Contains(text, "smoke rising from the woods") {
		t.Fatalf("expected smoke foreshadowing after the gift, got %q", text)
	}
	chooseOption(t, def, g, "Go to bed", "bandit_raid")
	if got := g.Campaign.ActorStates["bandit_leader"].CurrentRailStepID; got != "raiding" {
		t.Fatalf("expected bandit_leader at raiding step, got %q", got)
	}

	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{
		Labels: map[string]string{"raid_response": "searched_family"},
	})
	advanceTo(t, def, g, "rescue")
	if !g.Campaign.BoolFlags["saw_ros_taken"] {
		t.Fatalf("expected saw_ros_taken flag, got %#v", g.Campaign.BoolFlags)
	}
	if got := g.Campaign.ActorStates["guild_master"].CurrentRailStepID; got != "arrived" {
		t.Fatalf("expected guild_master arrived, got %q", got)
	}

	chooseOption(t, def, g, "Take his hand", "road_to_guild")
	if g.Campaign.Labels["guild_answer"] != "went_willingly" {
		t.Fatalf("expected guild_answer=went_willingly, got %#v", g.Campaign.Labels)
	}

	statuses := objectiveStatuses(g)
	for _, id := range []string{"hear_father", "earn_coins", "buy_gift", "give_gift", "survive_raid"} {
		if statuses[id] != "completed" {
			t.Fatalf("expected %s completed, got %#v", id, statuses)
		}
	}
}

func TestBriarwell_KeepingTheGiftFailsObjectiveButRaidStillComes(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "Do I have to", "village_deeds")
	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{
		Labels: map[string]string{"deed_miller": "took_bribe", "deed_bully": "stopped"},
	})
	advanceTo(t, def, g, "gift_stall")
	chooseOption(t, def, g, "honey sweets", "birthday")
	chooseAndStay(t, def, g, "Keep the gift")
	chooseOption(t, def, g, "Ask Father", "bandit_raid")

	if got := objectiveStatuses(g)["give_gift"]; got != "failed" {
		t.Fatalf("expected give_gift failed, got %q", got)
	}
	if !g.Campaign.BoolFlags["gift_kept"] {
		t.Fatalf("expected gift_kept flag, got %#v", g.Campaign.BoolFlags)
	}
}

func TestBriarwell_ResolverCannotSkipAhead(t *testing.T) {
	def, _, _ := bootstrapBriarwell(t)
	// During the village deeds, the resolver must not be able to set raid or
	// gift state that belongs to later nodes.
	node := def.StoryNodes["village_deeds"]
	filtered := campaigns.FilterConditionResolution(campaigns.AllowedConditionSpecsForNode(node), campaigns.ConditionResolution{
		Labels: map[string]string{
			"deed_doll":     "returned",
			"deed_miller":   "murdered",
			"raid_response": "hid",
			"gift":          "sweets",
		},
	})
	if len(filtered.Labels) != 1 || filtered.Labels["deed_doll"] != "returned" {
		t.Fatalf("expected only deed_doll=returned to survive filtering, got %#v", filtered.Labels)
	}
}
