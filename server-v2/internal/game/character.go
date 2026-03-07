package game

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// Character represents the player or any NPC.
type Character struct {
	ID          string    `json:"id" dynamodbav:"id"`
	Name        string    `json:"name" dynamodbav:"name"`
	Description string    `json:"description" dynamodbav:"description"`
	Backstory   string    `json:"backstory,omitempty" dynamodbav:"backstory,omitempty"`
	Age         string    `json:"age,omitempty" dynamodbav:"age,omitempty"` // optional age descriptor, e.g. "early 30s"
	LocationID  string    `json:"location_id" dynamodbav:"location_id"`     // room UUID; "" = not placed
	Alive       bool      `json:"alive" dynamodbav:"alive"`
	Health      int       `json:"health" dynamodbav:"health"` // 0-100
	Friendly    bool      `json:"friendly" dynamodbav:"friendly"`
	Inventory   []string  `json:"inventory" dynamodbav:"inventory"` // item IDs
	Equipment   Equipment `json:"equipment" dynamodbav:"equipment"`
}

// NewCharacter creates a new Character with a server-generated UUID.
func NewCharacter(name, description string) Character {
	return Character{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Alive:       true,
		Health:      100,
		Friendly:    true,
		Inventory:   []string{},
	}
}

// TakeDamage reduces health and marks dead at zero.
func (c *Character) TakeDamage(amount int) error {
	if amount < 0 {
		return fmt.Errorf("damage cannot be negative")
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

// Heal increases health up to 100.
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

// Revive brings a dead character back with the given health.
func (c *Character) Revive(health int) error {
	if c.Alive {
		return fmt.Errorf("character is already alive")
	}
	if health <= 0 || health > 100 {
		return fmt.Errorf("health must be 1-100")
	}
	c.Alive = true
	c.Health = health
	return nil
}

// AddItemID adds an item ID to inventory.
func (c *Character) AddItemID(itemID string) error {
	if slices.Contains(c.Inventory, itemID) {
		return fmt.Errorf("item already in inventory")
	}
	c.Inventory = append(c.Inventory, itemID)
	return nil
}

// RemoveItemID removes an item ID from inventory.
func (c *Character) RemoveItemID(itemID string) error {
	idx := slices.Index(c.Inventory, itemID)
	if idx < 0 {
		return fmt.Errorf("item not in inventory")
	}
	c.Inventory = slices.Delete(c.Inventory, idx, idx+1)
	return nil
}

// HasItem checks if an item ID is in inventory.
func (c *Character) HasItem(itemID string) bool {
	return slices.Contains(c.Inventory, itemID)
}

// EquipItem moves an item from inventory into the appropriate equipment slot.
// The item must already be in the character's inventory and must be equippable
// with a non-empty slot.  Returns an error otherwise.
func (c *Character) EquipItem(item Item) error {
	if item.Slot == "" || !item.Equippable {
		return fmt.Errorf("item %q is not equippable", item.Name)
	}
	if !c.HasItem(item.ID) {
		return fmt.Errorf("item %q is not in inventory", item.Name)
	}
	id := item.ID
	switch item.Slot {
	case SlotHead:
		c.Equipment.Head = &id
	case SlotChest:
		c.Equipment.Chest = &id
	case SlotLegs:
		c.Equipment.Legs = &id
	case SlotHands:
		c.Equipment.Hands = &id
	case SlotFeet:
		c.Equipment.Feet = &id
	case SlotBack:
		c.Equipment.Back = &id
	default:
		return fmt.Errorf("unknown equipment slot: %s", item.Slot)
	}
	// Keep the item in inventory (equipped ≠ removed from pack in this design)
	return nil
}

// UnequipItem clears the given equipment slot and returns the item ID that was
// in the slot, or an error if the slot is already empty.
func (c *Character) UnequipItem(slot EquipmentSlot) (string, error) {
	var ptr **string
	switch slot {
	case SlotHead:
		ptr = &c.Equipment.Head
	case SlotChest:
		ptr = &c.Equipment.Chest
	case SlotLegs:
		ptr = &c.Equipment.Legs
	case SlotHands:
		ptr = &c.Equipment.Hands
	case SlotFeet:
		ptr = &c.Equipment.Feet
	case SlotBack:
		ptr = &c.Equipment.Back
	default:
		return "", fmt.Errorf("unknown equipment slot: %s", slot)
	}
	if *ptr == nil {
		return "", fmt.Errorf("slot %s is already empty", slot)
	}
	id := **ptr
	*ptr = nil
	return id, nil
}
