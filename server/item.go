package main

import (
	"fmt"
)

type Item struct {
	Name        string
	Description string
	Weight      float64
	Value       int
	Weapon      bool
	Damage      int
	Consumable  bool
	Uses        int
	Location    interface{} // Can be Area or Character
}

// NewItem creates a new item with the given name and description
func NewItem(name, description string) Item {
	return Item{
		Name:        name,
		Description: description,
		Weight:      1.0,
		Value:       1,
		Weapon:      false,
		Damage:      0,
		Consumable:  false,
		Uses:        1,
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

// SetValue sets the item's value
func (i *Item) SetValue(value int) error {
	if value < 0 {
		return fmt.Errorf("value cannot be negative")
	}
	i.Value = value
	return nil
}

// SetWeapon sets whether the item is a weapon and its damage
func (i *Item) SetWeapon(isWeapon bool, damage int) error {
	if isWeapon && damage < 0 {
		return fmt.Errorf("weapon damage cannot be negative")
	}
	i.Weapon = isWeapon
	i.Damage = damage
	return nil
}

// SetConsumable sets whether the item is consumable and its number of uses
func (i *Item) SetConsumable(isConsumable bool, uses int) error {
	if isConsumable && uses < 1 {
		return fmt.Errorf("consumable must have at least 1 use")
	}
	i.Consumable = isConsumable
	i.Uses = uses
	return nil
}

// Use consumes one use of the item if it's consumable
func (i *Item) Use() error {
	if !i.Consumable {
		return fmt.Errorf("item is not consumable")
	}
	if i.Uses <= 0 {
		return fmt.Errorf("item has no uses remaining")
	}
	i.Uses--
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

// GetValue returns the item's value
func (i *Item) GetValue() int {
	return i.Value
}

// IsWeapon returns whether the item is a weapon
func (i *Item) IsWeapon() bool {
	return i.Weapon
}

// GetDamage returns the item's damage if it's a weapon
func (i *Item) GetDamage() int {
	return i.Damage
}

// IsConsumable returns whether the item is consumable
func (i *Item) IsConsumable() bool {
	return i.Consumable
}

// GetUses returns the number of uses remaining
func (i *Item) GetUses() int {
	return i.Uses
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
	itemType := "Item"
	if i.Weapon {
		itemType = "Weapon"
	} else if i.Consumable {
		itemType = "Consumable"
	}

	description := fmt.Sprintf("%s (%s) - %s", i.Name, i.Description, itemType)
	if i.Weapon {
		description += fmt.Sprintf(", Damage: %d", i.Damage)
	}
	if i.Consumable {
		description += fmt.Sprintf(", Uses: %d", i.Uses)
	}
	description += fmt.Sprintf(", Weight: %.1f, Value: %d", i.Weight, i.Value)
	return description
}
