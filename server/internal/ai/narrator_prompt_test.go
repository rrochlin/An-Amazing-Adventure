package ai

import (
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func TestNarratorSystemPrompt_CampaignAddsBriefAndDirection(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := reg.Get("briarwell")
	g, _, err := campaigns.BootstrapGame(def, "s", "u", game.NewCharacter("Wren", ""),
		game.CharacterCreationData{Name: "Wren", RaceID: "human", ClassID: "rogue"})
	if err != nil {
		t.Fatal(err)
	}

	prompt := narratorSystemPrompt(g, def)
	if !strings.Contains(prompt, "Authored campaign: The Burning of Briarwell") {
		t.Error("expected the campaign brief in the narrator prompt")
	}
	if strings.Contains(prompt, "Never list options") {
		t.Error("campaign prompts should ask for a concrete hook, not forbid direction")
	}

	plain := narratorSystemPrompt(g, nil)
	if strings.Contains(plain, "Authored campaign") || !strings.Contains(plain, "Never list options") {
		t.Error("non-campaign prompts should be unchanged")
	}
}
