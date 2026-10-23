package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// Budget is the target time for one turn (docs/decisions.md, 2026-10-22).
const Budget = 10 * time.Second

// rebuilt is a traced chat turn's starting point, ready to run again.
type rebuilt struct {
	Game     *game.Game
	Campaign *campaigns.CampaignDefinition
	History  []game.NarrativeMessage
	// Missing lists what the trace didn't hold, and how it was approximated.
	Missing []string
}

// rebuild reconstructs the state a traced chat turn started from. Traces
// with a starting world (Before.World) rebuild exactly; older ones fall back
// to the session's final world with the traced campaign state and player
// room laid over it, and say so in Missing.
func rebuild(ctx context.Context, save game.SaveState, t *trace.Turn, reg *campaigns.Registry) (*rebuilt, error) {
	r := &rebuilt{}
	if t.Before == nil {
		return nil, fmt.Errorf("trace has no starting snapshot")
	}

	world := save
	if len(t.Before.World) > 0 {
		world = game.SaveState{}
		if err := json.Unmarshal(t.Before.World, &world); err != nil {
			return nil, fmt.Errorf("decode starting world: %w", err)
		}
	} else {
		r.Missing = append(r.Missing, "starting world (rooms, items, characters): used the session's final world with the traced campaign state and player room")
		if len(t.Before.Campaign) > 0 {
			world.Campaign = &game.CampaignRuntimeState{}
			if err := json.Unmarshal(t.Before.Campaign, world.Campaign); err != nil {
				return nil, fmt.Errorf("decode starting campaign state: %w", err)
			}
		}
		world.Narrative = nil
	}

	g, err := game.FromSaveState(world)
	if err != nil {
		return nil, fmt.Errorf("load game: %w", err)
	}
	if world.PlayersData != nil {
		if _, err := g.LoadDnDCharacters(ctx, world.PlayersData); err != nil {
			r.Missing = append(r.Missing, fmt.Sprintf("D&D characters failed to load: %v", err))
		}
	}
	if len(t.Before.World) == 0 && t.Before.PlayerRoomID != "" {
		if owner, ok := g.OwnerCharacter(); ok {
			owner.LocationID = t.Before.PlayerRoomID
			g.SetPlayerCharacter(g.OwnerID, owner)
		}
	}
	r.Game = g

	// Narrator context: the save's, or what the traced narrator call was sent.
	r.History = world.Narrative
	if len(t.Before.World) == 0 {
		if msgs := narratorMessages(t); len(msgs) > 0 {
			r.History = msgs[:len(msgs)-1] // the last message is this turn's input
			r.Missing = append(r.Missing, "narrator context: used the traced narrator request, already trimmed")
		} else {
			r.Missing = append(r.Missing, "narrator context: none recorded; replayed with an empty history")
		}
	}

	if g.Campaign != nil && g.Campaign.CampaignID != "" {
		def, ok := reg.Get(g.Campaign.CampaignID)
		if !ok {
			return nil, fmt.Errorf("campaign %q is not in this build", g.Campaign.CampaignID)
		}
		r.Campaign = def
	}
	return r, nil
}

func narratorMessages(t *trace.Turn) []game.NarrativeMessage {
	for _, c := range t.Calls {
		if c.Role == "narrator" {
			return c.Messages
		}
	}
	return nil
}

// result is one replayed turn: the original trace next to the candidate run.
type result struct {
	Trace   *trace.Turn
	Missing []string
	// Skipped is set when the turn could not be replayed.
	Skipped string

	Output  story.TurnOutput
	Err     error
	Elapsed time.Duration
	After   *trace.Snapshot
	// Before is the rebuilt starting state, for the candidate's state diff.
	Before *trace.Snapshot
}

// runner replays traced chat turns through a candidate pipeline. It never
// writes anything: the pipeline works on rebuilt in-memory games.
type runner struct {
	Registry *campaigns.Registry
	// Pipeline returns the candidate pipeline for a turn.
	Pipeline func(*trace.Turn) story.Pipeline
	Now      func() time.Time
}

func (rn runner) replay(ctx context.Context, save game.SaveState, traces []*trace.Turn) []result {
	now := rn.Now
	if now == nil {
		now = time.Now
	}
	var out []result
	for _, t := range traces {
		res := result{Trace: t}
		if t.Kind != trace.KindChat {
			res.Skipped = fmt.Sprintf("%s turn: only chat turns run through the story pipeline", t.Kind)
			out = append(out, res)
			continue
		}
		r, err := rebuild(ctx, save, t, rn.Registry)
		if err != nil {
			res.Skipped = "could not rebuild: " + err.Error()
			out = append(out, res)
			continue
		}
		res.Missing = r.Missing
		res.Before = trace.Snap(r.Game)
		start := now()
		res.Output, res.Err = rn.Pipeline(t).RunTurn(ctx, story.TurnInput{
			Game:        r.Game,
			Campaign:    r.Campaign,
			History:     r.History,
			PlayerInput: t.Input,
		})
		res.Elapsed = now().Sub(start)
		res.After = trace.Snap(r.Game)
		out = append(out, res)
	}
	return out
}

// recorded is a pipeline that returns what the original turn said and
// changes nothing. It checks the rebuild and the report without calling a
// model.
type recorded struct{ t *trace.Turn }

func (p recorded) RunTurn(context.Context, story.TurnInput) (story.TurnOutput, error) {
	var out story.TurnOutput
	for _, c := range p.t.Calls {
		if c.Role == "narrator" {
			out.Narrative = c.Output
		}
	}
	return out, nil
}
