package game

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// Coordinates holds the 2.5D canvas position of a room for map rendering.
type Coordinates struct {
	X float64 `json:"x" dynamodbav:"x"`
	Y float64 `json:"y" dynamodbav:"y"`
	Z float64 `json:"z" dynamodbav:"z"` // floor level
}

// Area represents a single room in the game world.
// IDs are server-generated UUIDs — the AI uses room names to reference rooms.
type Area struct {
	ID          string            `json:"id" dynamodbav:"id"`
	Name        string            `json:"name" dynamodbav:"name"`
	Description string            `json:"description" dynamodbav:"description"`
	Connections map[string]string `json:"connections" dynamodbav:"connections"` // direction -> room ID
	Coordinates Coordinates       `json:"coordinates" dynamodbav:"coordinates"`
	Items       []string          `json:"items" dynamodbav:"items"`         // item IDs
	Occupants   []string          `json:"occupants" dynamodbav:"occupants"` // character IDs
}

// NewArea creates a new Area with a server-generated UUID.
func NewArea(name, description string) Area {
	return Area{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Connections: make(map[string]string),
		Items:       []string{},
		Occupants:   []string{},
	}
}

// DirectionVectors maps compass directions to coordinate deltas.
var DirectionVectors = map[string]Coordinates{
	"north":     {X: 0, Y: -1, Z: 0},
	"south":     {X: 0, Y: 1, Z: 0},
	"east":      {X: 1, Y: 0, Z: 0},
	"west":      {X: -1, Y: 0, Z: 0},
	"northeast": {X: 0.707, Y: -0.707, Z: 0},
	"northwest": {X: -0.707, Y: -0.707, Z: 0},
	"southeast": {X: 0.707, Y: 0.707, Z: 0},
	"southwest": {X: -0.707, Y: 0.707, Z: 0},
	"up":        {X: 0, Y: 0, Z: 1},
	"down":      {X: 0, Y: 0, Z: -1},
}

// OppositeDirection maps each direction to its inverse.
var OppositeDirection = map[string]string{
	"north":     "south",
	"south":     "north",
	"east":      "west",
	"west":      "east",
	"northeast": "southwest",
	"northwest": "southeast",
	"southeast": "northwest",
	"southwest": "northeast",
	"up":        "down",
	"down":      "up",
}

// AddConnection registers a directional exit. Returns error if direction is
// already occupied (use ForceConnection if overwrite is needed).
func (a *Area) AddConnection(direction, roomID string) error {
	if _, ok := DirectionVectors[direction]; !ok {
		return fmt.Errorf("invalid direction: %s", direction)
	}
	if existing, exists := a.Connections[direction]; exists {
		return fmt.Errorf("direction %s already connected to %s", direction, existing)
	}
	a.Connections[direction] = roomID
	return nil
}

// ForceConnection sets a directional exit, overwriting any existing connection.
func (a *Area) ForceConnection(direction, roomID string) error {
	if _, ok := DirectionVectors[direction]; !ok {
		return fmt.Errorf("invalid direction: %s", direction)
	}
	a.Connections[direction] = roomID
	return nil
}

// RemoveConnection removes a directional exit.
func (a *Area) RemoveConnection(direction string) error {
	if _, exists := a.Connections[direction]; !exists {
		return fmt.Errorf("no connection in direction %s", direction)
	}
	delete(a.Connections, direction)
	return nil
}

// AddItemID records an item (by ID) as present in this room.
func (a *Area) AddItemID(itemID string) error {
	if slices.Contains(a.Items, itemID) {
		return fmt.Errorf("item %s already in room", itemID)
	}
	a.Items = append(a.Items, itemID)
	return nil
}

// RemoveItemID removes an item ID from the room.
func (a *Area) RemoveItemID(itemID string) error {
	idx := slices.Index(a.Items, itemID)
	if idx < 0 {
		return fmt.Errorf("item %s not in room", itemID)
	}
	a.Items = slices.Delete(a.Items, idx, idx+1)
	return nil
}

// HasItem checks whether an item ID is present in the room.
func (a *Area) HasItem(itemID string) bool {
	return slices.Contains(a.Items, itemID)
}

// AddOccupant records a character (by ID) as being in this room.
func (a *Area) AddOccupant(charID string) error {
	if slices.Contains(a.Occupants, charID) {
		return fmt.Errorf("character %s already in room", charID)
	}
	a.Occupants = append(a.Occupants, charID)
	return nil
}

// RemoveOccupant removes a character ID from the room.
func (a *Area) RemoveOccupant(charID string) error {
	idx := slices.Index(a.Occupants, charID)
	if idx < 0 {
		return fmt.Errorf("character %s not in room", charID)
	}
	a.Occupants = slices.Delete(a.Occupants, idx, idx+1)
	return nil
}

// HasOccupant checks whether a character ID is present in the room.
func (a *Area) HasOccupant(charID string) bool {
	return slices.Contains(a.Occupants, charID)
}
