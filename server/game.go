package main

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

func NewGame(gameId uuid.UUID, userID uuid.UUID) Game {
	return Game{
		GameId:    gameId,
		UserId:    userID,
		Map:       make(map[string]Area),
		ItemList:  make(map[string]Item),
		NPCs:      make(map[string]Character),
		MapImages: make(map[string]string),
		Ready:     false,
	}
}

type ChatMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type Game struct {
	GameId      uuid.UUID
	UserId      uuid.UUID
	Player      Character
	Map         map[string]Area
	ItemList    map[string]Item
	NPCs        map[string]Character
	Narrative   []*genai.Content
	ChatHistory []ChatMessage
	MapImages   map[string]string // map of image type -> S3 URL
	Ready       bool
}

// Create a struct that contains all the necessary game state
type SaveState struct {
	SessionID   uuid.UUID            `json:"session_id" dynamodbav:"session_id"`
	UserID      uuid.UUID            `json:"user_id,omitempty" dynamodbav:"user_id,omitempty"`
	Player      Character            `json:"player"`
	Areas       []Area               `json:"areas"`
	Items       []Item               `json:"items"`
	Characters  map[string]Character `json:"characters"`
	Narrative   []genai.Content      `json:"narrative"`
	ChatHistory []ChatMessage        `json:"chat_history"`
	MapImages   map[string]string    `json:"map_images,omitempty" dynamodbav:"map_images,omitempty"`
	Ready       bool                 `json:"ready"`
}

// AddItem adds an item to the game's ItemList
func (g *Game) AddItem(item Item) error {
	if item.Name == "" {
		return fmt.Errorf("item must have a name")
	}
	g.ItemList[item.Name] = item
	return nil
}

// RemoveItem removes an item from the game's ItemList
func (g *Game) RemoveItem(itemName string) error {
	if _, exists := g.ItemList[itemName]; !exists {
		return fmt.Errorf("item %s not found", itemName)
	}
	delete(g.ItemList, itemName)
	return nil
}

// GetItem retrieves an item from the game's ItemList
func (g *Game) GetItem(itemName string) (Item, error) {
	item, exists := g.ItemList[itemName]
	if !exists {
		return Item{}, fmt.Errorf("item %s not found", itemName)
	}
	return item, nil
}

// AddArea adds an area to the game's Map
func (g *Game) AddArea(areaID string, area Area) error {
	if areaID == "" {
		return fmt.Errorf("area ID cannot be empty")
	}
	g.Map[areaID] = area
	return nil
}

// RemoveArea removes an area from the game's Map
func (g *Game) RemoveArea(areaID string) error {
	if _, exists := g.Map[areaID]; !exists {
		return fmt.Errorf("area %s not found", areaID)
	}
	delete(g.Map, areaID)
	return nil
}

// GetArea retrieves an area from the game's Map
func (g *Game) GetArea(areaID string) (Area, error) {
	area, exists := g.Map[areaID]
	if !exists {
		return Area{}, fmt.Errorf("area %s not found", areaID)
	}
	return area, nil
}

// AddNPC adds an NPC to the game's NPCs map
func (g *Game) AddNPC(npc Character) error {
	if npc.Name == "" {
		return fmt.Errorf("NPC must have a name")
	}
	g.NPCs[npc.Name] = npc
	return nil
}

// RemoveNPC removes an NPC from the game's NPCs map
func (g *Game) RemoveNPC(npcName string) error {
	if _, exists := g.NPCs[npcName]; !exists {
		return fmt.Errorf("NPC %s not found", npcName)
	}
	delete(g.NPCs, npcName)
	return nil
}

// GetNPC retrieves an NPC from the game's NPCs map
func (g *Game) GetNPC(npcName string) (Character, error) {
	npc, exists := g.NPCs[npcName]
	if !exists {
		return Character{}, fmt.Errorf("NPC %s not found", npcName)
	}
	return npc, nil
}

// AddItemToArea adds an item to a specific area
func (g *Game) AddItemToArea(areaID string, item Item) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}
	area.Items = append(area.Items, item)
	g.Map[areaID] = area
	return nil
}

// RemoveItemFromArea removes an item from a specific area
func (g *Game) RemoveItemFromArea(areaID string, itemName string) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}

	for i, item := range area.Items {
		if item.Name == itemName {
			area.Items = slices.Delete(area.Items, i, i+1)
			g.Map[areaID] = area
			return nil
		}
	}
	return fmt.Errorf("item %s not found in area %s", itemName, areaID)
}

// AddNPCToArea adds an NPC to a specific area
func (g *Game) AddNPCToArea(areaID string, npcName string) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}

	npc, err := g.GetNPC(npcName)
	if err != nil {
		return err
	}

	area.Occupants = append(area.Occupants, npcName)
	npc.Location = area
	g.Map[areaID] = area
	g.NPCs[npcName] = npc
	return nil
}

// RemoveNPCFromArea removes an NPC from a specific area
func (g *Game) RemoveNPCFromArea(areaID string, npcName string) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}

	for i, occupant := range area.Occupants {
		if occupant == npcName {
			area.Occupants = slices.Delete(area.Occupants, i, i+1)
			g.Map[areaID] = area
			return nil
		}
	}
	return fmt.Errorf("NPC %s not found in area %s", npcName, areaID)
}

// AddItemToInventory adds an item to the player's inventory
func (g *Game) AddItemToInventory(item Item) error {
	// Check if item exists in the game
	if _, err := g.GetItem(item.Name); err != nil {
		return fmt.Errorf("item %s not found in game", item.Name)
	}

	// Add item to player's inventory
	g.Player.Inventory = append(g.Player.Inventory, item)
	return nil
}

// RemoveItemFromInventory removes an item from the player's inventory
func (g *Game) RemoveItemFromInventory(itemName string) error {
	for i, item := range g.Player.Inventory {
		if item.Name == itemName {
			g.Player.Inventory = slices.Delete(g.Player.Inventory, i, i+1)
			return nil
		}
	}
	return fmt.Errorf("item %s not found in inventory", itemName)
}

// GetAllAreas returns all areas in the game
func (g *Game) GetAllAreas() []Area {
	areas := make([]Area, 0, len(g.Map))
	for _, area := range g.Map {
		areas = append(areas, area)
	}
	return areas
}

// SaveGameState writes the current game state to an object for external saving
func (g *Game) SaveGameState() SaveState {
	// Collect all areas
	areas := g.GetAllAreas()

	// Collect all items from areas and player inventory
	items := make([]Item, 0)
	itemMap := make(map[string]bool) // To track unique items
	for _, area := range areas {
		for _, item := range area.GetItems() {
			if !itemMap[item.GetName()] {
				items = append(items, item)
				itemMap[item.GetName()] = true
			}
		}
	}
	for _, item := range g.Player.Inventory {
		if !itemMap[item.GetName()] {
			items = append(items, item)
			itemMap[item.GetName()] = true
		}
	}

	savedNarrative := make([]genai.Content, 0)
	for _, nar := range g.Narrative {
		savedNarrative = append(savedNarrative, *nar)
	}

	// Create save state
	saveState := SaveState{
		SessionID:   g.GameId,
		UserID:      g.UserId,
		Player:      g.Player,
		Areas:       areas,
		Items:       items,
		Characters:  g.NPCs,
		Narrative:   savedNarrative,
		ChatHistory: g.ChatHistory,
		MapImages:   g.MapImages,
		Ready:       g.Ready,
	}

	return saveState
}

// LoadGame loads the game state from a JSON file
func (saveState *SaveState) LoadGame() Game {
	// Restore game state
	g := NewGame(saveState.SessionID, saveState.UserID)
	g.Player = saveState.Player
	g.Map = make(map[string]Area)
	for _, area := range saveState.Areas {
		g.Map[area.ID] = area
	}
	g.ItemList = make(map[string]Item)
	for _, item := range saveState.Items {
		g.ItemList[item.Name] = item
	}
	narrativePtrArr := make([]*genai.Content, 0)
	for _, nar := range saveState.Narrative {
		narrativePtrArr = append(narrativePtrArr, &nar)
	}
	g.NPCs = saveState.Characters
	g.Narrative = narrativePtrArr
	g.ChatHistory = saveState.ChatHistory
	g.MapImages = saveState.MapImages
	if g.MapImages == nil {
		g.MapImages = make(map[string]string)
	}
	g.Ready = saveState.Ready

	fmt.Printf("game state loaded: %+v\n", g)

	return g
}

// CalculateRoomCoordinates calculates coordinates for all rooms based on directional connections
// Starting from the specified root room (typically the player's starting room)
func (g *Game) CalculateRoomCoordinates(startRoomID string) error {
	// Check if start room exists
	startRoom, err := g.GetArea(startRoomID)
	if err != nil {
		return fmt.Errorf("start room %s not found: %w", startRoomID, err)
	}

	// Set starting room at origin
	startRoom.Coordinates = Coordinates{X: 0, Y: 0, Z: 0}
	g.Map[startRoomID] = startRoom

	// Use BFS to calculate coordinates for all connected rooms
	visited := make(map[string]bool)
	queue := []string{startRoomID}
	visited[startRoomID] = true

	// Spacing multiplier for better visualization
	const spacing = 100.0

	for len(queue) > 0 {
		currentRoomID := queue[0]
		queue = queue[1:]

		currentRoom, err := g.GetArea(currentRoomID)
		if err != nil {
			continue
		}

		// Process all connections
		for direction, connectedRoomID := range currentRoom.Connections {
			if visited[connectedRoomID] {
				continue
			}

			connectedRoom, err := g.GetArea(connectedRoomID)
			if err != nil {
				continue
			}

			// Get direction vector
			dirVector, ok := directionVectors[direction]
			if !ok {
				continue
			}

			// Calculate new coordinates based on current room and direction
			connectedRoom.Coordinates = Coordinates{
				X: currentRoom.Coordinates.X + (dirVector.X * spacing),
				Y: currentRoom.Coordinates.Y + (dirVector.Y * spacing),
				Z: currentRoom.Coordinates.Z + (dirVector.Z * spacing),
			}

			g.Map[connectedRoomID] = connectedRoom
			visited[connectedRoomID] = true
			queue = append(queue, connectedRoomID)
		}
	}

	return nil
}

// ConnectRooms connects two rooms bidirectionally with opposite directions
func (g *Game) ConnectRooms(fromRoomID, toRoomID, direction string) error {
	// Validate direction
	dirVector, ok := directionVectors[direction]
	if !ok {
		return fmt.Errorf("invalid direction: %s", direction)
	}

	// Get both rooms
	fromRoom, err := g.GetArea(fromRoomID)
	if err != nil {
		return fmt.Errorf("from room not found: %w", err)
	}

	toRoom, err := g.GetArea(toRoomID)
	if err != nil {
		return fmt.Errorf("to room not found: %w", err)
	}

	// Calculate coordinates for toRoom based on fromRoom's position + direction vector
	// Use a spacing multiplier to spread rooms out on the map
	const spacingMultiplier = 100.0
	toRoom.Coordinates = Coordinates{
		X: fromRoom.Coordinates.X + (dirVector.X * spacingMultiplier),
		Y: fromRoom.Coordinates.Y + (dirVector.Y * spacingMultiplier),
		Z: fromRoom.Coordinates.Z + (dirVector.Z * spacingMultiplier),
	}

	// Add connection from first room to second
	if err := fromRoom.AddConnection(direction, toRoomID); err != nil {
		return fmt.Errorf("failed to add connection: %w", err)
	}

	// Add opposite connection from second room to first
	oppositeDir, ok := oppositeDirection[direction]
	if !ok {
		return fmt.Errorf("no opposite direction for %s", direction)
	}

	if err := toRoom.AddConnection(oppositeDir, fromRoomID); err != nil {
		return fmt.Errorf("failed to add opposite connection: %w", err)
	}

	// Update rooms in map
	g.Map[fromRoomID] = fromRoom
	g.Map[toRoomID] = toRoom

	return nil
}
