// world-gen is an async Lambda invoked by http-games after a new game is created.
// It uses the Architect agent to produce a world blueprint in a single call,
// then builds the world deterministically from that blueprint.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

type worldGenEvent struct {
	SessionID  string `json:"session_id"`
	UserID     string `json:"user_id"`
	PlayerName string `json:"player_name"`
}

func handler(ctx context.Context, evt worldGenEvent) error {
	log.Printf("world-gen: starting for session %s player %q", evt.SessionID, evt.PlayerName)

	dbClient, err := db.New(ctx)
	if err != nil {
		return err
	}

	// Load the stub game record created by http-games
	saveState, err := dbClient.GetGame(ctx, evt.SessionID)
	if err != nil {
		return err
	}
	g, err := game.FromSaveState(saveState)
	if err != nil {
		return err
	}

	aiClient, err := ai.New(ctx)
	if err != nil {
		return err
	}

	// Step 1: Generate the world blueprint
	log.Printf("world-gen: generating blueprint...")
	blueprint, rawJSON, err := aiClient.GenerateBlueprint(ctx, evt.PlayerName, "")
	if err != nil {
		log.Printf("world-gen: blueprint error: %v\nraw: %s", err, rawJSON)
		return err
	}
	log.Printf("world-gen: blueprint has %d rooms, %d items, %d characters",
		len(blueprint.Rooms), len(blueprint.Items), len(blueprint.Characters))

	// Step 2: Execute blueprint deterministically — no AI loop, no surprises
	if err := ai.BuildWorldFromBlueprint(g, blueprint); err != nil {
		log.Printf("world-gen: build error: %v", err)
		return err
	}

	// Step 3: Store opening narrative in chat history
	openingHistory := []game.ChatMessage{
		{Type: "narrative", Content: blueprint.OpeningScene},
	}

	// Step 4: Build a compact narrative context for the narrator
	// (just the opening — narrator will grow this from here)
	openingNarrative := []game.NarrativeMessage{
		{
			Role: "assistant",
			Content: []game.NarrativeBlock{
				{Type: "text", Text: blueprint.OpeningScene},
			},
		},
	}

	// Step 5: Mark ready and persist
	g.Ready = true
	g.Version++
	saved := g.ToSaveState(openingNarrative, openingHistory)

	// Retry loop for optimistic locking (unlikely to conflict here but safe)
	for attempt := 0; attempt < 3; attempt++ {
		if err := dbClient.PutGame(ctx, saved); err != nil {
			log.Printf("world-gen: put game attempt %d: %v", attempt+1, err)
			if attempt == 2 {
				return err
			}
			// Reload version and retry
			if fresh, loadErr := dbClient.GetGame(ctx, evt.SessionID); loadErr == nil {
				saved.Version = fresh.Version + 1
			}
			continue
		}
		break
	}

	log.Printf("world-gen: complete for session %s — %d rooms created", evt.SessionID, len(g.Rooms))

	// Debug: log the blueprint for visibility
	if b, err := json.MarshalIndent(blueprint, "", "  "); err == nil {
		log.Printf("world-gen: blueprint:\n%s", string(b))
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
