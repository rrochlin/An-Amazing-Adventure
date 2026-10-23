package story

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story/gm"
)

// GMModels is the model access the game master pipeline needs. *ai.Client
// implements it.
type GMModels interface {
	GameMasterTurn(context.Context, ai.GameMasterRequest) (ai.GameMasterResult, error)
}

// GameMaster is the pipeline where one agent runs the turn: it narrates and
// hands in a Decision in a single model call (docs/story-engine/game-master.md).
// Code validates the decision against the scene, applies the world events
// through the engineer's tools, and advances the campaign from the beats.
type GameMaster struct {
	Models GMModels
}

// RunTurn implements Pipeline.
func (p GameMaster) RunTurn(ctx context.Context, in TurnInput) (TurnOutput, error) {
	var out TurnOutput
	g := in.Game
	errorf := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		log.Printf("story/gm: %s", msg)
		out.Errors = append(out.Errors, msg)
	}

	node := activeScene(in.Campaign, g)
	result, err := p.Models.GameMasterTurn(ctx, ai.GameMasterRequest{
		System:      gm.Brief(in.Campaign, g),
		History:     in.History,
		PlayerInput: in.PlayerInput,
		ToolName:    gm.ToolName,
		ToolSchema:  gm.Schema(node),
		OnChunk:     in.OnChunk,
	})
	out.Calls = append(out.Calls, result.Calls...)
	if err != nil {
		errorf("game master: %v", err)
		return out, fmt.Errorf("game master: %w", err)
	}
	out.Narrative = result.Narrative
	out.NewMessages = result.NewMessages
	out.Tokens = result.Tokens
	if in.OnNarrationDone != nil {
		in.OnNarrationDone()
	}

	// The decision: what the scene permits of what the model claimed.
	var decision gm.Decision
	if result.Decision == "" {
		errorf("game master handed in no decision")
	} else if err := json.Unmarshal([]byte(result.Decision), &decision); err != nil {
		errorf("game master decision is not valid JSON: %v", err)
	}
	kept, dropped := gm.Validate(node, decision)
	for _, d := range dropped {
		errorf("dropped: %s", d)
	}
	out.Decision = &kept

	// World changes, through the same tools the engineer used.
	for _, ev := range kept.WorldEvents {
		resultText, event, err := ai.DispatchTool(ctx, g, ev.Type, ev.Input)
		if err != nil {
			errorf("%s: %v", ev.Type, err)
			resultText = "error: " + err.Error()
		} else if event != nil {
			out.NarrativeEvents = append(out.NarrativeEvents, *event)
		}
		out.Mutations = append(out.Mutations, game.MutationEntry{
			SessionID: g.ID, Ts: time.Now().UnixMilli(), Turn: g.ConversationCount,
			Tool: ev.Type, Input: ev.Input, Result: resultText,
		})
	}

	// Campaign progress from the settled beats.
	if in.Campaign != nil {
		resolution := gm.Resolution(kept)
		out.Resolution = &resolution
		turn, err := campaigns.OrchestrateTurn(in.Campaign, g, resolution)
		if err != nil {
			errorf("transition: %v", err)
		} else {
			transition := turn.Transition
			out.Transition = &transition
			out.AuthoredDialogue = transition.DialogueText
			for _, e := range turn.Events {
				if out.AuthoredDialogue != "" && IsSceneEvent(e) {
					out.SceneEvents = append(out.SceneEvents, e)
				} else {
					out.NarrativeEvents = append(out.NarrativeEvents, e)
				}
			}
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

func activeScene(def *campaigns.CampaignDefinition, g *game.Game) *campaigns.StoryNode {
	if def == nil || g.Campaign == nil {
		return nil
	}
	if node, ok := def.StoryNodes[g.Campaign.ActiveNodeID]; ok {
		return &node
	}
	return nil
}

// PipelineFor picks the pipeline a campaign asks for (story_pipeline), the
// current one by default.
func PipelineFor(def *campaigns.CampaignDefinition, models interface {
	Models
	GMModels
}) Pipeline {
	if def != nil && def.StoryPipeline == campaigns.StoryPipelineGameMaster {
		return GameMaster{Models: models}
	}
	return Current{Models: models}
}
