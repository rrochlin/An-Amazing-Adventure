package game

import "github.com/google/uuid"

// EquipmentSlot identifies a character equipment position.
type EquipmentSlot string

const (
	SlotHead  EquipmentSlot = "head"
	SlotChest EquipmentSlot = "chest"
	SlotLegs  EquipmentSlot = "legs"
	SlotHands EquipmentSlot = "hands"
	SlotFeet  EquipmentSlot = "feet"
	SlotBack  EquipmentSlot = "back"
)

// Item is a game object. Items do NOT track their own location.
// The owning Room (Area.Items) or Character (Character.Inventory) is the
// single source of truth for where an item is.
type Item struct {
	ID          string        `json:"id" dynamodbav:"id"`
	Name        string        `json:"name" dynamodbav:"name"`
	Description string        `json:"description" dynamodbav:"description"`
	Weight      float64       `json:"weight" dynamodbav:"weight"`
	Equippable  bool          `json:"equippable" dynamodbav:"equippable"`
	Slot        EquipmentSlot `json:"slot,omitempty" dynamodbav:"slot,omitempty"`
}

// NewItem creates a new Item with a server-generated UUID.
func NewItem(name, description string) Item {
	return Item{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Weight:      1.0,
	}
}

// Equipment holds what a character has equipped.
// All slots are optional — nil means nothing equipped there.
// Stubbed for future UI implementation.
type Equipment struct {
	Head  *string `json:"head,omitempty" dynamodbav:"head,omitempty"`   // item ID
	Chest *string `json:"chest,omitempty" dynamodbav:"chest,omitempty"` // item ID
	Legs  *string `json:"legs,omitempty" dynamodbav:"legs,omitempty"`
	Hands *string `json:"hands,omitempty" dynamodbav:"hands,omitempty"`
	Feet  *string `json:"feet,omitempty" dynamodbav:"feet,omitempty"`
	Back  *string `json:"back,omitempty" dynamodbav:"back,omitempty"`
}
