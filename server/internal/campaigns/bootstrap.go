package campaigns

import (
	"fmt"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// BootstrapGame creates a minimal ready game from authored campaign data.
// This is intentionally small-scope for the current POC: it materializes rooms,
// places the player + companion NPCs, and seeds an opening narrative message.
func BootstrapGame(def *CampaignDefinition, sessionID, userID string, player game.Character, creation game.CharacterCreationData) (*game.Game, []game.ChatMessage, error) {
	if def == nil {
		return nil, nil, fmt.Errorf("campaign definition is required")
	}

	g := game.NewGame(sessionID, userID)
	g.Title = def.Title
	g.Theme = def.Tone
	g.QuestGoal = def.Premise
	g.CreationParams = creation
	g.Ready = true

	roomIDMap := make(map[string]string, len(def.Map.Rooms))
	for authoredID, room := range def.Map.Rooms {
		area := game.Area{
			ID:          authoredID,
			Name:        room.Name,
			Description: room.Description,
			Connections: make(map[string]string, len(room.Connections)),
			Coordinates: game.Coordinates{Z: float64(room.Floor)},
			Items:       []string{},
			Occupants:   []string{},
		}
		g.Rooms[area.ID] = area
		roomIDMap[authoredID] = area.ID
	}

	for authoredID, room := range def.Map.Rooms {
		area := g.Rooms[roomIDMap[authoredID]]
		for dir, target := range room.Connections {
			area.Connections[dir] = roomIDMap[target]
		}
		g.Rooms[area.ID] = area
	}

	startRoomID := def.Party.StartingRoomID
	if len(def.Map.Entrances) > 0 && def.Map.Entrances[0].StartRoomID != "" {
		// For the POC, start at the first entrance if present so the authored map
		// entry path is visible immediately.
		startRoomID = def.Map.Entrances[0].StartRoomID
	}
	startRoomID = roomIDMap[startRoomID]

	player.LocationID = startRoomID
	g.SetPlayerCharacter(userID, player)
	if room, ok := g.Rooms[startRoomID]; ok {
		_ = room.AddOccupant(player.ID)
		g.Rooms[startRoomID] = room
	}

	for _, actorID := range def.Party.CompanionIDs {
		a, ok := def.Actors[actorID]
		if !ok {
			continue
		}
		npc := game.NewCharacter(a.Name, a.Goal)
		npc.LocationID = startRoomID
		npc.Friendly = true
		g.NPCs[npc.ID] = npc
		if room, ok := g.Rooms[startRoomID]; ok {
			_ = room.AddOccupant(npc.ID)
			g.Rooms[startRoomID] = room
		}
	}

	history := []game.ChatMessage{}
	if def.IntroSequence.OpeningText != "" {
		history = append(history, game.ChatMessage{Type: "narrative", Content: def.IntroSequence.OpeningText})
	}

	return g, history, nil
}
