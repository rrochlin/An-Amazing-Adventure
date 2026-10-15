package campaigns_test

import (
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
)

func TestNarratorBrief_GroundsTheNarratorInTheAuthoredScene(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "I'll do it", "village_deeds")

	brief := campaigns.NarratorBrief(def, g)
	for _, want := range []string{
		"The Burning of Briarwell",
		"You are a child of about ten", // the locked background
		"Current scene: Good Turns and Bad",
		"Earn two coins",                           // the current objective
		"Little Bess", "Hob the Miller", "Dunstan", // the cast
		"Village Square", "Old Mill", "Meadow", // the places
		"Never reveal or hint at: the bandits; the coming raid",
		"Do NOT invent new quests",
		"End every reply with a clear, concrete hook",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("narrator brief is missing %q", want)
		}
	}
	if strings.Contains(brief, "Listen to Father") {
		t.Error("completed objectives should not be listed as active")
	}
}

func TestNarratorBrief_EmptyWithoutCampaign(t *testing.T) {
	_, g, _ := bootstrapBriarwell(t)
	if got := campaigns.NarratorBrief(nil, g); got != "" {
		t.Fatalf("expected no brief without a campaign definition, got %q", got)
	}
}

func TestBriarwell_ChoiceIntoAISceneOpensWithIntroText(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	if _, err := campaigns.ResumeDialogueWithChoice(def, g, g.Campaign.ActiveDialogue.PendingChoices[0].ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if result.ToNodeID != "village_deeds" || !strings.Contains(result.DialogueText, "Little Bess is crying by the well") {
		t.Fatalf("expected the village scene intro after Father's errand, got %#v", result)
	}
}

func TestNarratorBrief_HidesActorsOutsideTheirScenes(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	chooseOption(t, def, g, "I'll do it", "village_deeds")
	brief := campaigns.NarratorBrief(def, g)
	for _, hidden := range []string{"Red Corwin", "Master Holloway"} {
		if strings.Contains(brief, hidden) {
			t.Errorf("%s should not be briefed during the village scene", hidden)
		}
	}

	g.Campaign.ActiveNodeID = "bandit_raid"
	if !strings.Contains(campaigns.NarratorBrief(def, g), "Red Corwin") {
		t.Error("Red Corwin should be briefed during the raid")
	}
}
