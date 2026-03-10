// world-gen is an async Lambda invoked by http-games after a new game is created.
// It generates a dungeon layout procedurally using rpg-toolkit/tools/environments,
// seeds encounters, then calls Claude Sonnet once for narrative framing.
// While running it emits world_gen_log frames over WebSocket so the client can
// show a live terminal. A world_gen_ready frame is sent on completion.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	rpgevents "github.com/KirkDiggler/rpg-toolkit/events"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster"
	"github.com/KirkDiggler/rpg-toolkit/tools/environments"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/combat"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

type worldGenEvent struct {
	SessionID      string                     `json:"session_id"`
	UserID         string                     `json:"user_id"`
	CreationParams game.CharacterCreationData `json:"creation_params"`
	// Legacy fields — preserved for backward-compat
	PlayerName        string   `json:"player_name,omitempty"`
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

	// Best-effort WebSocket push — if no clients are connected we just log
	// and skip the push rather than failing the whole job.
	var sender *wsutil.Sender
	var gameConns []db.Connection
	if ws, wsErr := wsutil.New(ctx); wsErr == nil {
		sender = ws
		conns, connErr := dbClient.GetConnectionsByGameID(ctx, evt.SessionID)
		if connErr == nil && len(conns) > 0 {
			gameConns = conns
			log.Printf("world-gen: will push progress to %d connection(s)", len(gameConns))
		} else {
			log.Printf("world-gen: no active connections for session, skipping WS push: %v", connErr)
		}
	} else {
		log.Printf("world-gen: WS sender unavailable (WEBSOCKET_API_ENDPOINT not set?): %v", wsErr)
	}

	// emit pushes a log line to the terminal for all connected party members.
	emit := func(line string) {
		log.Printf("world-gen: %s", line)
		if sender != nil {
			for _, gc := range gameConns {
				if err := sender.SendWorldGenLog(ctx, gc.ConnectionID, line); err != nil {
					log.Printf("world-gen: send log frame to %s: %v", gc.ConnectionID, err)
				}
			}
		}
	}

	// Load the stub game record created by http-games.
	emit("Loading game record...")
	saveState, err := dbClient.GetGame(ctx, evt.SessionID)
	if err != nil {
		return err
	}
	g, err := game.FromSaveState(saveState)
	if err != nil {
		return err
	}

	// Load existing DnD characters (if any) so they survive the save at the end.
	if saveState.PlayersData != nil {
		bus, loadErr := g.LoadDnDCharacters(ctx, saveState.PlayersData)
		if loadErr != nil {
			log.Printf("world-gen: LoadDnDCharacters (non-fatal): %v", loadErr)
		} else {
			_ = bus
		}
	}

	// Resolve creation params — prefer the v3 struct, fall back to legacy fields.
	creationParams := evt.CreationParams
	if creationParams.ThemeHint == "" {
		creationParams.ThemeHint = evt.ThemeHint
	}
	if len(creationParams.Preferences) == 0 {
		creationParams.Preferences = evt.Preferences
	}
	if creationParams.Name == "" {
		creationParams.Name = evt.PlayerName
	}

	// Apply player character details to the owner stub.
	owner, hasOwner := g.GetPlayerCharacter(g.OwnerID)
	if !hasOwner {
		owner = game.NewCharacter("Adventurer", "")
	}
	if creationParams.Name != "" {
		owner.Name = creationParams.Name
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

	aiClient, err := ai.New(ctx)
	if err != nil {
		return err
	}

	// Use a time-based seed for procedural generation.
	seed := time.Now().UnixNano()

	// ── Step 1: Generate dungeon layout ──────────────────────────────────────
	emit("Generating dungeon layout...")
	envData, err := generateDungeonLayout(ctx, seed, creationParams.ThemeHint)
	if err != nil {
		emit(fmt.Sprintf("ERROR: dungeon layout failed: %v", err))
		return err
	}
	emit(fmt.Sprintf("Layout ready: %d rooms, %d passages", len(envData.Zones), len(envData.Passages)))

	// ── Step 2: Populate encounters ───────────────────────────────────────────
	emit("Placing encounters...")
	roomMonsters := populateEncounters(envData, seed)
	totalMonsters := 0
	for _, ms := range roomMonsters {
		totalMonsters += len(ms)
	}
	emit(fmt.Sprintf("Placed %d monsters across %d rooms", totalMonsters, len(roomMonsters)))
	emit("Rolling initiative for room bosses...")

	// ── Step 3: Generate narrative framing ───────────────────────────────────
	emit("Generating narrative...")
	dungeonSummary := buildDungeonSummary(envData, roomMonsters, creationParams)
	framing, framingTokens, err := aiClient.GenerateNarrativeFraming(ctx, dungeonSummary, creationParams)
	if err != nil {
		emit(fmt.Sprintf("ERROR: narrative framing failed: %v", err))
		log.Printf("world-gen: framing error: %v\ndungeon summary: %s", err, dungeonSummary)
		return err
	}
	emit(fmt.Sprintf("Narrative ready: %q", framing.Title))
	emit(fmt.Sprintf("Theme: %s", framing.Theme))
	emit(fmt.Sprintf("Quest: %s", framing.QuestGoal))

	// ── Step 4: Build DungeonData ─────────────────────────────────────────────
	emit("Building world...")
	dungeonData := buildDungeonData(envData, framing, seed)

	// Persist monsters into g.RoomMonsters so combat resolution still works.
	for roomID, ms := range roomMonsters {
		monsterData := make([]*monster.Data, 0, len(ms))
		for _, m := range ms {
			monsterData = append(monsterData, m.ToData())
		}
		g.SetRoomMonsters(roomID, monsterData)
	}

	// Place the owner in the dungeon's starting room.
	if dungeonData.StartRoomID != "" {
		owner.LocationID = dungeonData.StartRoomID
		g.SetPlayerCharacter(g.OwnerID, owner)
	}

	// Build the legacy Rooms map from DungeonData so existing navigation code works.
	buildLegacyRooms(g, dungeonData)

	// Account for narrative framing token usage.
	// Non-fatal: world is already built; don't abort on accounting failure.
	// ErrUserNotFound here means the user was deleted mid-flight — log loudly.
	if accountErr := dbClient.UpdateUserTokens(ctx, evt.UserID, framingTokens.Total()); accountErr != nil {
		log.Printf("world-gen: UpdateUserTokens FAILED (non-fatal) user=%s: %v", evt.UserID, accountErr)
	}

	// ── Step 5: Persist and mark ready ───────────────────────────────────────
	emit("Sealing the world into the tome...")
	openingHistory := []game.ChatMessage{
		{Type: "narrative", Content: framing.OpeningScene},
	}
	openingNarrative := []game.NarrativeMessage{
		{
			Role: "assistant",
			Content: []game.NarrativeBlock{
				{Type: "text", Text: framing.OpeningScene},
			},
		},
	}

	g.Ready = true
	g.Version++
	g.Title = framing.Title
	g.Theme = framing.Theme
	g.QuestGoal = framing.QuestGoal
	g.TotalTokens = framingTokens.Total()
	g.DungeonData = dungeonData

	// Preserve creation params.
	if creationParams.ClassID != "" || creationParams.RaceID != "" {
		g.CreationParams = creationParams
	} else {
		g.CreationParams = game.CharacterCreationData{
			Name:        evt.PlayerName,
			ThemeHint:   evt.ThemeHint,
			Preferences: evt.Preferences,
		}
		g.LegacyCreationParams = game.AdventureCreationParams{
			PlayerDescription: evt.PlayerDescription,
			PlayerAge:         evt.PlayerAge,
			PlayerBackstory:   evt.PlayerBackstory,
			ThemeHint:         evt.ThemeHint,
			Preferences:       evt.Preferences,
		}
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
	log.Printf("world-gen: complete for session %s — %d rooms", evt.SessionID, len(dungeonData.Rooms))

	// Signal all connected party members to transition to the game.
	if sender != nil {
		for _, gc := range gameConns {
			if err := sender.SendWorldGenReady(ctx, gc.ConnectionID); err != nil {
				log.Printf("world-gen: send world_ready to %s: %v", gc.ConnectionID, err)
			}
		}
	}

	return nil
}

// ── Dungeon layout generation ─────────────────────────────────────────────────

// generateDungeonLayout uses rpg-toolkit/tools/environments to create a room
// graph and returns the serializable EnvironmentData.
func generateDungeonLayout(ctx context.Context, seed int64, themeHint string) (*environments.EnvironmentData, error) {
	gen := environments.NewGraphBasedGenerator(environments.GraphBasedGeneratorConfig{
		ID:   uuid.New().String(),
		Type: "dungeon",
		Seed: seed,
	})
	// The generator requires an event bus to be wired before calling Generate.
	gen.ConnectToEventBus(rpgevents.NewEventBus())

	theme := "dungeon"
	if themeHint != "" {
		theme = themeHint
	}

	cfg := environments.GenerationConfig{
		ID:        uuid.New().String(),
		Type:      environments.GenerationTypeGraph,
		Seed:      seed,
		Theme:     theme,
		Size:      environments.EnvironmentSizeCustom,
		RoomCount: 8,
		Layout:    environments.LayoutTypeBranching,
		RoomTypes: []string{
			environments.RoomTypeEntrance,
			environments.RoomTypeChamber,
			environments.RoomTypeChamber,
			environments.RoomTypeChamber,
			environments.RoomTypeChamber,
			environments.RoomTypeTreasure,
			environments.RoomTypeBoss,
		},
		Density:      0.6,
		Connectivity: 0.5,
		Metadata: environments.EnvironmentMetadata{
			Name:        "dungeon",
			GeneratedBy: "world-gen-v4",
		},
	}

	env, err := gen.Generate(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("environments.Generate: %w", err)
	}

	// Export to EnvironmentData via ToData() if available, else via JSON Export().
	if be, ok := env.(*environments.BasicEnvironment); ok {
		data := be.ToData()
		return &data, nil
	}

	raw, err := env.Export()
	if err != nil {
		return nil, fmt.Errorf("environments.Export: %w", err)
	}
	var data environments.EnvironmentData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parse environment data: %w", err)
	}
	return &data, nil
}

// ── Encounter population ──────────────────────────────────────────────────────

// populateEncounters seeds rooms with monsters based on room type.
// Returns a map of zoneID → live *monster.Monster slice.
func populateEncounters(env *environments.EnvironmentData, seed int64) map[string][]*monster.Monster {
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec
	roomMonsters := make(map[string][]*monster.Monster)

	// Weighted low-CR monster table.
	type entry struct {
		monsterType string
		weight      int
	}
	lowCRTable := []entry{
		{"skeleton", 3},
		{"zombie", 3},
		{"goblin", 2},
		{"wolf", 2},
		{"giant_rat", 2},
		{"bandit", 2},
		{"ghoul", 1},
	}
	totalWeight := 0
	for _, e := range lowCRTable {
		totalWeight += e.weight
	}
	pickLowCR := func() string {
		r := rng.Intn(totalWeight)
		cum := 0
		for _, e := range lowCRTable {
			cum += e.weight
			if r < cum {
				return e.monsterType
			}
		}
		return "skeleton"
	}

	for _, zone := range env.Zones {
		switch zone.Type {
		case environments.RoomTypeEntrance:
			// Safe starting area — no monsters.

		case environments.RoomTypeBoss:
			// Boss room: Brown Bear + 2 Ghouls.
			var ms []*monster.Monster
			if m := combat.NewMonsterByType("brown_bear"); m != nil {
				ms = append(ms, m)
			}
			for i := 0; i < 2; i++ {
				if m := combat.NewMonsterByType("ghoul"); m != nil {
					ms = append(ms, m)
				}
			}
			if len(ms) > 0 {
				roomMonsters[zone.ID] = ms
			}

		case environments.RoomTypeTreasure:
			// Treasure room: 1 guardian.
			guardType := "thug"
			if rng.Intn(2) == 0 {
				guardType = "skeleton"
			}
			if m := combat.NewMonsterByType(guardType); m != nil {
				roomMonsters[zone.ID] = []*monster.Monster{m}
			}

		default:
			// Standard chamber: 50% chance of 1-3 low-CR monsters.
			if rng.Intn(2) == 0 {
				continue
			}
			count := 1 + rng.Intn(3)
			var ms []*monster.Monster
			for i := 0; i < count; i++ {
				if m := combat.NewMonsterByType(pickLowCR()); m != nil {
					ms = append(ms, m)
				}
			}
			if len(ms) > 0 {
				roomMonsters[zone.ID] = ms
			}
		}
	}

	return roomMonsters
}

// ── Dungeon summary for Claude ────────────────────────────────────────────────

// buildDungeonSummary produces a human-readable description of the dungeon
// layout and encounters to send to Claude for narrative framing.
func buildDungeonSummary(
	env *environments.EnvironmentData,
	roomMonsters map[string][]*monster.Monster,
	params game.CharacterCreationData,
) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dungeon layout: %d rooms\n", len(env.Zones)))

	for _, zone := range env.Zones {
		// Count connections.
		connectionCount := 0
		for _, p := range env.Passages {
			if p.FromZoneID == zone.ID || (p.Bidirectional && p.ToZoneID == zone.ID) {
				connectionCount++
			}
		}
		// Describe monsters.
		monsterDesc := "no enemies"
		if ms, ok := roomMonsters[zone.ID]; ok && len(ms) > 0 {
			names := make([]string, 0, len(ms))
			for _, m := range ms {
				names = append(names, m.Name())
			}
			monsterDesc = strings.Join(names, ", ")
		}
		sb.WriteString(fmt.Sprintf("  Room ID %s (type: %s, connections: %d): %s\n",
			zone.ID, zone.Type, connectionCount, monsterDesc))
	}

	if params.Name != "" {
		sb.WriteString(fmt.Sprintf("\nPlayer: %s", params.Name))
		if params.ClassID != "" {
			sb.WriteString(fmt.Sprintf(" the %s", params.ClassID))
		}
		if params.RaceID != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", params.RaceID))
		}
	}

	return sb.String()
}

// ── Build DungeonData ─────────────────────────────────────────────────────────

// buildDungeonData converts EnvironmentData + narrative framing into our
// persistent DungeonData struct.
func buildDungeonData(
	env *environments.EnvironmentData,
	framing ai.NarrativeFraming,
	seed int64,
) *game.DungeonData {
	// Build connection map: zoneID → connected zone IDs (deduped).
	connectedTo := make(map[string][]string)
	seen := make(map[string]map[string]bool)
	for _, p := range env.Passages {
		if seen[p.FromZoneID] == nil {
			seen[p.FromZoneID] = make(map[string]bool)
		}
		if seen[p.ToZoneID] == nil {
			seen[p.ToZoneID] = make(map[string]bool)
		}
		if !seen[p.FromZoneID][p.ToZoneID] {
			connectedTo[p.FromZoneID] = append(connectedTo[p.FromZoneID], p.ToZoneID)
			seen[p.FromZoneID][p.ToZoneID] = true
		}
		if p.Bidirectional && !seen[p.ToZoneID][p.FromZoneID] {
			connectedTo[p.ToZoneID] = append(connectedTo[p.ToZoneID], p.FromZoneID)
			seen[p.ToZoneID][p.FromZoneID] = true
		}
	}

	// Identify entrance and boss rooms.
	startRoomID := ""
	bossRoomID := ""
	for _, zone := range env.Zones {
		switch zone.Type {
		case environments.RoomTypeEntrance:
			startRoomID = zone.ID
		case environments.RoomTypeBoss:
			bossRoomID = zone.ID
		}
	}
	if startRoomID == "" && len(env.Zones) > 0 {
		startRoomID = env.Zones[0].ID
	}
	if bossRoomID == "" && len(env.Zones) > 0 {
		bossRoomID = env.Zones[len(env.Zones)-1].ID
	}

	rooms := make(map[string]*game.DungeonRoomData, len(env.Zones))
	for _, zone := range env.Zones {
		name := framing.RoomNames[zone.ID]
		if name == "" {
			name = fallbackRoomName(zone.Type)
		}
		desc := framing.RoomDescriptions[zone.ID] // may be empty for legacy framing calls
		rooms[zone.ID] = &game.DungeonRoomData{
			ID:               zone.ID,
			Name:             name,
			Description:      desc,
			Type:             mapRoomType(zone.Type),
			ConnectedRoomIDs: connectedTo[zone.ID],
		}
	}

	return &game.DungeonData{
		ID:            uuid.New().String(),
		StartRoomID:   startRoomID,
		BossRoomID:    bossRoomID,
		CurrentRoomID: startRoomID,
		Rooms:         rooms,
		RevealedRooms: map[string]bool{startRoomID: true},
		Seed:          seed,
		State:         game.DungeonStateActive,
		CreatedAt:     time.Now(),
	}
}

// mapRoomType converts an environments room type string to our DungeonRoomType.
func mapRoomType(t string) game.DungeonRoomType {
	switch t {
	case environments.RoomTypeEntrance:
		return game.DungeonRoomTypeEntrance
	case environments.RoomTypeBoss:
		return game.DungeonRoomTypeBoss
	case environments.RoomTypeTreasure:
		return game.DungeonRoomTypeTreasure
	case environments.RoomTypeCorridor:
		return game.DungeonRoomTypeCorridor
	case environments.RoomTypeJunction:
		return game.DungeonRoomTypeJunction
	default:
		return game.DungeonRoomTypeChamber
	}
}

// fallbackRoomName returns a generic name when Claude didn't provide one.
func fallbackRoomName(roomType string) string {
	switch roomType {
	case environments.RoomTypeEntrance:
		return "The Entrance"
	case environments.RoomTypeBoss:
		return "The Boss Chamber"
	case environments.RoomTypeTreasure:
		return "The Vault"
	case environments.RoomTypeCorridor:
		return "The Corridor"
	default:
		return "The Chamber"
	}
}

// ── Legacy room bridge ────────────────────────────────────────────────────────

// buildLegacyRooms populates g.Rooms from DungeonData using a BFS spatial
// layout starting from the entrance. Each unvisited neighbour is assigned the
// next available compass direction (clockwise: north, east, south, west) from
// its parent, avoiding direction conflicts. Coordinates are computed from the
// assigned directions so the visual map reflects a meaningful topology.
// The resolved Connections and Coordinates are also written back into
// DungeonData.Rooms so they are persisted and don't need recomputation.
func buildLegacyRooms(g *game.Game, dd *game.DungeonData) {
	if len(dd.Rooms) == 0 {
		return
	}

	// First pass: add all rooms with no connections.
	for _, r := range dd.Rooms {
		area := game.NewArea(r.Name, r.Description)
		area.ID = r.ID
		_ = g.AddRoom(area)
	}

	// BFS from the entrance to assign spatially meaningful compass directions.
	// State: for each visited room, track which directions are already taken.
	takenDirs := make(map[string]map[string]bool) // roomID → set of taken directions
	for id := range dd.Rooms {
		takenDirs[id] = make(map[string]bool)
	}

	// Preferred direction order: clockwise from north.
	dirOrder := []string{"north", "east", "south", "west"}

	startID := dd.StartRoomID
	if startID == "" {
		for id := range dd.Rooms {
			startID = id
			break
		}
	}

	visited := map[string]bool{startID: true}
	queue := []string{startID}

	// Set start room at origin.
	if start, err := g.GetRoom(startID); err == nil {
		start.Coordinates = game.Coordinates{}
		g.UpdateRoom(start)
		if dr, ok := dd.Rooms[startID]; ok {
			dr.Coordinates = game.Coordinates{}
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		r, ok := dd.Rooms[cur]
		if !ok {
			continue
		}

		for _, connID := range r.ConnectedRoomIDs {
			// Find a direction that is free on both sides.
			var chosenDir string
			for _, d := range dirOrder {
				if takenDirs[cur][d] {
					continue
				}
				opp := game.OppositeDirection[d]
				if takenDirs[connID][opp] {
					continue
				}
				chosenDir = d
				break
			}
			if chosenDir == "" {
				// All preferred directions exhausted — skip this edge.
				// The room will still be reachable via another path if the graph
				// is connected; log and continue gracefully.
				log.Printf("buildLegacyRooms: no free direction for edge %s→%s, skipping", cur, connID)
				continue
			}

			// Mark directions as taken on both sides.
			takenDirs[cur][chosenDir] = true
			takenDirs[connID][game.OppositeDirection[chosenDir]] = true

			// Wire the connection (also updates coordinates of connID based on cur).
			if err := g.ConnectRooms(cur, connID, chosenDir); err != nil {
				log.Printf("buildLegacyRooms: ConnectRooms %s→%s (%s): %v", cur, connID, chosenDir, err)
				continue
			}

			// Persist connections and coordinates back into DungeonData.
			if curArea, err := g.GetRoom(cur); err == nil {
				if dr, ok := dd.Rooms[cur]; ok {
					dr.Connections = curArea.Connections
					dr.Coordinates = curArea.Coordinates
				}
			}
			if connArea, err := g.GetRoom(connID); err == nil {
				if dr, ok := dd.Rooms[connID]; ok {
					dr.Connections = connArea.Connections
					dr.Coordinates = connArea.Coordinates
				}
			}

			if !visited[connID] {
				visited[connID] = true
				queue = append(queue, connID)
			}
		}
	}
}

func main() {
	lambda.Start(handler)
}
