// world-gen is an async Lambda invoked by http-games after a new game is created.
// It uses the Architect agent to produce a world blueprint in a single call,
// then builds the world deterministically from that blueprint.
// While running it emits world_gen_log frames over WebSocket so the client can
// show a live terminal. A world_gen_ready frame is sent on completion.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

type worldGenEvent struct {
	SessionID         string   `json:"session_id"`
	UserID            string   `json:"user_id"`
	PlayerName        string   `json:"player_name"`
	PlayerDescription string   `json:"player_description,omitempty"`
	PlayerAge         string   `json:"player_age,omitempty"`
	PlayerBackstory   string   `json:"player_backstory,omitempty"`
	ThemeHint         string   `json:"theme_hint,omitempty"`
	Preferences       []string `json:"preferences,omitempty"`
}

func handler(ctx context.Context, evt worldGenEvent) error {
	log.Printf("world-gen: starting for session %s player %q", evt.SessionID, evt.PlayerName)

	dbClient, err := db.New(ctx)
	if err != nil {
		return err
	}

	// Best-effort WebSocket push — if the client isn't connected we just log
	// and skip the push rather than failing the whole job.
	var sender *wsutil.Sender
	var connID string
	if ws, wsErr := wsutil.New(ctx); wsErr == nil {
		conn, connErr := dbClient.GetConnectionByUserID(ctx, evt.UserID)
		if connErr == nil {
			sender = ws
			connID = conn.ConnectionID
			log.Printf("world-gen: will push progress to connection %s", connID)
		} else {
			log.Printf("world-gen: no active connection for user, skipping WS push: %v", connErr)
		}
	} else {
		log.Printf("world-gen: WS sender unavailable (WEBSOCKET_API_ENDPOINT not set?): %v", wsErr)
	}

	// emit pushes a log line to the terminal (non-fatal on WS error).
	emit := func(line string) {
		log.Printf("world-gen: %s", line)
		if sender != nil {
			if err := sender.SendWorldGenLog(ctx, connID, line); err != nil {
				log.Printf("world-gen: send log frame: %v", err)
				// Stale connection — stop trying to push
				sender = nil
			}
		}
	}

	// Load the stub game record created by http-games
	emit("Loading game record...")
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
	nameLabel := evt.PlayerName
	if nameLabel == "" {
		nameLabel = "an unnamed adventurer"
	}
	emit(fmt.Sprintf("Summoning the Architect for %q...", nameLabel))
	blueprint, rawJSON, blueprintTokens, err := aiClient.GenerateBlueprint(
		ctx,
		evt.PlayerName,
		evt.PlayerDescription,
		evt.PlayerBackstory,
		evt.ThemeHint,
		evt.Preferences,
	)
	if err != nil {
		emit(fmt.Sprintf("ERROR: blueprint generation failed: %v", err))
		log.Printf("world-gen: blueprint error: %v\nraw: %s", err, rawJSON)
		return err
	}
	emit(fmt.Sprintf("Blueprint ready: %q — %d rooms, %d items, %d characters",
		blueprint.Title, len(blueprint.Rooms), len(blueprint.Items), len(blueprint.Characters)))
	emit(fmt.Sprintf("Theme: %s", blueprint.Theme))
	emit(fmt.Sprintf("Quest: %s", blueprint.QuestGoal))

	// If the AI invented a player name (player left it blank), write it back.
	// Apply player-supplied character details (may override stub values).
	owner, hasOwner := g.GetPlayerCharacter(g.OwnerID)
	if !hasOwner {
		owner = game.NewCharacter("Adventurer", "")
	}
	if evt.PlayerName == "" && blueprint.PlayerName != "" {
		owner.Name = blueprint.PlayerName
		emit(fmt.Sprintf("Character named: %q", blueprint.PlayerName))
	}
	if evt.PlayerDescription != "" {
		owner.Description = evt.PlayerDescription
	}
	if evt.PlayerAge != "" {
		owner.Age = evt.PlayerAge
	}
	if evt.PlayerBackstory != "" {
		owner.Backstory = evt.PlayerBackstory
	}
	g.SetPlayerCharacter(g.OwnerID, owner)

	// Step 2: Build world deterministically
	emit("Constructing world...")
	if err := ai.BuildWorldFromBlueprint(g, blueprint); err != nil {
		emit(fmt.Sprintf("ERROR: world build failed: %v", err))
		log.Printf("world-gen: build error: %v", err)
		return err
	}
	emit(fmt.Sprintf("World built — %d rooms placed", len(g.Rooms)))

	// Step 3: Populate opening state
	emit("Writing opening scene...")
	openingHistory := []game.ChatMessage{
		{Type: "narrative", Content: blueprint.OpeningScene},
	}
	openingNarrative := []game.NarrativeMessage{
		{
			Role: "assistant",
			Content: []game.NarrativeBlock{
				{Type: "text", Text: blueprint.OpeningScene},
			},
		},
	}

	// Account for blueprint token usage — best-effort, non-fatal.
	if accountErr := dbClient.UpdateUserTokens(ctx, evt.UserID, blueprintTokens.Total()); accountErr != nil {
		log.Printf("world-gen: UpdateUserTokens (non-fatal): %v", accountErr)
	}

	// Step 4: Persist and mark ready
	emit("Sealing the world into the tome...")
	g.Ready = true
	g.Version++
	// Populate metadata fields from the blueprint
	g.Title = blueprint.Title
	g.Theme = blueprint.Theme
	g.QuestGoal = blueprint.QuestGoal
	g.TotalTokens = blueprintTokens.Total()
	g.CreationParams = game.AdventureCreationParams{
		PlayerDescription: evt.PlayerDescription,
		PlayerAge:         evt.PlayerAge,
		PlayerBackstory:   evt.PlayerBackstory,
		ThemeHint:         evt.ThemeHint,
		Preferences:       evt.Preferences,
	}
	saved := g.ToSaveState(openingNarrative, openingHistory)

	for attempt := 0; attempt < 3; attempt++ {
		if err := dbClient.PutGame(ctx, saved); err != nil {
			log.Printf("world-gen: put game attempt %d: %v", attempt+1, err)
			if attempt == 2 {
				emit("ERROR: failed to save world")
				return err
			}
			if fresh, loadErr := dbClient.GetGame(ctx, evt.SessionID); loadErr == nil {
				saved.Version = fresh.Version + 1
			}
			continue
		}
		break
	}

	emit("Your adventure awaits.")
	log.Printf("world-gen: complete for session %s — %d rooms", evt.SessionID, len(g.Rooms))

	// Signal the client to transition to the game
	if sender != nil {
		if err := sender.SendWorldGenReady(ctx, connID); err != nil {
			log.Printf("world-gen: send world_ready frame: %v", err)
		}
	}

	// Debug: log the blueprint
	if b, err := json.MarshalIndent(blueprint, "", "  "); err == nil {
		log.Printf("world-gen: blueprint:\n%s", string(b))
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
