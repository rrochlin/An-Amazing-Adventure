// Package story runs one chat turn of the story: from the player's words to
// narration, world changes and campaign progress. A Pipeline is the unit the
// handler calls and the replay tool swaps, so a candidate pipeline (such as a
// game master) can be compared against the current one on the same turns.
package story

import (
	"context"
	"fmt"
	"log"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story/gm"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// Pipeline runs one chat turn. It mutates Game in place (world changes and
// campaign progress) and reports everything the caller needs to send frames,
// persist the turn and record its trace. It never writes to storage or
// sends frames itself.
type Pipeline interface {
	RunTurn(ctx context.Context, in TurnInput) (TurnOutput, error)
}

// TurnInput is everything a pipeline may read for one turn.
type TurnInput struct {
	Game        *game.Game
	Campaign    *campaigns.CampaignDefinition // nil for non-campaign games
	History     []game.NarrativeMessage       // narrator context so far
	PlayerInput string
	// OnChunk receives narration as it streams. Optional.
	OnChunk func(string)
	// OnNarrationDone is called once the narration is complete, before any
	// post-narration work, so the client isn't kept waiting. Optional.
	OnNarrationDone func()
}

// TurnOutput is what a turn produced. A non-nil error from RunTurn means the
// turn failed before any narration was delivered; partial failures after that
// are listed in Errors and the turn still completes.
type TurnOutput struct {
	Narrative string
	// NewMessages is the narrator context to persist after this turn,
	// including the authored dialogue when there is one.
	NewMessages []game.NarrativeMessage
	// AuthoredDialogue is scripted text entered by a transition this turn,
	// delivered after the narration. Empty when nothing was entered.
	AuthoredDialogue string
	// NarrativeEvents belong to the narration; SceneEvents belong to the
	// authored dialogue (a scene change shows before the new scene's text).
	NarrativeEvents []game.WorldEvent
	SceneEvents     []game.WorldEvent
	Mutations       []game.MutationEntry
	Tokens          ai.TokenUsage
	Calls           []trace.ModelCall
	// Resolution and Transition are set when the campaign was evaluated.
	Resolution *campaigns.ConditionResolution
	Transition *campaigns.TransitionResult
	// Decision is the game master's validated decision (nil for the
	// current pipeline).
	Decision *gm.Decision
	// Errors are non-fatal problems, in the order they happened.
	Errors []string
}

// Models is the model access the current pipeline needs. *ai.Client
// implements it.
type Models interface {
	NarrateStream(context.Context, *game.Game, []game.NarrativeMessage, string, *campaigns.CampaignDefinition, func(string)) (ai.NarratorResult, error)
	EngineerScan(context.Context, *game.Game, string) (ai.EngineerResult, error)
	ResolveCampaignConditions(context.Context, *game.Game, *campaigns.CampaignDefinition, string, string) (ai.CampaignConditionResolutionResult, error)
}

// Current is the pipeline the game runs today: the narrator streams prose,
// the engineer turns it into world changes, and for campaigns the resolver
// classifies the turn and the campaign orchestrator advances the story.
type Current struct {
	Models Models
}

// RunTurn implements Pipeline.
func (p Current) RunTurn(ctx context.Context, in TurnInput) (TurnOutput, error) {
	var out TurnOutput
	g := in.Game
	errorf := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		log.Printf("story: %s", msg)
		out.Errors = append(out.Errors, msg)
	}

	// 1. Narration, streamed to the caller.
	narrator, err := p.Models.NarrateStream(ctx, g, in.History, in.PlayerInput, in.Campaign, in.OnChunk)
	out.Calls = append(out.Calls, narrator.Calls...)
	if err != nil {
		errorf("narrator: %v", err)
		return out, fmt.Errorf("narrator: %w", err)
	}
	out.Narrative = narrator.Narrative
	out.NewMessages = narrator.NewMessages
	out.Tokens = narrator.Tokens
	if in.OnNarrationDone != nil {
		in.OnNarrationDone()
	}

	// 2. World changes implied by the narration. Non-fatal: the narration
	// has already been delivered.
	engineer, err := p.Models.EngineerScan(ctx, g, narrator.Narrative)
	out.Calls = append(out.Calls, engineer.Calls...)
	if err != nil {
		errorf("engineer: %v", err)
	}
	out.NarrativeEvents = engineer.Events
	out.Mutations = engineer.Mutations
	out.Tokens.InputTokens += engineer.Tokens.InputTokens
	out.Tokens.OutputTokens += engineer.Tokens.OutputTokens

	// 3. Campaign progress.
	if in.Campaign != nil {
		resolver, err := p.Models.ResolveCampaignConditions(ctx, g, in.Campaign, in.PlayerInput, narrator.Narrative)
		out.Calls = append(out.Calls, resolver.Calls...)
		if err != nil {
			errorf("resolver: %v", err)
		} else {
			resolution := resolver.Resolution
			out.Resolution = &resolution
			turn, err := campaigns.OrchestrateTurn(in.Campaign, g, resolution)
			if err != nil {
				errorf("transition: %v", err)
			} else {
				transition := turn.Transition
				out.Transition = &transition
				out.AuthoredDialogue = transition.DialogueText
			// Staging the new scene's dialogue ran into plays on its text.
			out.SceneEvents = append(out.SceneEvents, campaigns.DrainStaging(g, out.AuthoredDialogue)...)
				// Scene changes belong to the new scene's authored text (so
				// the client shows the title card before it); everything
				// else stays with the narration that caused it.
				for _, e := range turn.Events {
					if out.AuthoredDialogue != "" && IsSceneEvent(e) {
						out.SceneEvents = append(out.SceneEvents, e)
					} else {
						out.NarrativeEvents = append(out.NarrativeEvents, e)
					}
				}
			}
			out.Tokens.InputTokens += resolver.Tokens.InputTokens
			out.Tokens.OutputTokens += resolver.Tokens.OutputTokens
		}
	}

	if out.AuthoredDialogue != "" {
		out.NewMessages = append(out.NewMessages, game.NarrativeMessage{
			Role:    "assistant",
			Content: []game.NarrativeBlock{{Type: "text", Text: out.AuthoredDialogue}},
		})
	}
	return out, nil
}

// IsSceneEvent reports whether an event announces a change of scene.
func IsSceneEvent(e game.WorldEvent) bool {
	switch e.Type {
	case "scene_started", "campaign_progress", "campaign_ended":
		return true
	}
	return false
}
