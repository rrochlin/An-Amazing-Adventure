package main

import (
	"fmt"
	"slices"
)

type Area struct {
	ID          string
	Connections []Area
	Items       []Item
	Occupants   []string
	Description string
}

// NewArea creates a new empty area with a unique ID
func NewArea(id string, description ...string) Area {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	return Area{
		ID:          id,
		Connections: make([]Area, 0),
		Items:       make([]Item, 0),
		Occupants:   make([]string, 0),
		Description: desc,
	}
}

// AddConnection adds a new area connection
func (a *Area) AddConnection(area Area) error {
	// Check if connection already exists
	for _, conn := range a.Connections {
		if conn.ID == area.ID {
			return fmt.Errorf("area already connected")
		}
	}
	a.Connections = append(a.Connections, area)
	return nil
}

// RemoveConnection removes an area connection
func (a *Area) RemoveConnection(area Area) error {
	for i, conn := range a.Connections {
		if conn.ID == area.ID {
			a.Connections = append(a.Connections[:i], a.Connections[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("area not connected")
}

// GetConnections returns all connected areas
func (a *Area) GetConnections() []Area {
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
func (a *Area) GetItems() []Item {
	return a.Items
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
func (a *Area) IsConnected(area Area) bool {
	return slices.ContainsFunc(
		a.Connections,
		func(c Area) bool { return c.ID == area.ID },
	)
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
	ids := make([]string, len(a.Connections))
	for i, conn := range a.Connections {
		ids[i] = conn.ID
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
