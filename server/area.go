package main

import (
	"fmt"
	"slices"
)

type Coordinates struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"` // For vertical levels (up/down)
}

type Area struct {
	ID          string            `json:"id"`
	Connections map[string]string `json:"connections"` // direction -> room_id
	Coordinates Coordinates       `json:"coordinates"`
	Items       []Item            `json:"items"`
	Occupants   []string          `json:"occupants"`
	Description string            `json:"description"`
}

// NewArea creates a new empty area with a unique ID
func NewArea(id string, description ...string) Area {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	return Area{
		ID:          id,
		Connections: make(map[string]string),
		Items:       make([]Item, 0),
		Occupants:   make([]string, 0),
		Description: desc,
		Coordinates: Coordinates{X: 0, Y: 0, Z: 0},
	}
}

// Direction vectors for coordinate calculation
var directionVectors = map[string]Coordinates{
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

// Opposite directions for bidirectional connections
var oppositeDirection = map[string]string{
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

// AddConnection adds a new area connection with a direction
func (a *Area) AddConnection(direction string, roomID string) error {
	// Validate direction
	if _, ok := directionVectors[direction]; !ok {
		return fmt.Errorf("invalid direction: %s", direction)
	}
	// Check if direction already has a connection
	if existingID, exists := a.Connections[direction]; exists {
		return fmt.Errorf("direction %s already connected to %s", direction, existingID)
	}
	a.Connections[direction] = roomID
	return nil
}

// RemoveConnection removes an area connection in a specific direction
func (a *Area) RemoveConnection(direction string) error {
	if _, exists := a.Connections[direction]; !exists {
		return fmt.Errorf("no connection in direction %s", direction)
	}
	delete(a.Connections, direction)
	return nil
}

// GetConnections returns all connected room IDs
func (a *Area) GetConnections() []string {
	connections := make([]string, 0, len(a.Connections))
	for _, roomID := range a.Connections {
		connections = append(connections, roomID)
	}
	return connections
}

// GetConnectionsMap returns the directional connections map
func (a *Area) GetConnectionsMap() map[string]string {
	return a.Connections
}

// AddItem adds an item to the area
func (a *Area) AddItem(item Item) error {
	for _, existingItem := range a.Items {
		if existingItem.Name == item.Name {
			return fmt.Errorf("item already exists in area")
		}
	}
	a.Items = append(a.Items, item)
	return nil
}

// RemoveItem removes an item from the area
func (a *Area) RemoveItem(itemName string) error {
	for i, item := range a.Items {
		if item.Name == itemName {
			a.Items = append(a.Items[:i], a.Items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("item not found in area")
}

// GetItems returns all items in the area
func (a *Area) GetItems() map[string]Item {
	res := map[string]Item{}
	for _, item := range a.Items {
		res[item.Name] = item
	}
	return res
}

// AddOccupant adds an occupant to the area
func (a *Area) AddOccupant(name string) error {
	// Check if occupant already exists
	if slices.Contains(a.Occupants, name) {
		return fmt.Errorf("occupant already in area")
	}
	a.Occupants = append(a.Occupants, name)
	return nil
}

// RemoveOccupant removes an occupant from the area
func (a *Area) RemoveOccupant(name string) error {
	for i, occupant := range a.Occupants {
		if occupant == name {
			a.Occupants = append(a.Occupants[:i], a.Occupants[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("occupant not found in area")
}

// GetOccupants returns all occupants in the area
func (a *Area) GetOccupants() []string {
	return a.Occupants
}

// HasItem checks if an item exists in the area
func (a *Area) HasItem(itemName string) bool {
	return slices.ContainsFunc(
		a.Items,
		func(o Item) bool { return o.Name == itemName },
	)
}

// HasOccupant checks if an occupant is in the area
func (a *Area) HasOccupant(name string) bool {
	return slices.Contains(a.Occupants, name)
}

// IsConnected checks if an area is connected
func (a *Area) IsConnected(roomID string) bool {
	for _, connectedID := range a.Connections {
		if connectedID == roomID {
			return true
		}
	}
	return false
}

// GetDirectionTo returns the direction to a connected room
func (a *Area) GetDirectionTo(roomID string) (string, error) {
	for direction, connectedID := range a.Connections {
		if connectedID == roomID {
			return direction, nil
		}
	}
	return "", fmt.Errorf("room %s is not connected", roomID)
}

// String returns a string representation of the area
func (a *Area) String() string {
	return fmt.Sprintf("Area %s with %d connections, %d items, and %d occupants",
		a.ID, len(a.Connections), len(a.Items), len(a.Occupants))
}

// GetDescription returns the description of the area
func (a *Area) GetDescription() string {
	return a.Description
}

// GetConnectionIDs returns a slice of connected area IDs
func (a *Area) GetConnectionIDs() []string {
	ids := make([]string, 0, len(a.Connections))
	for _, roomID := range a.Connections {
		ids = append(ids, roomID)
	}
	return ids
}

// GetItemNames returns a slice of item names in the area
func (a *Area) GetItemNames() []string {
	names := make([]string, len(a.Items))
	for i, item := range a.Items {
		names[i] = item.Name
	}
	return names
}

// GetOccupantNames returns a slice of occupant names in the area
func (a *Area) GetOccupantNames() []string {
	return a.Occupants
}
