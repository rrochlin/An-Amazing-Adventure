package main

import (
	"fmt"
	"strings"
)

// Tool represents a single tool in the MCP system
type Tool struct {
	Name        string
	Description string
	Arguments   map[string]string
}

// GetTools returns the available tools and their descriptions
func GetTools() []Tool {
	return []Tool{
		{
			Name:        "create_room",
			Description: "Creates a new room with a unique ID and optional description",
			Arguments: map[string]string{
				"id":          "Unique identifier for the room",
				"description": "Optional description of the room",
			},
		},
		{
			Name:        "create_item",
			Description: "Creates a new item with various properties",
			Arguments: map[string]string{
				"name":          "Name of the item",
				"description":   "Description of the item",
				"weight":        "Optional weight of the item (default: 1.0)",
				"value":         "Optional value of the item (default: 1)",
				"is_weapon":     "Optional boolean indicating if the item is a weapon (default: false)",
				"damage":        "Optional damage value if the item is a weapon (default: 0)",
				"is_consumable": "Optional boolean indicating if the item is consumable (default: false)",
				"uses":          "Optional number of uses if the item is consumable (default: 1)",
			},
		},
		{
			Name:        "create_character",
			Description: "Creates a new character with a name, description, and friendly status",
			Arguments: map[string]string{
				"name":        "Name of the character",
				"description": "Description of the character",
				"is_friendly": "true/false indicating if the character is friendly",
			},
		},
		{
			Name:        "set_item_location",
			Description: "Sets the location of an item to a room or character inventory",
			Arguments: map[string]string{
				"item_name":     "Name of the item to move",
				"location_type": "Type of location ('room' or 'inventory')",
				"location_id":   "ID of the room or name of the character (use 'player' for player inventory)",
			},
		},
		{
			Name:        "set_character_location",
			Description: "Sets the location of any character (including the player) to a specified room",
			Arguments: map[string]string{
				"character_name": "Name of the character to move (use 'player' for the player character)",
				"room_id":        "ID of the room to move the character to",
			},
		},
		{
			Name:        "connect_rooms",
			Description: "Creates a connection between two rooms",
			Arguments: map[string]string{
				"room_id_1": "ID of the first room",
				"room_id_2": "ID of the second room",
			},
		},
		{
			Name:        "get_room_info",
			Description: "Gets comprehensive information about a room, including its description, items, occupants, and connections",
			Arguments: map[string]string{
				"room_id": "ID of the room to get information for",
			},
		},
		{
			Name:        "get_item_info",
			Description: "Gets information about an item",
			Arguments: map[string]string{
				"item_name": "Name of the item to get info for",
			},
		},
		{
			Name:        "get_character_info",
			Description: "Gets information about a character",
			Arguments: map[string]string{
				"character_name": "Name of the character to get info for",
			},
		},
		{
			Name:        "list_connected_rooms",
			Description: "Lists all rooms connected to a given room",
			Arguments: map[string]string{
				"room_id": "ID of the room to get connections for",
			},
		},
		{
			Name:        "list_items_in_room",
			Description: "Lists all items in a given room",
			Arguments: map[string]string{
				"room_id": "ID of the room to get items for",
			},
		},
		{
			Name:        "list_characters_in_room",
			Description: "Lists all characters in a given room",
			Arguments: map[string]string{
				"room_id": "ID of the room to get characters for",
			},
		},
		{
			Name:        "stop_generation",
			Description: "Signals that world generation is complete",
			Arguments:   map[string]string{},
		},
	}
}

// ExecuteCreateRoom creates a new room in the game
func (cfg *apiConfig) ExecuteCreateRoom(args map[string]any) string {
	id, ok := args["id"].(string)
	if !ok {
		return "Invalid room ID"
	}

	description, _ := args["description"].(string)

	room := NewArea(id)
	if description != "" {
		// Add description to room metadata if needed
	}

	err := cfg.game.AddArea(id, room)
	if err != nil {
		return fmt.Sprintf("Failed to create room: %v", err)
	}

	return fmt.Sprintf("Successfully created room %s", id)
}

// ExecuteCreateItem creates a new item in the game
func (cfg *apiConfig) ExecuteCreateItem(args map[string]any) string {
	name, ok := args["name"].(string)
	if !ok {
		return "Invalid item name"
	}

	description, _ := args["description"].(string)

	item := NewItem(name, description)

	// Handle weight
	if weight, ok := args["weight"].(float64); ok {
		if err := item.SetWeight(weight); err != nil {
			return fmt.Sprintf("Invalid weight: %v", err)
		}
	}

	// Handle value
	if value, ok := args["value"].(int); ok {
		if err := item.SetValue(value); err != nil {
			return fmt.Sprintf("Invalid value: %v", err)
		}
	}

	// Handle weapon properties
	if isWeapon, ok := args["is_weapon"].(bool); ok {
		damage := 0
		if dmg, ok := args["damage"].(int); ok {
			damage = dmg
		}
		if err := item.SetWeapon(isWeapon, damage); err != nil {
			return fmt.Sprintf("Invalid weapon properties: %v", err)
		}
	}

	// Handle consumable properties
	if isConsumable, ok := args["is_consumable"].(bool); ok {
		uses := 1
		if u, ok := args["uses"].(int); ok {
			uses = u
		}
		if err := item.SetConsumable(isConsumable, uses); err != nil {
			return fmt.Sprintf("Invalid consumable properties: %v", err)
		}
	}

	err := cfg.game.AddItem(item)
	if err != nil {
		return fmt.Sprintf("Failed to create item: %v", err)
	}

	return fmt.Sprintf("Successfully created item %s", name)
}

// ExecuteCreateCharacter creates a new character in the game
func (cfg *apiConfig) ExecuteCreateCharacter(args map[string]any) string {
	name, ok := args["name"].(string)
	if !ok {
		return "Invalid character name"
	}

	description, _ := args["description"].(string)

	character := NewCharacter(name, description)

	// Handle friendly status
	if isFriendly, ok := args["is_friendly"].(bool); ok {
		character.SetFriendly(isFriendly)
	}

	err := cfg.game.AddNPC(character)
	if err != nil {
		return fmt.Sprintf("Failed to create character: %v", err)
	}

	return fmt.Sprintf("Successfully created character %s", name)
}

// ExecuteSetItemLocation sets the location of an item
func (cfg *apiConfig) ExecuteSetItemLocation(args map[string]any) string {
	itemName, ok := args["item_name"].(string)
	if !ok {
		return "Invalid item name"
	}

	locationType, ok := args["location_type"].(string)
	if !ok || (locationType != "room" && locationType != "inventory") {
		return "Invalid location type (must be 'room' or 'inventory')"
	}

	locationID, ok := args["location_id"].(string)
	if !ok {
		return "Invalid location ID"
	}

	item, err := cfg.game.GetItem(itemName)
	if err != nil {
		return fmt.Sprintf("Item not found: %v", err)
	}

	// Remove item from current location if it exists
	if currentLoc := item.GetLocation(); currentLoc != nil {
		switch loc := currentLoc.(type) {
		case Area:
			if err := loc.RemoveItem(itemName); err != nil {
				return fmt.Sprintf("Failed to remove item from current location: %v", err)
			}
		case Character:
			if err := loc.RemoveItem(itemName); err != nil {
				return fmt.Sprintf("Failed to remove item from current location: %v", err)
			}
		}
	}

	if locationType == "room" {
		// Move item to room
		room, err := cfg.game.GetArea(locationID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		if err := room.AddItem(item); err != nil {
			return fmt.Sprintf("Failed to add item to room: %v", err)
		}
		item.SetLocation(room)
		return fmt.Sprintf("Successfully moved item %s to room %s", itemName, locationID)
	} else {
		// Move item to inventory
		if locationID == "player" {
			if err := cfg.game.AddItemToInventory(item); err != nil {
				return fmt.Sprintf("Failed to add item to player inventory: %v", err)
			}
			item.SetLocation(&cfg.game.Player)
			return fmt.Sprintf("Successfully moved item %s to player inventory", itemName)
		} else {
			character, err := cfg.game.GetNPC(locationID)
			if err != nil {
				return fmt.Sprintf("Character not found: %v", err)
			}
			if err := character.AddItem(item); err != nil {
				return fmt.Sprintf("Failed to add item to character inventory: %v", err)
			}
			item.SetLocation(character)
			return fmt.Sprintf("Successfully moved item %s to %s's inventory", itemName, locationID)
		}
	}
}

// ExecuteSetCharacterLocation sets the location of a character
func (cfg *apiConfig) ExecuteSetCharacterLocation(args map[string]any) string {
	characterName, ok := args["character_name"].(string)
	if !ok {
		return "Invalid character name"
	}

	roomID, ok := args["room_id"].(string)
	if !ok {
		return "Invalid room ID"
	}

	room, err := cfg.game.GetArea(roomID)
	if err != nil {
		return fmt.Sprintf("Room not found: %v", err)
	}

	if characterName == "player" {
		cfg.game.Player.Location = room
		return fmt.Sprintf("Successfully moved player to room %s", roomID)
	}

	// For NPCs, we'll need to update their location in the game state
	character, err := cfg.game.GetNPC(characterName)
	if err != nil {
		return fmt.Sprintf("Character not found: %v", err)
	}

	// Remove character from current room if they're in one
	if character.Location.ID != "" {
		if err := character.Location.RemoveOccupant(characterName); err != nil {
			return fmt.Sprintf("Failed to remove character from current room: %v", err)
		}
	}

	// Add character to new room
	if err := room.AddOccupant(characterName); err != nil {
		return fmt.Sprintf("Failed to add character to new room: %v", err)
	}

	// Update character's location
	character.Location = room
	return fmt.Sprintf("Successfully moved character %s to room %s", characterName, roomID)
}

// ExecuteConnectRooms creates a connection between two rooms
func (cfg *apiConfig) ExecuteConnectRooms(args map[string]any) string {
	roomID1, ok := args["room_id_1"].(string)
	if !ok {
		return "Invalid first room ID"
	}

	roomID2, ok := args["room_id_2"].(string)
	if !ok {
		return "Invalid second room ID"
	}

	room1, err := cfg.game.GetArea(roomID1)
	if err != nil {
		return fmt.Sprintf("First room not found: %v", err)
	}

	room2, err := cfg.game.GetArea(roomID2)
	if err != nil {
		return fmt.Sprintf("Second room not found: %v", err)
	}

	err = room1.AddConnection(room2)
	if err != nil {
		return fmt.Sprintf("Failed to connect rooms: %v", err)
	}

	err = room2.AddConnection(room1)
	if err != nil {
		return fmt.Sprintf("Failed to connect rooms: %v", err)
	}

	// Update the rooms in the game map
	cfg.game.Map[roomID1] = room1
	cfg.game.Map[roomID2] = room2

	return fmt.Sprintf("Successfully connected rooms %s and %s", roomID1, roomID2)
}

// GetSystemInstructions returns the complete system instructions for the AI
func GetSystemInstructions() string {
	tools := GetTools()
	var toolsInfo strings.Builder
	toolsInfo.WriteString("Available tools:\n")
	for _, tool := range tools {
		toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		if len(tool.Arguments) > 0 {
			toolsInfo.WriteString("  Arguments:\n")
			for argName, argDesc := range tool.Arguments {
				toolsInfo.WriteString(fmt.Sprintf("    - %s: %s\n", argName, argDesc))
			}
		}
		toolsInfo.WriteString("\n")
	}

	return fmt.Sprintf(`You are an AI Game Master for a text-based adventure game. 
	Your role is to create an engaging and immersive experience for the player.

IMPORTANT: You will be asked to first plan out your actions. You can do this in whatever format
you want. Afterwards you will be asked to provide a responde.

You must structure your responses as valid JSON objects with the following format:
{
    "narrative": "Your narrative response describing what happens, what the player sees, etc.",
    "tool_calls": [
        {
            "tool": "tool_name",
            "arguments": {
                "arg1": "value1",
                "arg2": "value2"
            }
        }
    ]
}

The narrative field should contain your descriptive text about what happens in the game. 
The tool_calls field should contain an array of tools you want to use to modify the game state.

%s

Use the available tools to modify the game state as needed. 
The tool definitions above contain all the information you need about how to use each tool, including required and optional arguments.

Game state management guidelines:
- To move characters (including the player): use tools to set the character's location to the appropriate room
- To manage inventory: use tools to move items to the appropriate inventory (player or character)
- To place items in rooms: use tools to set the item's location to the desired room
- To create new content: use tools to create new rooms, items, or characters as needed
- To connect areas: use tools to establish connections between rooms
- To gather information: use tools to get details about rooms, items, or characters

Always maintain consistency in the game world and provide clear, engaging descriptions.`, toolsInfo.String())
}

// ExecuteTool executes a tool based on its name and arguments
func (cfg *apiConfig) ExecuteTool(toolName string, args map[string]any) string {
	fmt.Printf("Executing tool: %s\n", toolName)
	fmt.Printf("With arguments: %+v\n", args)

	switch toolName {
	case "create_room":
		return cfg.ExecuteCreateRoom(args)
	case "create_item":
		return cfg.ExecuteCreateItem(args)
	case "create_character":
		return cfg.ExecuteCreateCharacter(args)
	case "set_item_location":
		return cfg.ExecuteSetItemLocation(args)
	case "set_character_location":
		return cfg.ExecuteSetCharacterLocation(args)
	case "connect_rooms":
		return cfg.ExecuteConnectRooms(args)
	case "get_room_info":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}

		// Create a comprehensive room description
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Room %s:\n", room.ID))
		sb.WriteString(fmt.Sprintf("Description: %s\n", room.Description))

		// Add items
		sb.WriteString("\nItems:\n")
		for _, item := range room.GetItems() {
			sb.WriteString(fmt.Sprintf("- %s\n", item.String()))
		}

		// Add occupants
		sb.WriteString("\nOccupants:\n")
		for _, occupant := range room.GetOccupants() {
			sb.WriteString(fmt.Sprintf("- %s\n", occupant))
		}

		// Add connections
		sb.WriteString("\nConnections:\n")
		for _, conn := range room.GetConnections() {
			sb.WriteString(fmt.Sprintf("- Room %s\n", conn.ID))
		}

		// Return as JSON with only the narrative key
		return fmt.Sprintf(`{"narrative": %q}`, sb.String())
	case "get_item_info":
		itemName, ok := args["item_name"].(string)
		if !ok {
			return "Invalid item name"
		}
		item, err := cfg.game.GetItem(itemName)
		if err != nil {
			return fmt.Sprintf("Item not found: %v", err)
		}
		return fmt.Sprintf(`{"narrative": %q}`, fmt.Sprintf("Item %s: %s", item.Name, item.Description))
	case "get_character_info":
		characterName, ok := args["character_name"].(string)
		if !ok {
			return "Invalid character name"
		}
		character, err := cfg.game.GetNPC(characterName)
		if err != nil {
			return fmt.Sprintf("Character not found: %v", err)
		}
		return fmt.Sprintf(`{"narrative": %q}`, fmt.Sprintf("Character %s: %s", character.Name, character.Description))
	case "list_connected_rooms":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Rooms connected to %s:\n", room.ID))
		for _, conn := range room.GetConnections() {
			sb.WriteString(fmt.Sprintf("- %s\n", conn.ID))
		}
		return sb.String()
	case "list_items_in_room":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Items in room %s:\n", room.ID))
		for _, item := range room.GetItems() {
			sb.WriteString(fmt.Sprintf("- %s\n", item.String()))
		}
		return sb.String()
	case "list_characters_in_room":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Characters in room %s:\n", room.ID))
		for _, occupant := range room.GetOccupants() {
			sb.WriteString(fmt.Sprintf("- %s\n", occupant))
		}
		return sb.String()
	case "stop_generation":
		return "World generation complete"
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}
