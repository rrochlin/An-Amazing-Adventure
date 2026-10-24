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

	activeNodeID := def.StartNodeID
	if def.IntroSequence.OpeningNodeID != "" {
		activeNodeID = def.IntroSequence.OpeningNodeID
	}
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      def.ID,
		CampaignVersion: def.Version,
		ActiveNodeID:    activeNodeID,
		BoolFlags:       make(map[string]bool),
		Labels:          make(map[string]string),
		Counters:        make(map[string]int),
		ActorStates:     make(map[string]game.ActorRuntimeState),
		CompanionStates: make(map[string]game.CompanionState),
	}

	roomIDMap := make(map[string]string, len(def.Map.Rooms))
	for authoredID, room := range def.Map.Rooms {
		area := game.Area{
			ID:               authoredID,
			Name:             room.Name,
			Description:      room.Description,
			Connections:      make(map[string]string, len(room.Connections)),
			Coordinates:      game.Coordinates{Z: float64(room.Floor)},
			Items:            []string{},
			Occupants:        []string{},
			PointsOfInterest: append([]game.PointOfInterest(nil), room.PointsOfInterest...),
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
	if def.PlayerRole.Scale != 0 {
		player.Scale = def.PlayerRole.Scale
	}
	g.SetPlayerCharacter(userID, player)
	if room, ok := g.Rooms[startRoomID]; ok {
		_ = room.AddOccupant(player.ID)
		g.Rooms[startRoomID] = room
	}
	g.Discover(startRoomID)

	// Lay the map out from the exits (east is one step right, north one step
	// up), as procedural worlds do. Without this every room sat at (0, 0) and
	// the map drew a single node. The authored floor stays the map layer.
	g.CalculateRoomCoordinates()
	for authoredID, room := range def.Map.Rooms {
		area := g.Rooms[roomIDMap[authoredID]]
		area.Coordinates.Z = float64(room.Floor)
		g.Rooms[area.ID] = area
	}

	// Authored items, where the campaign says they lie.
	for authoredID, room := range def.Map.Rooms {
		for _, it := range room.Items {
			item := game.NewItem(it.Name, it.Description)
			if err := g.AddItem(item); err != nil {
				return nil, nil, fmt.Errorf("room %q item %q: %w", authoredID, it.Name, err)
			}
			if err := g.PlaceItemInRoom(item.ID, roomIDMap[authoredID]); err != nil {
				return nil, nil, fmt.Errorf("room %q item %q: %w", authoredID, it.Name, err)
			}
		}
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
		g.Campaign.CompanionStates[actorID] = game.CompanionState{
			CompanionID:       actorID,
			Status:            "active",
			CurrentLocationID: startRoomID,
		}
	}

	// The cast with a starting place exists in the world as real characters,
	// keyed by actor ID, so the map, the engineer and later agents see them.
	for actorID, actor := range def.Actors {
		if actor.Kind == "companion" || actor.StartLocationID == "" {
			continue
		}
		roomID, ok := roomIDMap[actor.StartLocationID]
		if !ok {
			return nil, nil, fmt.Errorf("actor %q starts in unknown room %q", actorID, actor.StartLocationID)
		}
		npc := newActorCharacter(actorID, actor)
		npc.LocationID = roomID
		g.NPCs[npc.ID] = npc
		room := g.Rooms[roomID]
		_ = room.AddOccupant(npc.ID)
		g.Rooms[roomID] = room
	}

	for actorID, actor := range def.Actors {
		if actor.Kind == "companion" {
			continue
		}
		state := game.ActorRuntimeState{ActorID: actorID, Status: "idle"}
		if len(actor.Rail.Steps) > 0 {
			state.CurrentRailStepID = actor.Rail.Steps[0].ID
			state.CurrentLocationID = actor.Rail.Steps[0].LocationID
		}
		g.Campaign.ActorStates[actorID] = state
	}

	history := []game.ChatMessage{}
	// The opening scene gets a title card like every later scene.
	if node, ok := def.StoryNodes[g.Campaign.ActiveNodeID]; ok {
		history = append(history, game.ChatMessage{
			Type:   "narrative",
			Events: []game.WorldEvent{SceneStartedEvent(node)},
		})
	}
	if def.IntroSequence.OpeningText != "" {
		history = append(history, game.ChatMessage{Type: "narrative", Content: def.IntroSequence.OpeningText})
	}
	if dialogueText, err := EnterActiveNode(def, g); err != nil {
		return nil, nil, err
	} else if dialogueText != "" {
		history = append(history, game.ChatMessage{Type: "narrative", Content: dialogueText, Events: DrainStaging(g, dialogueText)})
	}

	return g, history, nil
}

// newActorCharacter is an actor as a character in the world, keyed by the
// actor's ID; it isn't placed anywhere yet.
func newActorCharacter(actorID string, actor ActorTemplate) game.Character {
	description := actor.Description
	if description == "" {
		description = actor.Goal
	}
	npc := game.NewCharacter(actor.Name, description)
	npc.ID = actorID
	npc.Friendly = actor.Kind != "antagonist"
	npc.Scale = actor.Scale
	return npc
}
