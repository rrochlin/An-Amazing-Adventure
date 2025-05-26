package main

import (
	"fmt"
)

type Character struct {
	Location    Area
	Name        string
	Description string
	Alive       bool
	Health      int
	Inventory   []Item
	Friendly    bool
}

// NewCharacter creates a new character with the given name and description
func NewCharacter(name, description string) Character {
	return Character{
		Name:        name,
		Description: description,
		Alive:       true,
		Health:      100,
		Friendly:    true, // Default to friendly
	}
}

// SetName sets the character's name
func (c *Character) SetName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	c.Name = name
	return nil
}

// SetDescription sets the character's description
func (c *Character) SetDescription(description string) {
	c.Description = description
}

// SetLocation sets the character's location
func (c *Character) SetLocation(location Area) {
	c.Location = location
}

// GetLocation returns the character's current location
func (c *Character) GetLocation() Area {
	return c.Location
}

// TakeDamage reduces the character's health by the specified amount
func (c *Character) TakeDamage(amount int) error {
	if amount < 0 {
		return fmt.Errorf("damage amount cannot be negative")
	}
	if !c.Alive {
		return fmt.Errorf("character is already dead")
	}

	c.Health -= amount
	if c.Health <= 0 {
		c.Health = 0
		c.Alive = false
	}
	return nil
}

// Heal increases the character's health by the specified amount
func (c *Character) Heal(amount int) error {
	if amount < 0 {
		return fmt.Errorf("heal amount cannot be negative")
	}
	if !c.Alive {
		return fmt.Errorf("cannot heal a dead character")
	}

	c.Health += amount
	if c.Health > 100 {
		c.Health = 100
	}
	return nil
}

// IsAlive returns whether the character is alive
func (c *Character) IsAlive() bool {
	return c.Alive
}

// GetHealth returns the character's current health
func (c *Character) GetHealth() int {
	return c.Health
}

// GetName returns the character's name
func (c *Character) GetName() string {
	return c.Name
}

// GetDescription returns the character's description
func (c *Character) GetDescription() string {
	return c.Description
}

// Revive brings a dead character back to life with specified health
func (c *Character) Revive(health int) error {
	if c.Alive {
		return fmt.Errorf("character is already alive")
	}
	if health <= 0 {
		return fmt.Errorf("health must be positive")
	}
	if health > 100 {
		health = 100
	}

	c.Alive = true
	c.Health = health
	return nil
}

// SetFriendly sets the character's friendly status
func (c *Character) SetFriendly(isFriendly bool) {
	c.Friendly = isFriendly
}

// IsFriendly returns whether the character is friendly
func (c *Character) IsFriendly() bool {
	return c.Friendly
}

// String returns a string representation of the character
func (c *Character) String() string {
	status := "alive"
	if !c.Alive {
		status = "dead"
	}
	disposition := "friendly"
	if !c.Friendly {
		disposition = "hostile"
	}
	return fmt.Sprintf("%s (%s) - Health: %d, Status: %s, Disposition: %s", c.Name, c.Description, c.Health, status, disposition)
}
