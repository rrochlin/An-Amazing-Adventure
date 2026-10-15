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

func TestBriarwell_DeedObjectivesTickOffAndUntouchedOneIsMissed(t *testing.T) {
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
	for id, want := range map[string]string{"miller_secret": "completed", "return_doll": "failed", "earn_coins": "completed"} {
		if got := objectiveStatus(g, id); got != want {
			t.Errorf("objective %s = %q, want %q", id, got, want)
		}
	}
	var messages []string
	for _, e := range second.Events {
		messages = append(messages, e.Message)
	}
	joined := strings.Join(messages, " | ")
	for _, want := range []string{"Objective complete: The Miller's Secret", "Objective missed: Bess's Lost Doll", "Objective updated: Choose a gift"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected event %q, got %s", want, joined)
		}
	}
}
