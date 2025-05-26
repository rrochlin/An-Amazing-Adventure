package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

// WorldGenAction represents a single action to be performed during world generation
type WorldGenAction struct {
	Tool      string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
}

// WorldGenerator handles the world generation process
type WorldGenerator struct {
	game    *Game
	mu      sync.Mutex
	isReady bool
	chat    *genai.Chat
	cfg     *apiConfig
}

// NewWorldGenerator creates a new world generator
func NewWorldGenerator(game *Game) *WorldGenerator {
	return &WorldGenerator{
		game:    game,
		isReady: false,
	}
}

// SetChat sets the chat session for AI communication
func (wg *WorldGenerator) SetChat(chat *genai.Chat) {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	wg.chat = chat
}

// SetConfig sets the API configuration
func (wg *WorldGenerator) SetConfig(cfg *apiConfig) {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	wg.cfg = cfg
}

// GenerateWorld communicates with the AI to generate the game world
func (wg *WorldGenerator) GenerateWorld(ctx context.Context) error {
	if wg.chat == nil {
		return fmt.Errorf("chat session not initialized")
	}

	// Create a timeout context for the AI request
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Initial prompt to get the AI's general idea
	initialPrompt := `You are creating a text adventure game world. First, describe your general idea for the world, including:
1. The theme and setting
2. The starting room and its immediate surroundings
3. Key characters and items that will be present
4. How rooms will be connected

After describing your plan, you will receive a blank chat where you can start implementing the world using the available tools.`

	// Get the AI's initial plan
	part := genai.Part{Text: initialPrompt}
	result, err := wg.chat.SendMessage(ctx, part)
	if err != nil {
		return fmt.Errorf("failed to get initial plan: %w", err)
	}

	// Print the AI's plan
	fmt.Println("\nAI's World Generation Plan:")
	fmt.Println("=========================")
	fmt.Println(result.Text())
	fmt.Println("=========================")

	// Create the player character
	player := NewCharacter("Player", "The main character of the adventure")
	wg.game.Player = player

	// Start the iterative world building process
	for {
		// Get current world state
		worldState := wg.getWorldState()

		// Create prompt for next iteration
		iterationPrompt := fmt.Sprintf(`Current World State:
%s

Please follow this process:
1. First, create all rooms with their descriptions
2. Then, connect the rooms to each other as needed
3. Next, create all items and characters
4. Finally, add items and characters to their appropriate rooms
5. Once everything is placed, set the player's starting room
6. Call stop_generation when complete

Tool calls should be in JSON format:
{
    "tool": "tool-name",
    "arguments": {
        "argument-name": "value"
    }
}`, worldState)

		// Get AI's next action
		part = genai.Part{Text: iterationPrompt}
		result, err = wg.chat.SendMessage(ctx, part)
		if err != nil {
			return fmt.Errorf("failed to get next action: %w", err)
		}

		// Extract JSON from AI response
		text := result.Text()
		pattern := `(?is)\s*(\{.*\}|\[.*\])\s*`
		regex := regexp.MustCompile(pattern)
		matches := regex.FindStringSubmatch(text)

		if len(matches) < 2 {
			return fmt.Errorf("no valid JSON found in AI response")
		}

		// Parse actions
		var actions []WorldGenAction
		err = json.Unmarshal([]byte(matches[1]), &actions)
		if err != nil {
			// Try parsing as single action
			var singleAction WorldGenAction
			err = json.Unmarshal([]byte(matches[1]), &singleAction)
			if err != nil {
				return fmt.Errorf("failed to parse AI action: %w", err)
			}
			actions = []WorldGenAction{singleAction}
		}

		// Execute all actions
		for _, action := range actions {
			// Check if generation is complete
			if action.Tool == "stop_generation" {
				// Verify that player has a starting room
				if wg.game.Player.GetLocation().ID == "" {
					return fmt.Errorf("world generation complete but player has no starting room")
				}
				break
			}

			// Execute the action
			result := wg.cfg.ExecuteTool(action.Tool, action.Arguments)
			fmt.Printf("World gen action: %s - Result: %s\n", action.Tool, result)
		}

		// If we hit stop_generation, break the loop
		if len(actions) > 0 && actions[len(actions)-1].Tool == "stop_generation" {
			break
		}
	}

	// Print final world state
	fmt.Println("\nWorld Generation Complete!")
	fmt.Println("=========================")
	for _, area := range wg.game.GetAllAreas() {
		fmt.Printf("\nRoom: %s\n", area.ID)
		fmt.Printf("Description: %s\n", area.GetDescription())

		fmt.Println("Connections:")
		for _, conn := range area.GetConnections() {
			fmt.Printf("  - %s\n", conn.ID)
		}

		fmt.Println("Items:")
		for _, item := range area.GetItems() {
			fmt.Printf("  - %s\n", item.String())
		}

		fmt.Println("Occupants:")
		for _, occupant := range area.GetOccupants() {
			fmt.Printf("  - %s\n", occupant)
		}
	}
	fmt.Println("=========================")

	wg.mu.Lock()
	wg.isReady = true
	wg.mu.Unlock()

	return nil
}

// getWorldState returns a string representation of the current world state
func (wg *WorldGenerator) getWorldState() string {
	var sb strings.Builder

	// Get all rooms
	rooms := wg.game.GetAllAreas()
	if len(rooms) == 0 {
		return "No rooms have been created yet."
	}

	// Build room information
	for _, room := range rooms {
		sb.WriteString(fmt.Sprintf("\nRoom: %s\n", room.ID))
		sb.WriteString(fmt.Sprintf("Description: %s\n", room.GetDescription()))

		// List connections
		sb.WriteString("Connections:\n")
		connections := room.GetConnections()
		if len(connections) == 0 {
			sb.WriteString("  - None\n")
		} else {
			for _, conn := range connections {
				sb.WriteString(fmt.Sprintf("  - %s\n", conn.ID))
			}
		}

		// List items
		sb.WriteString("Items:\n")
		items := room.GetItems()
		if len(items) == 0 {
			sb.WriteString("  - None\n")
		} else {
			for _, item := range items {
				sb.WriteString(fmt.Sprintf("  - %s\n", item.String()))
			}
		}

		// List occupants
		sb.WriteString("Occupants:\n")
		occupants := room.GetOccupants()
		if len(occupants) == 0 {
			sb.WriteString("  - None\n")
		} else {
			for _, occupant := range occupants {
				sb.WriteString(fmt.Sprintf("  - %s\n", occupant))
			}
		}
	}

	return sb.String()
}

// IsReady returns whether world generation is complete
func (wg *WorldGenerator) IsReady() bool {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	return wg.isReady
}
