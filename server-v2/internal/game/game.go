package game

import (
	"fmt"

	"github.com/google/uuid"
)

// SchemaVersion is incremented whenever SaveState's structure changes
// in a backward-incompatible way. Records with a lower version are
// rejected on load rather than silently corrupted.
const SchemaVersion = 1

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
	UserID            string
	Player            Character
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
		ID:     sessionID,
		UserID: userID,
		Rooms:  make(map[string]Area),
		Items:  make(map[string]Item),
		NPCs:   make(map[string]Character),
	}
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

// CalculateRoomCoordinates runs BFS from the player's starting room to assign
// map coordinates to all reachable rooms.
func (g *Game) CalculateRoomCoordinates() {
	startID := g.Player.LocationID
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

// GiveItemToPlayer moves an item into the player's inventory.
func (g *Game) GiveItemToPlayer(itemID string) error {
	if _, err := g.GetItem(itemID); err != nil {
		return err
	}
	g.removeItemFromAnywhere(itemID)
	return g.Player.AddItemID(itemID)
}

// TakeItemFromPlayer moves an item from the player's inventory into a room.
func (g *Game) TakeItemFromPlayer(itemID, roomID string) error {
	if !g.Player.HasItem(itemID) {
		return fmt.Errorf("player does not have item %s", itemID)
	}
	if err := g.Player.RemoveItemID(itemID); err != nil {
		return err
	}
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
	_ = g.Player.RemoveItemID(itemID)
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

// MovePlayer moves the player to an adjacent room via a direction.
// Returns the destination room and an error if the direction has no exit.
func (g *Game) MovePlayer(direction string) (Area, error) {
	currentRoom, err := g.GetRoom(g.Player.LocationID)
	if err != nil {
		return Area{}, fmt.Errorf("player has no current room: %w", err)
	}
	destID, ok := currentRoom.Connections[direction]
	if !ok {
		return Area{}, fmt.Errorf("no exit to the %s", direction)
	}
	destRoom, err := g.GetRoom(destID)
	if err != nil {
		return Area{}, err
	}
	// Update player location
	_ = currentRoom.RemoveOccupant(g.Player.ID)
	g.Rooms[currentRoom.ID] = currentRoom
	_ = destRoom.AddOccupant(g.Player.ID)
	g.Rooms[destID] = destRoom
	g.Player.LocationID = destID
	return destRoom, nil
}

// PlacePlayer sets the player's starting room during world generation.
func (g *Game) PlacePlayer(roomID string) error {
	room, err := g.GetRoom(roomID)
	if err != nil {
		return err
	}
	// Remove from previous room if set
	if g.Player.LocationID != "" {
		if old, ok := g.Rooms[g.Player.LocationID]; ok {
			_ = old.RemoveOccupant(g.Player.ID)
			g.Rooms[g.Player.LocationID] = old
		}
	}
	if err := room.AddOccupant(g.Player.ID); err != nil {
		// Occupant already there — fine
		_ = err
	}
	g.Rooms[roomID] = room
	g.Player.LocationID = roomID
	return nil
}

// -------------------------------------------------------------------
// Persistence helpers
// -------------------------------------------------------------------

// SaveState is the DynamoDB-serialisable snapshot of a Game.
type SaveState struct {
	SessionID         string                  `json:"session_id" dynamodbav:"session_id"`
	UserID            string                  `json:"user_id" dynamodbav:"user_id"`
	SchemaVersion     int                     `json:"schema_version" dynamodbav:"schema_version"`
	Version           int                     `json:"version" dynamodbav:"version"` // optimistic lock
	Player            Character               `json:"player" dynamodbav:"player"`
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

// ChatMessage is the player-facing chat log entry shown in the UI.
type ChatMessage struct {
	Type    string `json:"type" dynamodbav:"type"` // "player" | "narrative"
	Content string `json:"content" dynamodbav:"content"`
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
		UserID:            g.UserID,
		SchemaVersion:     SchemaVersion,
		Version:           g.Version,
		Player:            g.Player,
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
// Returns an error if the schema version is incompatible.
func FromSaveState(s SaveState) (*Game, error) {
	if s.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("incompatible schema version %d (current: %d)", s.SchemaVersion, SchemaVersion)
	}
	g := &Game{
		ID:                s.SessionID,
		UserID:            s.UserID,
		Player:            s.Player,
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
type GameStateView struct {
	CurrentRoom RoomView            `json:"current_room"`
	Player      CharacterView       `json:"player"`
	Rooms       map[string]RoomView `json:"rooms"`
	ChatHistory []ChatMessage       `json:"chat_history"`
}

// BuildGameStateView constructs a full snapshot from the Game.
func (g *Game) BuildGameStateView(history []ChatMessage) GameStateView {
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

	toRoomView := func(a Area) RoomView {
		occupantViews := make([]CharacterView, 0)
		for _, cid := range a.Occupants {
			if c, ok := g.NPCs[cid]; ok {
				occupantViews = append(occupantViews, CharacterView{
					ID: c.ID, Name: c.Name, Description: c.Description,
					Alive: c.Alive, Health: c.Health, Friendly: c.Friendly,
					Inventory: resolveItems(c.Inventory),
				})
			}
		}
		return RoomView{
			ID: a.ID, Name: a.Name, Description: a.Description,
			Connections: a.Connections, Coordinates: a.Coordinates,
			Items:     resolveItems(a.Items),
			Occupants: occupantViews,
		}
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

	playerEquipment := EquipmentView{
		Head:  resolveSlot(g.Player.Equipment.Head),
		Chest: resolveSlot(g.Player.Equipment.Chest),
		Legs:  resolveSlot(g.Player.Equipment.Legs),
		Hands: resolveSlot(g.Player.Equipment.Hands),
		Feet:  resolveSlot(g.Player.Equipment.Feet),
		Back:  resolveSlot(g.Player.Equipment.Back),
	}

	playerView := CharacterView{
		ID: g.Player.ID, Name: g.Player.Name,
		Description: g.Player.Description,
		Alive:       g.Player.Alive,
		Health:      g.Player.Health,
		Friendly:    g.Player.Friendly,
		Inventory:   resolveItems(g.Player.Inventory),
		Equipment:   playerEquipment,
	}

	roomViews := make(map[string]RoomView, len(g.Rooms))
	for id, room := range g.Rooms {
		roomViews[id] = toRoomView(room)
	}

	var currentRoom RoomView
	if r, ok := g.Rooms[g.Player.LocationID]; ok {
		currentRoom = toRoomView(r)
	}

	return GameStateView{
		CurrentRoom: currentRoom,
		Player:      playerView,
		Rooms:       roomViews,
		ChatHistory: history,
	}
}

// StateDelta holds only what changed between two game states.
// Fields are nil/empty when unchanged.
type StateDelta struct {
	CurrentRoom  *RoomView           `json:"current_room,omitempty"`
	Player       *CharacterView      `json:"player,omitempty"`
	UpdatedRooms map[string]RoomView `json:"updated_rooms,omitempty"`
	NewMessage   *ChatMessage        `json:"new_message,omitempty"`
}
