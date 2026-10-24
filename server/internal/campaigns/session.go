package campaigns

import (
	"context"
	"errors"
	"fmt"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// ErrCharacterCreation wraps a character the rules couldn't build.
var ErrCharacterCreation = errors.New("character creation failed")

// StartSession creates a campaign session as a player starts one, from
// creation data the campaign's rules have already been applied to (so a
// quick start has the preset's class, race and scores): the run is
// bootstrapped and the player's D&D character is built under the hood, so
// every session has real stats even when the player only chose a name.
func StartSession(ctx context.Context, def *CampaignDefinition, sessionID, userID string, creation game.CharacterCreationData) (*game.Game, []game.ChatMessage, error) {
	name := creation.Name
	if name == "" {
		name = "Adventurer"
	}
	g, history, err := BootstrapGame(def, sessionID, userID, game.NewCharacter(name, creation.Backstory), creation)
	if err != nil {
		return nil, nil, err
	}
	if creation.ClassID != "" && creation.RaceID != "" && len(creation.AbilityScores) == 6 {
		dnd, err := game.BuildDnDCharacter(ctx, creation)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrCharacterCreation, err)
		}
		g.SetDnDCharacter(userID, dnd)
	}
	return g, history, nil
}
