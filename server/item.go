package main

import (
	"fmt"
)

type Item struct {
	Name        string
	Description string
	Weight      float64
	Location    interface{} // Can be Area or Character
}

// NewItem creates a new item with the given name and description
func NewItem(name, description string) Item {
	return Item{
		Name:        name,
		Description: description,
		Weight:      1.0,
	}
}

// SetName sets the item's name
func (i *Item) SetName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	i.Name = name
	return nil
}

// SetDescription sets the item's description
func (i *Item) SetDescription(description string) {
	i.Description = description
}

// SetWeight sets the item's weight
func (i *Item) SetWeight(weight float64) error {
	if weight < 0 {
		return fmt.Errorf("weight cannot be negative")
	}
	i.Weight = weight
	return nil
}

// GetName returns the item's name
func (i *Item) GetName() string {
	return i.Name
}

// GetDescription returns the item's description
func (i *Item) GetDescription() string {
	return i.Description
}

// GetWeight returns the item's weight
func (i *Item) GetWeight() float64 {
	return i.Weight
}

// SetLocation sets the item's location
func (i *Item) SetLocation(location interface{}) {
	i.Location = location
}

// GetLocation returns the item's current location
func (i *Item) GetLocation() interface{} {
	return i.Location
}

// String returns a string representation of the item
func (i *Item) String() string {

	description := fmt.Sprintf("%s (%s)", i.Name, i.Description)
	description += fmt.Sprintf(", Weight: %.1f", i.Weight)
	return description
}
