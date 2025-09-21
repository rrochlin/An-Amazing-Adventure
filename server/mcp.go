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
				"name":        "Name of the item",
				"description": "Description of the item",
				"weight":      "Optional weight of the item (default: 1.0)",
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
func (game *Game) ExecuteCreateRoom(args map[string]any) string {
	id, ok := args["id"].(string)
	if !ok {
		return "Invalid room ID"
	}

	description, _ := args["description"].(string)

	room := NewArea(id)
	if description != "" {
		// Add description to room metadata if needed
	}

	err := game.AddArea(id, room)
	if err != nil {
		return fmt.Sprintf("Failed to create room: %v", err)
	}

	return fmt.Sprintf("Successfully created room %s", id)
}

// ExecuteCreateItem creates a new item in the game
func (game *Game) ExecuteCreateItem(args map[string]any) string {
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

	err := game.AddItem(item)
	if err != nil {
		return fmt.Sprintf("Failed to create item: %v", err)
	}

	return fmt.Sprintf("Successfully created item %s", name)
}

// ExecuteCreateCharacter creates a new character in the game
func (game *Game) ExecuteCreateCharacter(args map[string]any) string {
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

	err := game.AddNPC(character)
	if err != nil {
		return fmt.Sprintf("Failed to create character: %v", err)
	}

	return fmt.Sprintf("Successfully created character %s", name)
}

// ExecuteSetItemLocation sets the location of an item
func (game *Game) ExecuteSetItemLocation(args map[string]any) string {
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

	item, err := game.GetItem(itemName)
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
		room, err := game.GetArea(locationID)
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
			if err := game.AddItemToInventory(item); err != nil {
				return fmt.Sprintf("Failed to add item to player inventory: %v", err)
			}
			item.SetLocation(&game.Player)
			return fmt.Sprintf("Successfully moved item %s to player inventory", itemName)
		} else {
			character, err := game.GetNPC(locationID)
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
func (game *Game) ExecuteSetCharacterLocation(args map[string]any) string {
	characterName, ok := args["character_name"].(string)
	if !ok {
		return "Invalid character name"
	}

	roomID, ok := args["room_id"].(string)
	if !ok {
		return "Invalid room ID"
	}

	room, err := game.GetArea(roomID)
	if err != nil {
		return fmt.Sprintf("Room not found: %v", err)
	}

	if characterName == "player" {
		game.Player.Location = room
		return fmt.Sprintf("Successfully moved player to room %s", roomID)
	}

	// For NPCs, we'll need to update their location in the game state
	character, err := game.GetNPC(characterName)
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
func (game *Game) ExecuteConnectRooms(args map[string]any) string {
	roomID1, ok := args["room_id_1"].(string)
	if !ok {
		return "Invalid first room ID"
	}

	roomID2, ok := args["room_id_2"].(string)
	if !ok {
		return "Invalid second room ID"
	}

	room1, err := game.GetArea(roomID1)
	if err != nil {
		return fmt.Sprintf("First room not found: %v", err)
	}

	room2, err := game.GetArea(roomID2)
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
	game.Map[roomID1] = room1
	game.Map[roomID2] = room2

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
	Your role is to create an engaging and immersive experience for the player. Here is some general advice on how to be a good DM. Remember that you don't actually have dice rolling yet so
	you can't ask the player to do that. Instead of rolling the dice make them lead a convincing narrative to either pass or fail the task if it's important.
	IMPORTANT: do not directly tell the player what they can/cannot do with options. They need to read the narrative and determing themselves what to do. they must ask clarifying questions etc.
	Don't be afraid to rebuke the player if they phrase something that you don't want them to do as if they've already done it. You can chastise them and tell them they need to attempt to preform
	the action or even warn them that trying to do that is almost certainly going to result in their demise or failure.
Number One absolutely has to be the most important rule as a GM in any game: Say Yes or Roll the Dice. Whenever your player wants to do something, ask yourself this question: Is there anything at stake here? If there's nothing at stake, don't bother pulling out the dice. Just say "yes, you do that, here's what you find." Sometimes new GMs get into the habit of always trying to roll the dice for things. You don't always have to. Sometimes, if it doesn't really matter, just say "Yes."

Think of yourself as a movie Director. If things start to drag, move the scene along, or, even better, cut to the next scene. It's your job, as the director, to make sure that the pacing is right in the "movie." That means don't let things drag for too long but it also means making sure that you don't have tons of action all the time. You need dramatic moments and quiet reflective moments together, in the right amount. Thinking of myself as a director and doing my best to make the "scene" flow well has given me a lot of insight into when I should intervene and slow things down or speed them up.

Let It Ride. This is a rule from one of my favorite Indie RPGs called "The Burning Wheel" (which has a ton of great advice for GMs in it). "Let It Ride" is both a way to keep the game from staling and a way to get more creativity and drama from your players. The rule goes like this: The results of any ability check stand until the circumstances change. If you fail to climb a wall, you don't get to keep rolling until you succeed. That completely defeats the purpose of rolling in the first place. If you have all the time in the world and there are no complications for failure, the GM should just "Say Yes (see point #1). If the GM is making you roll, it means you should be able to fail. If you can't climb the wall the first time, there's no way you're going to climb it on the second attempt unless something changes (maybe you go and get some climbing gear, or you have a friend help you up over it). Another important note: this applies to successes as well. The GM can't demand a stealth check every 20 feet just to make sure you eventually get captured. If your character is trying to sneak into a camp and get to the prisoner's tent, that's one roll. If you pass the check, then you make it all the way, no need to roll again.

Failure Is Complication This is super important as a GM. If someone is trying to pick a lock and they fail, you can't let them just try again (see #3 above) because that's boring and it goes against #1 (you should have just said yes instead of rolling). Before every single roll you need to have a complication in mind for failure. Ideally it should be more than "yep, you can't pick the lock." It should instead be something that adds drama to the moment. Failure means their intent doesn't come to pass, but the actual details are up to you. If the character is trying to pick a lock, the complication for failure could be "you are interrupted by one of the guards patrolling the camp, he spots you!" Always try to have in mind some complication beyond "it didn't work."

Intent and Task - Try to train your players to give you intent and task whenever they want to do something. "I poison his drink!" isn't very useful to the GM, nor is it really good at telling the story. "I want to make him sick so he can't make it to the duel tomorrow, so I take the poison the alchemist gave me and I put exactly 1 drop into his soup while he's distracted." Now that is something a GM can work with! This ties in well to #4 above. As a GM, you may require the player to roll a Dex(Slight of Hand) to perform this maneuver. If the player surpasses the DC you set, then he gets his intent exactly as he states it. If he fails, you have a number of options as a GM. Maybe it means he's caught in the act. Or maybe it means that he accidentally dumped too much poison in but nobody saw him. Don't let your players get away with something boring like "I poison him." Make sure they always give you intent and task!

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
func (game *Game) ExecuteTool(toolName string, args map[string]any) string {
	fmt.Printf("Executing tool: %s\n", toolName)
	fmt.Printf("With arguments: %+v\n", args)

	switch toolName {
	case "create_room":
		return game.ExecuteCreateRoom(args)
	case "create_item":
		return game.ExecuteCreateItem(args)
	case "create_character":
		return game.ExecuteCreateCharacter(args)
	case "set_item_location":
		return game.ExecuteSetItemLocation(args)
	case "set_character_location":
		return game.ExecuteSetCharacterLocation(args)
	case "connect_rooms":
		return game.ExecuteConnectRooms(args)
	case "get_room_info":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := game.GetArea(roomID)
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
			sb.WriteString(fmt.Sprintf("- Room %s\n", conn))
		}

		// Return as JSON with only the narrative key
		return fmt.Sprintf(`{"narrative": %q}`, sb.String())
	case "get_item_info":
		itemName, ok := args["item_name"].(string)
		if !ok {
			return "Invalid item name"
		}
		item, err := game.GetItem(itemName)
		if err != nil {
			return fmt.Sprintf("Item not found: %v", err)
		}
		return fmt.Sprintf(`{"narrative": %q}`, fmt.Sprintf("Item %s: %s", item.Name, item.Description))
	case "get_character_info":
		characterName, ok := args["character_name"].(string)
		if !ok {
			return "Invalid character name"
		}
		character, err := game.GetNPC(characterName)
		if err != nil {
			return fmt.Sprintf("Character not found: %v", err)
		}
		return fmt.Sprintf(`{"narrative": %q}`, fmt.Sprintf("Character %s: %s", character.Name, character.Description))
	case "list_connected_rooms":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := game.GetArea(roomID)
		if err != nil {
			return fmt.Sprintf("Room not found: %v", err)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Rooms connected to %s:\n", room.ID))
		for _, conn := range room.GetConnections() {
			sb.WriteString(fmt.Sprintf("- %s\n", conn))
		}
		return sb.String()
	case "list_items_in_room":
		roomID, ok := args["room_id"].(string)
		if !ok {
			return "Invalid room ID"
		}
		room, err := game.GetArea(roomID)
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
		room, err := game.GetArea(roomID)
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
