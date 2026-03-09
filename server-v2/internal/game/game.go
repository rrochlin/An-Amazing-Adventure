package game

import (
	"fmt"

	"github.com/google/uuid"
)

// SchemaVersion is incremented whenever SaveState's structure changes
// in a backward-incompatible way. FromSaveState handles migration from
// older versions.
const SchemaVersion = 2

// AdventureCreationParams holds the player-provided setup choices that were
// used when the game was created. All fields are optional — the AI fills in
// any blanks. These are persisted so they can be shown on the details page.
type AdventureCreationParams struct {
	PlayerDescription string   `json:"player_description,omitempty" dynamodbav:"player_description,omitempty"`
	PlayerAge         string   `json:"player_age,omitempty" dynamodbav:"player_age,omitempty"`
	PlayerBackstory   string   `json:"player_backstory,omitempty" dynamodbav:"player_backstory,omitempty"`
	ThemeHint         string   `json:"theme_hint,omitempty" dynamodbav:"theme_hint,omitempty"`
	Preferences       []string `json:"preferences,omitempty" dynamodbav:"preferences,omitempty"`
}

// Game is the in-memory representation of a live game session.
// It is never stored directly; SaveState is the DynamoDB-serialisable form.
type Game struct {
	ID                string
	OwnerID           string               // session owner — only they can delete
	UserID            string               // preserved for backward compat (= OwnerID)
	Players           map[string]Character // keyed by userID — all party members
	PartySize         int                  // max party members (default 4)
	InviteCode        string               // active invite code, if any
	Rooms             map[string]Area      // keyed by room ID
	Items             map[string]Item      // keyed by item ID — global item registry
	NPCs              map[string]Character // keyed by character ID
	Ready             bool
	Version           int                     // optimistic locking counter, mirrors SaveState.Version
	Title             string                  // adventure title from blueprint
	Theme             string                  // world theme from blueprint
	QuestGoal         string                  // win condition from blueprint
	TotalTokens       int                     // cumulative Bedrock tokens used
	ConversationCount int                     // number of completed narrator turns
	CreationParams    AdventureCreationParams // player-supplied setup choices
}

// NewGame creates a blank Game with server-generated IDs.
func NewGame(sessionID, userID string) *Game {
	return &Game{
		ID:        sessionID,
		OwnerID:   userID,
		UserID:    userID,
		Players:   make(map[string]Character),
		PartySize: 4,
		Rooms:     make(map[string]Area),
		Items:     make(map[string]Item),
		NPCs:      make(map[string]Character),
	}
}

// -------------------------------------------------------------------
// Party helpers
// -------------------------------------------------------------------

// GetPlayerCharacter returns the character for a specific user.
// Returns zero-value Character and false if user is not in this game.
func (g *Game) GetPlayerCharacter(userID string) (Character, bool) {
	if g.Players == nil {
		return Character{}, false
	}
	c, ok := g.Players[userID]
	return c, ok
}

// SetPlayerCharacter writes a character for a user.
func (g *Game) SetPlayerCharacter(userID string, c Character) {
	if g.Players == nil {
		g.Players = make(map[string]Character)
	}
	g.Players[userID] = c
}

// PlayerInGame returns true if the given user is a party member (owner or joined).
func (g *Game) PlayerInGame(userID string) bool {
	_, ok := g.Players[userID]
	return ok
}

// OwnerCharacter is a convenience accessor for the session owner's character.
// Returns zero-value Character and false if the owner has no character yet.
func (g *Game) OwnerCharacter() (Character, bool) {
	return g.GetPlayerCharacter(g.OwnerID)
}

// -------------------------------------------------------------------
// Room operations
// -------------------------------------------------------------------

// AddRoom inserts a new Area into the world. Returns error on duplicate ID.
func (g *Game) AddRoom(a Area) error {
	if _, exists := g.Rooms[a.ID]; exists {
		return fmt.Errorf("room %s already exists", a.ID)
	}
	g.Rooms[a.ID] = a
	return nil
}

// GetRoom retrieves a room by ID.
func (g *Game) GetRoom(id string) (Area, error) {
	a, ok := g.Rooms[id]
	if !ok {
		return Area{}, fmt.Errorf("room %s not found", id)
	}
	return a, nil
}

// GetRoomByName finds the first room with a matching name (case-sensitive).
// Used by AI tools that reference rooms by name rather than UUID.
func (g *Game) GetRoomByName(name string) (Area, error) {
	for _, a := range g.Rooms {
		if a.Name == name {
			return a, nil
		}
	}
	return Area{}, fmt.Errorf("room named %q not found", name)
}

// UpdateRoom writes a modified Area back into the map.
func (g *Game) UpdateRoom(a Area) {
	g.Rooms[a.ID] = a
}

// DeleteRoom removes a room and cleans up connections from neighbouring rooms.
func (g *Game) DeleteRoom(id string) error {
	if _, ok := g.Rooms[id]; !ok {
		return fmt.Errorf("room %s not found", id)
	}
	// Remove back-references from connected rooms
	for dir, connID := range g.Rooms[id].Connections {
		opp := OppositeDirection[dir]
		if neighbour, ok := g.Rooms[connID]; ok {
			delete(neighbour.Connections, opp)
			g.Rooms[connID] = neighbour
		}
	}
	delete(g.Rooms, id)
	return nil
}

// ConnectRooms creates a bidirectional connection and updates coordinates.
func (g *Game) ConnectRooms(fromID, toID, direction string) error {
	vec, ok := DirectionVectors[direction]
	if !ok {
		return fmt.Errorf("invalid direction: %s", direction)
	}
	from, err := g.GetRoom(fromID)
	if err != nil {
		return fmt.Errorf("from room: %w", err)
	}
	to, err := g.GetRoom(toID)
	if err != nil {
		return fmt.Errorf("to room: %w", err)
	}
	const spacing = 100.0
	to.Coordinates = Coordinates{
		X: from.Coordinates.X + vec.X*spacing,
		Y: from.Coordinates.Y + vec.Y*spacing,
		Z: from.Coordinates.Z + vec.Z*spacing,
	}
	if err := from.ForceConnection(direction, toID); err != nil {
		return err
	}
	opp := OppositeDirection[direction]
	if err := to.ForceConnection(opp, fromID); err != nil {
		return err
	}
	g.Rooms[fromID] = from
	g.Rooms[toID] = to
	return nil
}

// CalculateRoomCoordinates runs BFS from the owner's starting room to assign
// map coordinates to all reachable rooms.
func (g *Game) CalculateRoomCoordinates() {
	owner, ok := g.GetPlayerCharacter(g.OwnerID)
	if !ok {
		return
	}
	startID := owner.LocationID
	if startID == "" {
		return
	}
	start, ok := g.Rooms[startID]
	if !ok {
		return
	}
	start.Coordinates = Coordinates{}
	g.Rooms[startID] = start

	visited := map[string]bool{startID: true}
	queue := []string{startID}
	const spacing = 100.0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curRoom := g.Rooms[cur]
		for dir, nextID := range curRoom.Connections {
			if visited[nextID] {
				continue
			}
			vec, ok := DirectionVectors[dir]
			if !ok {
				continue
			}
			next := g.Rooms[nextID]
			next.Coordinates = Coordinates{
				X: curRoom.Coordinates.X + vec.X*spacing,
				Y: curRoom.Coordinates.Y + vec.Y*spacing,
				Z: curRoom.Coordinates.Z + vec.Z*spacing,
			}
			g.Rooms[nextID] = next
			visited[nextID] = true
			queue = append(queue, nextID)
		}
	}
}

// -------------------------------------------------------------------
// Item operations
// -------------------------------------------------------------------

// AddItem registers a new item in the global registry.
func (g *Game) AddItem(item Item) error {
	if _, exists := g.Items[item.ID]; exists {
		return fmt.Errorf("item %s already exists", item.ID)
	}
	g.Items[item.ID] = item
	return nil
}

// GetItem retrieves an item by ID.
func (g *Game) GetItem(id string) (Item, error) {
	item, ok := g.Items[id]
	if !ok {
		return Item{}, fmt.Errorf("item %s not found", id)
	}
	return item, nil
}

// GetItemByName finds the first item with a matching name.
func (g *Game) GetItemByName(name string) (Item, error) {
	for _, item := range g.Items {
		if item.Name == name {
			return item, nil
		}
	}
	return Item{}, fmt.Errorf("item named %q not found", name)
}

// PlaceItemInRoom moves an item from wherever it is into a room.
// If the item is currently in a character's inventory or another room it is
// removed from there first.
func (g *Game) PlaceItemInRoom(itemID, roomID string) error {
	if _, err := g.GetItem(itemID); err != nil {
		return err
	}
	room, err := g.GetRoom(roomID)
	if err != nil {
		return err
	}
	g.removeItemFromAnywhere(itemID)
	if err := room.AddItemID(itemID); err != nil {
		return err
	}
	g.Rooms[roomID] = room
	return nil
}

// GiveItemToPlayer moves an item into the owner's inventory.
// For party sessions, items are given to the session owner.
func (g *Game) GiveItemToPlayer(itemID string) error {
	if _, err := g.GetItem(itemID); err != nil {
		return err
	}
	g.removeItemFromAnywhere(itemID)
	owner, ok := g.GetPlayerCharacter(g.OwnerID)
	if !ok {
		return fmt.Errorf("owner character not found")
	}
	if err := owner.AddItemID(itemID); err != nil {
		return err
	}
	g.Players[g.OwnerID] = owner
	return nil
}

// GiveItemToCharacter moves an item into the specified player's inventory.
func (g *Game) GiveItemToCharacter(itemID, userID string) error {
	if _, err := g.GetItem(itemID); err != nil {
		return err
	}
	player, ok := g.GetPlayerCharacter(userID)
	if !ok {
		return fmt.Errorf("player %s not found in game", userID)
	}
	g.removeItemFromAnywhere(itemID)
	if err := player.AddItemID(itemID); err != nil {
		return err
	}
	g.Players[userID] = player
	return nil
}

// TakeItemFromPlayer moves an item from the owner's inventory into a room.
func (g *Game) TakeItemFromPlayer(itemID, roomID string) error {
	owner, ok := g.GetPlayerCharacter(g.OwnerID)
	if !ok {
		return fmt.Errorf("owner character not found")
	}
	if !owner.HasItem(itemID) {
		return fmt.Errorf("player does not have item %s", itemID)
	}
	if err := owner.RemoveItemID(itemID); err != nil {
		return err
	}
	g.Players[g.OwnerID] = owner
	room, err := g.GetRoom(roomID)
	if err != nil {
		return err
	}
	if err := room.AddItemID(itemID); err != nil {
		return err
	}
	g.Rooms[roomID] = room
	return nil
}

// removeItemFromAnywhere removes an item ID from every room and character
// inventory it might currently be in (brute-force scan; game worlds are small).
func (g *Game) removeItemFromAnywhere(itemID string) {
	for id, room := range g.Rooms {
		if room.HasItem(itemID) {
			_ = room.RemoveItemID(itemID)
			g.Rooms[id] = room
		}
	}
	for uid, player := range g.Players {
		if player.HasItem(itemID) {
			_ = player.RemoveItemID(itemID)
			g.Players[uid] = player
		}
	}
	for id, npc := range g.NPCs {
		if npc.HasItem(itemID) {
			_ = npc.RemoveItemID(itemID)
			g.NPCs[id] = npc
		}
	}
}

// -------------------------------------------------------------------
// NPC operations
// -------------------------------------------------------------------

// AddNPC registers a new NPC.
func (g *Game) AddNPC(c Character) error {
	if _, exists := g.NPCs[c.ID]; exists {
		return fmt.Errorf("NPC %s already exists", c.ID)
	}
	g.NPCs[c.ID] = c
	return nil
}

// GetNPC retrieves an NPC by ID.
func (g *Game) GetNPC(id string) (Character, error) {
	c, ok := g.NPCs[id]
	if !ok {
		return Character{}, fmt.Errorf("NPC %s not found", id)
	}
	return c, nil
}

// GetNPCByName finds the first NPC with a matching name.
func (g *Game) GetNPCByName(name string) (Character, error) {
	for _, c := range g.NPCs {
		if c.Name == name {
			return c, nil
		}
	}
	return Character{}, fmt.Errorf("NPC named %q not found", name)
}

// MoveNPC moves an NPC from its current room to a target room.
func (g *Game) MoveNPC(npcID, roomID string) error {
	npc, err := g.GetNPC(npcID)
	if err != nil {
		return err
	}
	// Remove from old room
	if npc.LocationID != "" {
		if old, ok := g.Rooms[npc.LocationID]; ok {
			_ = old.RemoveOccupant(npcID)
			g.Rooms[npc.LocationID] = old
		}
	}
	// Place in new room
	room, err := g.GetRoom(roomID)
	if err != nil {
		return err
	}
	if err := room.AddOccupant(npcID); err != nil {
		return err
	}
	npc.LocationID = roomID
	g.Rooms[roomID] = room
	g.NPCs[npcID] = npc
	return nil
}

// -------------------------------------------------------------------
// Player movement
// -------------------------------------------------------------------

// MovePlayer moves the owner's character to an adjacent room via a direction.
// Returns the destination room and an error if the direction has no exit.
func (g *Game) MovePlayer(direction string) (Area, error) {
	return g.MoveCharacter(g.OwnerID, direction)
}

// MoveCharacter moves a specific player's character to an adjacent room.
// Returns the destination room and an error if the direction has no exit.
func (g *Game) MoveCharacter(userID, direction string) (Area, error) {
	player, ok := g.GetPlayerCharacter(userID)
	if !ok {
		return Area{}, fmt.Errorf("player %s not found in game", userID)
	}
	currentRoom, err := g.GetRoom(player.LocationID)
	if err != nil {
		return Area{}, fmt.Errorf("player has no current room: %w", err)
	}
	destID, connOK := currentRoom.Connections[direction]
	if !connOK {
		return Area{}, fmt.Errorf("no exit to the %s", direction)
	}
	destRoom, err := g.GetRoom(destID)
	if err != nil {
		return Area{}, err
	}
	_ = currentRoom.RemoveOccupant(player.ID)
	g.Rooms[currentRoom.ID] = currentRoom
	_ = destRoom.AddOccupant(player.ID)
	g.Rooms[destID] = destRoom
	player.LocationID = destID
	g.Players[userID] = player
	return destRoom, nil
}

// PlacePlayer sets the owner's starting room during world generation.
func (g *Game) PlacePlayer(roomID string) error {
	return g.PlaceCharacter(g.OwnerID, roomID)
}

// PlaceCharacter sets a specific player's starting room.
func (g *Game) PlaceCharacter(userID, roomID string) error {
	room, err := g.GetRoom(roomID)
	if err != nil {
		return err
	}
	player, ok := g.GetPlayerCharacter(userID)
	if !ok {
		return fmt.Errorf("player %s not found in game", userID)
	}
	// Remove from previous room if set
	if player.LocationID != "" {
		if old, exists := g.Rooms[player.LocationID]; exists {
			_ = old.RemoveOccupant(player.ID)
			g.Rooms[player.LocationID] = old
		}
	}
	if err := room.AddOccupant(player.ID); err != nil {
		// Occupant already there — fine
		_ = err
	}
	g.Rooms[roomID] = room
	player.LocationID = roomID
	g.Players[userID] = player
	return nil
}

// -------------------------------------------------------------------
// Persistence helpers
// -------------------------------------------------------------------

// SaveState is the DynamoDB-serialisable snapshot of a Game.
type SaveState struct {
	SessionID         string                  `json:"session_id" dynamodbav:"session_id"`
	OwnerID           string                  `json:"owner_id,omitempty" dynamodbav:"owner_id,omitempty"`
	UserID            string                  `json:"user_id" dynamodbav:"user_id"` // preserved for backward compat
	SchemaVersion     int                     `json:"schema_version" dynamodbav:"schema_version"`
	Version           int                     `json:"version" dynamodbav:"version"`                     // optimistic lock
	Players           map[string]Character    `json:"players,omitempty" dynamodbav:"players,omitempty"` // v2+: keyed by userID
	Player            Character               `json:"player,omitempty" dynamodbav:"player,omitempty"`   // v1 compat only
	PartySize         int                     `json:"party_size,omitempty" dynamodbav:"party_size,omitempty"`
	InviteCode        string                  `json:"invite_code,omitempty" dynamodbav:"invite_code,omitempty"`
	Rooms             []Area                  `json:"rooms" dynamodbav:"rooms"`
	Items             []Item                  `json:"items" dynamodbav:"items"`
	NPCs              []Character             `json:"npcs" dynamodbav:"npcs"`
	Narrative         []NarrativeMessage      `json:"narrative" dynamodbav:"narrative"`
	ChatHistory       []ChatMessage           `json:"chat_history" dynamodbav:"chat_history"`
	Ready             bool                    `json:"ready" dynamodbav:"ready"`
	Title             string                  `json:"title,omitempty" dynamodbav:"title,omitempty"`
	Theme             string                  `json:"theme,omitempty" dynamodbav:"theme,omitempty"`
	QuestGoal         string                  `json:"quest_goal,omitempty" dynamodbav:"quest_goal,omitempty"`
	TotalTokens       int                     `json:"total_tokens,omitempty" dynamodbav:"total_tokens,omitempty"`
	ConversationCount int                     `json:"conversation_count,omitempty" dynamodbav:"conversation_count,omitempty"`
	CreationParams    AdventureCreationParams `json:"creation_params,omitempty" dynamodbav:"creation_params,omitempty"`
}

// NarrativeMessage stores a single turn of Bedrock conversation history.
type NarrativeMessage struct {
	Role    string           `json:"role" dynamodbav:"role"` // "user" | "assistant"
	Content []NarrativeBlock `json:"content" dynamodbav:"content"`
}

// NarrativeBlock holds a single content block within a message.
type NarrativeBlock struct {
	Type string `json:"type" dynamodbav:"type"` // "text" | "tool_use" | "tool_result"
	Text string `json:"text,omitempty" dynamodbav:"text,omitempty"`
}

// WorldEvent is a player-visible description of a game state mutation that
// occurred during a narrator turn. Events are only produced when the player
// can observe the change (see visibility table in docs/TODO.md).
type WorldEvent struct {
	Type    string `json:"type" dynamodbav:"type"`       // "damage","heal","death","revive","item_gained","item_lost","item_appeared","character_arrived","character_departed"
	Message string `json:"message" dynamodbav:"message"` // human-readable, player's perspective
}

// MutationEntry is a durable audit log record written to the mutations table
// for every DispatchTool call, regardless of player visibility.
type MutationEntry struct {
	SessionID string         `json:"session_id" dynamodbav:"session_id"`
	Ts        int64          `json:"ts" dynamodbav:"ts"`     // unix milliseconds — sort key
	Turn      int            `json:"turn" dynamodbav:"turn"` // g.ConversationCount at dispatch time
	Tool      string         `json:"tool" dynamodbav:"tool"`
	Input     map[string]any `json:"input" dynamodbav:"input"`
	Result    string         `json:"result" dynamodbav:"result"`
}

// ChatMessage is the player-facing chat log entry shown in the UI.
type ChatMessage struct {
	Type    string       `json:"type" dynamodbav:"type"` // "player" | "narrative"
	Content string       `json:"content" dynamodbav:"content"`
	Events  []WorldEvent `json:"events,omitempty" dynamodbav:"events,omitempty"` // non-nil on narrative messages when world events occurred
}

// ToSaveState serialises the Game to a DynamoDB-ready SaveState.
func (g *Game) ToSaveState(narrative []NarrativeMessage, history []ChatMessage) SaveState {
	rooms := make([]Area, 0, len(g.Rooms))
	for _, r := range g.Rooms {
		rooms = append(rooms, r)
	}
	items := make([]Item, 0, len(g.Items))
	for _, i := range g.Items {
		items = append(items, i)
	}
	npcs := make([]Character, 0, len(g.NPCs))
	for _, n := range g.NPCs {
		npcs = append(npcs, n)
	}
	return SaveState{
		SessionID:         g.ID,
		OwnerID:           g.OwnerID,
		UserID:            g.UserID,
		SchemaVersion:     SchemaVersion,
		Version:           g.Version,
		Players:           g.Players,
		PartySize:         g.PartySize,
		InviteCode:        g.InviteCode,
		Rooms:             rooms,
		Items:             items,
		NPCs:              npcs,
		Narrative:         narrative,
		ChatHistory:       history,
		Ready:             g.Ready,
		Title:             g.Title,
		Theme:             g.Theme,
		QuestGoal:         g.QuestGoal,
		TotalTokens:       g.TotalTokens,
		ConversationCount: g.ConversationCount,
		CreationParams:    g.CreationParams,
	}
}

// FromSaveState restores a Game from a SaveState.
// Supports schema version 1 (solo) and 2 (party). Returns error for unknown
// future versions.
func FromSaveState(s SaveState) (*Game, error) {
	if s.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("unsupported schema version %d (current: %d)", s.SchemaVersion, SchemaVersion)
	}
	ownerID := s.OwnerID
	if ownerID == "" {
		ownerID = s.UserID // v1 migration
	}
	partySize := s.PartySize
	if partySize == 0 {
		partySize = 4 // default
	}
	g := &Game{
		ID:                s.SessionID,
		OwnerID:           ownerID,
		UserID:            s.UserID,
		PartySize:         partySize,
		InviteCode:        s.InviteCode,
		Ready:             s.Ready,
		Version:           s.Version,
		Rooms:             make(map[string]Area, len(s.Rooms)),
		Items:             make(map[string]Item, len(s.Items)),
		NPCs:              make(map[string]Character, len(s.NPCs)),
		Title:             s.Title,
		Theme:             s.Theme,
		QuestGoal:         s.QuestGoal,
		TotalTokens:       s.TotalTokens,
		ConversationCount: s.ConversationCount,
		CreationParams:    s.CreationParams,
	}
	// v1 had a single Player field; migrate it into the Players map.
	if s.SchemaVersion <= 1 {
		g.Players = map[string]Character{ownerID: s.Player}
	} else {
		g.Players = s.Players
		if g.Players == nil {
			g.Players = make(map[string]Character)
		}
	}
	for _, r := range s.Rooms {
		g.Rooms[r.ID] = r
	}
	for _, i := range s.Items {
		g.Items[i.ID] = i
	}
	for _, n := range s.NPCs {
		g.NPCs[n.ID] = n
	}
	return g, nil
}

// NewSessionID returns a new random UUID string for use as a session ID.
func NewSessionID() string {
	return uuid.NewString()
}

// -------------------------------------------------------------------
// View helpers (used to build the wire GameState sent to the client)
// -------------------------------------------------------------------

// RoomView is the client-facing representation of a room.
type RoomView struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Connections map[string]string `json:"connections"`
	Coordinates Coordinates       `json:"coordinates"`
	Items       []ItemView        `json:"items"`
	Occupants   []CharacterView   `json:"occupants"`
}

// ItemView is the client-facing representation of an item.
type ItemView struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Weight      float64       `json:"weight"`
	Equippable  bool          `json:"equippable"`
	Slot        EquipmentSlot `json:"slot,omitempty"`
}

// EquipmentView is the client-facing representation of a character's equipment.
// Each field is nil when the slot is empty.
type EquipmentView struct {
	Head  *ItemView `json:"head,omitempty"`
	Chest *ItemView `json:"chest,omitempty"`
	Legs  *ItemView `json:"legs,omitempty"`
	Hands *ItemView `json:"hands,omitempty"`
	Feet  *ItemView `json:"feet,omitempty"`
	Back  *ItemView `json:"back,omitempty"`
}

// CharacterView is the client-facing representation of a character.
type CharacterView struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Alive       bool          `json:"alive"`
	Health      int           `json:"health"`
	Friendly    bool          `json:"friendly"`
	Inventory   []ItemView    `json:"inventory"`
	Equipment   EquipmentView `json:"equipment"`
}

// GameStateView is the full snapshot sent to the client on load or update.
// Self is the calling user's own character; Party contains all other members.
type GameStateView struct {
	CurrentRoom RoomView            `json:"current_room"`
	Player      CharacterView       `json:"player"` // kept for backward compat — same as Self
	Self        CharacterView       `json:"self"`
	Party       []CharacterView     `json:"party"`
	Rooms       map[string]RoomView `json:"rooms"`
	ChatHistory []ChatMessage       `json:"chat_history"`
}

// buildCharacterView constructs a CharacterView for a given character.
func (g *Game) buildCharacterView(c Character) CharacterView {
	resolveItems := func(ids []string) []ItemView {
		views := make([]ItemView, 0, len(ids))
		for _, id := range ids {
			if item, ok := g.Items[id]; ok {
				views = append(views, ItemView{
					ID: item.ID, Name: item.Name,
					Description: item.Description,
					Weight:      item.Weight,
					Equippable:  item.Equippable,
					Slot:        item.Slot,
				})
			}
		}
		return views
	}
	resolveSlot := func(idPtr *string) *ItemView {
		if idPtr == nil {
			return nil
		}
		if item, ok := g.Items[*idPtr]; ok {
			v := ItemView{
				ID: item.ID, Name: item.Name,
				Description: item.Description,
				Weight:      item.Weight,
				Equippable:  item.Equippable,
				Slot:        item.Slot,
			}
			return &v
		}
		return nil
	}
	return CharacterView{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Alive:       c.Alive,
		Health:      c.Health,
		Friendly:    c.Friendly,
		Inventory:   resolveItems(c.Inventory),
		Equipment: EquipmentView{
			Head:  resolveSlot(c.Equipment.Head),
			Chest: resolveSlot(c.Equipment.Chest),
			Legs:  resolveSlot(c.Equipment.Legs),
			Hands: resolveSlot(c.Equipment.Hands),
			Feet:  resolveSlot(c.Equipment.Feet),
			Back:  resolveSlot(c.Equipment.Back),
		},
	}
}

// BuildGameStateView constructs a full snapshot from the Game for a specific
// caller. Self is the caller's own character; Party contains all other members.
// If callerUserID is empty or not found, falls back to the owner's character.
func (g *Game) BuildGameStateView(callerUserID string, history []ChatMessage) GameStateView {
	toRoomView := func(a Area) RoomView {
		occupantViews := make([]CharacterView, 0)
		for _, cid := range a.Occupants {
			if c, ok := g.NPCs[cid]; ok {
				occupantViews = append(occupantViews, g.buildCharacterView(c))
			}
		}
		return RoomView{
			ID: a.ID, Name: a.Name, Description: a.Description,
			Connections: a.Connections, Coordinates: a.Coordinates,
			Items:     g.buildItemViews(a.Items),
			Occupants: occupantViews,
		}
	}

	caller, ok := g.GetPlayerCharacter(callerUserID)
	if !ok {
		caller, _ = g.OwnerCharacter()
	}
	selfView := g.buildCharacterView(caller)

	party := make([]CharacterView, 0, len(g.Players)-1)
	for uid, char := range g.Players {
		if uid == callerUserID {
			continue
		}
		party = append(party, g.buildCharacterView(char))
	}

	roomViews := make(map[string]RoomView, len(g.Rooms))
	for id, room := range g.Rooms {
		roomViews[id] = toRoomView(room)
	}

	var currentRoom RoomView
	if r, ok := g.Rooms[caller.LocationID]; ok {
		currentRoom = toRoomView(r)
	}

	return GameStateView{
		CurrentRoom: currentRoom,
		Player:      selfView, // backward compat
		Self:        selfView,
		Party:       party,
		Rooms:       roomViews,
		ChatHistory: history,
	}
}

// buildItemViews resolves a list of item IDs into ItemView slices.
func (g *Game) buildItemViews(ids []string) []ItemView {
	views := make([]ItemView, 0, len(ids))
	for _, id := range ids {
		if item, ok := g.Items[id]; ok {
			views = append(views, ItemView{
				ID: item.ID, Name: item.Name,
				Description: item.Description,
				Weight:      item.Weight,
				Equippable:  item.Equippable,
				Slot:        item.Slot,
			})
		}
	}
	return views
}

// StateDelta holds only what changed between two game states.
// Fields are nil/empty when unchanged.
// NewMessage was removed — narrative text reaches the client via streaming
// (narrative_chunk / narrative_end frames), so including it in state_delta
// caused duplicate messages in the chat log.
type StateDelta struct {
	CurrentRoom  *RoomView           `json:"current_room,omitempty"`
	Player       *CharacterView      `json:"player,omitempty"` // backward compat — same as Self
	Self         *CharacterView      `json:"self,omitempty"`
	Party        []CharacterView     `json:"party,omitempty"` // updated party member views
	UpdatedRooms map[string]RoomView `json:"updated_rooms,omitempty"`
	Events       []WorldEvent        `json:"events,omitempty"` // player-visible world events this turn
}
