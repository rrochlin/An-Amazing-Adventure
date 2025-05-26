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
			Name:        "add_item_to_room",
			Description: "Adds an existing item to a room",
			Arguments: map[string]string{
				"room_id":   "ID of the room",
				"item_name": "Name of the item to add",
			},
		},
		{
			Name:        "add_character_to_room",
			Description: "Adds an existing character to a room",
			Arguments: map[string]string{
				"room_id":        "ID of the room",
				"character_name": "Name of the character to add",
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
			Name:        "describe_room",
			Description: "Returns a detailed description of a room and its contents",
			Arguments: map[string]string{
				"room_id": "ID of the room to describe",
			},
		},
		{
			Name:        "get_room_info",
			Description: "Gets basic information about a room",
			Arguments: map[string]string{
				"room_id": "ID of the room to get info for",
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
			Name:        "remove_item_from_room",
			Description: "Removes an item from a room",
			Arguments: map[string]string{
				"room_id":   "ID of the room",
				"item_name": "Name of the item to remove",
			},
		},
		{
			Name:        "remove_character_from_room",
			Description: "Removes a character from a room",
			Arguments: map[string]string{
				"room_id":        "ID of the room",
				"character_name": "Name of the character to remove",
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
			Name:        "set_player_starting_room",
			Description: "Sets the player's starting room",
			Arguments: map[string]string{
				"room_id": "ID of the room to set as player's starting location",
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

// ExecuteAddItemToRoom adds an item to a room
func (cfg *apiConfig) ExecuteAddItemToRoom(args map[string]any) string {
	roomID, ok := args["room_id"].(string)
	if !ok {
		return "Invalid room ID"
	}

	itemName, ok := args["item_name"].(string)
	if !ok {
		return "Invalid item name"
	}

	item, err := cfg.game.GetItem(itemName)
	if err != nil {
		return fmt.Sprintf("Item not found: %v", err)
	}

	err = cfg.game.AddItemToArea(roomID, item)
	if err != nil {
		return fmt.Sprintf("Failed to add item to room: %v", err)
	}

	return fmt.Sprintf("Successfully added item %s to room %s", itemName, roomID)
}

// ExecuteAddCharacterToRoom adds a character to a room
func (cfg *apiConfig) ExecuteAddCharacterToRoom(args map[string]any) string {
	roomID, ok := args["room_id"].(string)
	if !ok {
		return "Invalid room ID"
	}

	characterName, ok := args["character_name"].(string)
	if !ok {
		return "Invalid character name"
	}

	err := cfg.game.AddNPCToArea(roomID, characterName)
	if err != nil {
		return fmt.Sprintf("Failed to add character to room: %v", err)
	}

	return fmt.Sprintf("Successfully added character %s to room %s", characterName, roomID)
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

// ExecuteDescribeRoom returns a detailed description of a room
func (cfg *apiConfig) ExecuteDescribeRoom(args map[string]any) string {
	roomID, ok := args["room_id"].(string)
	if !ok {
		return "Invalid room ID"
	}

	room, err := cfg.game.GetArea(roomID)
	if err != nil {
		return fmt.Sprintf("Room not found: %v", err)
	}

	// Create a detailed description of the room
	description := fmt.Sprintf("Room %s:\n", room.ID)

	// Add items
	description += "\nItems:\n"
	for _, item := range room.GetItems() {
		description += fmt.Sprintf("- %s: %s\n", item.Name, item.Description)
	}

	// Add occupants
	description += "\nOccupants:\n"
	for _, occupant := range room.GetOccupants() {
		description += fmt.Sprintf("- %s\n", occupant)
	}

	// Add connections
	description += "\nConnections:\n"
	for _, conn := range room.GetConnections() {
		description += fmt.Sprintf("- Room %s\n", conn.ID)
	}

	return description
}

// GetSystemInstructions returns the complete system instructions for the AI
func GetSystemInstructions() string {
	introduction := fmt.Sprintf("You are a D&D DM AI with access to these tools:"+
		"\n\n%v\nChoose whether to respond to the user with narrative or invoking one or more "+
		"tools and then responding with narrative afterwards.\n\n", GetTools())

	toolFormatInstruction := "IMPORTANT: When you need to use tools, you must ONLY respond " +
		"with the exact format below, nothing else:\n" +
		"[\n" +
		"    {\n" +
		"        \"tool\": \"tool-name\",\n" +
		"        \"arguments\": {\n" +
		"            \"argument-name\": \"value\"\n" +
		"        }\n" +
		"    },\n" +
		"    {\n" +
		"        \"tool\": \"another-tool-name\",\n" +
		"        \"arguments\": {\n" +
		"            \"argument-name\": \"value\"\n" +
		"        }\n" +
		"    }\n" +
		"]\n\n" +
		"Or for a single tool call:\n" +
		"{\n" +
		"    \"tool\": \"tool-name\",\n" +
		"    \"arguments\": {\n" +
		"        \"argument-name\": \"value\"\n" +
		"    }\n" +
		"}\n\n"

	worldExplanation := "You are a D&D Dungeon Master creating an interactive adventure. " +
		"You can create rooms, populate them with items and characters, and connect them together. " +
		"Use the available tools to build the world as the player explores it. " +
		"Create interesting characters (both friendly and unfriendly) and meaningful items " +
		"that contribute to the story. Connect rooms logically to create a coherent world. " +
		"Respond to player actions by describing what they see and experience."

	finalConstraint := "After invoking tools you will receive another chat with the results " +
		"of having invoked the tools, you can then respond with narrative. Please use only the tools " +
		"that are explicitly defined above."

	return introduction + toolFormatInstruction + worldExplanation + finalConstraint
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
	case "add_item_to_room":
		return cfg.ExecuteAddItemToRoom(args)
	case "add_character_to_room":
		return cfg.ExecuteAddCharacterToRoom(args)
	case "connect_rooms":
		return cfg.ExecuteConnectRooms(args)
	case "describe_room":
		return cfg.ExecuteDescribeRoom(args)
	case "get_room_info":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		return fmt.Sprintf("Room %s: %s", room.ID, room.Description)
	case "get_item_info":
		itemName, ok := args["item_name"].(string)
		if !ok {
			return "Invalid item name"
		}
		item, err := cfg.game.GetItem(itemName)
		if err != nil {
			return fmt.Sprintf("Item not found: %v", err)
		}
		return fmt.Sprintf("Item %s: %s", item.Name, item.Description)
	case "get_character_info":
		characterName, ok := args["character_name"].(string)
		if !ok {
			return "Invalid character name"
		}
		character, err := cfg.game.GetNPC(characterName)
		if err != nil {
			return fmt.Sprintf("Character not found: %v", err)
		}
		return fmt.Sprintf("Character %s: %s", character.Name, character.Description)
	case "remove_item_from_room":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		itemName, ok := args["item_name"].(string)
		if !ok {
			return "Invalid item name"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		if err := room.RemoveItem(itemName); err != nil {
			return fmt.Sprintf("Failed to remove item: %v", err)
		}
		return fmt.Sprintf("Successfully removed item %s from room %s", itemName, roomID)
	case "remove_character_from_room":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		characterName, ok := args["character_name"].(string)
		if !ok {
			return "Invalid character name"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		if err := room.RemoveOccupant(characterName); err != nil {
			return fmt.Sprintf("Failed to remove character: %v", err)
		}
		return fmt.Sprintf("Successfully removed character %s from room %s", characterName, roomID)
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
	case "set_player_starting_room":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := cfg.game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		cfg.game.Player.SetLocation(room)
		return fmt.Sprintf("Successfully set player starting room to %s", roomID)
	case "stop_generation":
		return "World generation complete"
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}
