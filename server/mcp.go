package main

import (
	"fmt"
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
			Name:        "move_player",
			Description: "Attempts to move the player character to a location provided in the coordinates. On success will return new information about what the player sees.",
			Arguments: map[string]string{
				"position": "(x, y)",
			},
		},
		{
			Name:        "describe_room",
			Description: "returns a list of coordinates and a list of values. The coordinates represent the position of what the player can see. The values can be 0, 1, or 2. 0:room, 1:hall, 2:door",
			Arguments:   nil,
		},
	}
}

// ExecuteMovePlayer attempts to move the player and returns a success/failure message
func (cfg *apiConfig) ExecuteMovePlayer(position string) string {
	// Parse the position string into x,y coordinates
	var x, y int
	_, err := fmt.Sscanf(position, "(%d,%d)", &x, &y)
	if err != nil {
		return "Invalid position format. Please use (x,y) format."
	}

	// Attempt to move the player
	pos := Position{X: x, Y: y}
	err = cfg.game.Move(pos)

	if err != nil {
		return fmt.Sprintf("Failed to move to position (%d,%d): %v", x, y, err)
	}

	newRoom, err := cfg.game.describeRoom()
	if err != nil {
		return fmt.Sprintf("Failed to get new position data for (%d,%d): %v", x, y, err)
	}
	return fmt.Sprintf(
		"Successfully moved to position (%d,%d)\nNow seeing %v",
		cfg.game.Player.Pos.X,
		cfg.game.Player.Pos.Y,
		newRoom)
}

// ExecuteDescribeRoom returns a description of what the player can see
func (cfg *apiConfig) ExecuteDescribeRoom() string {
	// Get the maze description
	newInfo, err := cfg.game.Describe()
	if err != nil {
		return fmt.Sprintf("Error getting room description: %v", err)
	}

	// Format the description
	description := fmt.Sprintf("Current position: (%d,%d)\nVisible areas:\n",
		cfg.game.Player.Pos.X, cfg.game.Player.Pos.Y)

	for pos, val := range newInfo {
		var typeStr string
		switch val {
		case 0:
			typeStr = "room"
		case 1:
			typeStr = "hall"
		case 2:
			typeStr = "door"
		default:
			typeStr = "unknown"
		}
		description += fmt.Sprintf("- Position (%d,%d): %s\n", pos.X, pos.Y, typeStr)
	}
	fmt.Printf("description: %v\n", description)

	return description
}

// GetSystemInstructions returns the complete system instructions for the AI
func GetSystemInstructions() string {
	introduction := fmt.Sprintf("You are a D&D DM AI with access to these tools:"+
		"\n\n%v\nChoose wether to respond to the user with narrative or invoking a "+
		"tool and then responding with narrative aferwards.\n\n", GetTools())
	toolFormatInstruction := "IMPORTANT: When you need to use a tool, you must ONLY respond " +
		"with the exact format below, nothing else:\n" +
		"{\n" +
		"    \"tool\": \"tool-name\",\n" +
		"    \"arguments\": {\n" +
		"        \"argument-name\": \"value\"\n" +
		"    }\n" +
		"}\n\n"
	worldExplanation := "the game map is a square grid, the player starts in the northwest corner 0,0. " +
		"There are no negative coordinates, to move south inc y, for east inc x. In order to open a door " +
		"invoke the move tool with the door coordinates. The player will be put into the next room. " +
		"At the start of the game you're going to want to call the describe room tool so you know what the player " +
		"can see. "

	finalConstraint := "After Invoking a tool you will recieve another chat with the result " +
		"of having invoked the tool, you can then respond with narrative. Please use only the tools " +
		"that are explicitly defined above."
	debugContstraint := "IMPORTANT: You are currently in debug mode, wheneve providing a narrative also " +
		"provide information on the context you had for answering the question and whatever else you " +
		"find relevant, clearly labeled as DEBUG information. Also if the question has --V at the end " +
		"enter verbose debugging mode and provide as much detail as you can"

	return introduction + toolFormatInstruction + worldExplanation + finalConstraint + debugContstraint
}

// ExecuteTool executes a tool based on its name and arguments
func (cfg *apiConfig) ExecuteTool(toolName string, args map[string]any) string {
	fmt.Printf("Executing tool: %s\n", toolName)
	fmt.Printf("With arguments: %+v\n", args)
	switch toolName {
	case "move_player":
		if pos, ok := args["position"].(string); ok {
			return cfg.ExecuteMovePlayer(pos)
		}
		return "Invalid arguments for move_player tool"
	case "describe_room":
		return cfg.ExecuteDescribeRoom()
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}
