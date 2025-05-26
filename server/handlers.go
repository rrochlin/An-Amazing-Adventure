package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"google.golang.org/genai"
)

func (cfg *apiConfig) HandlerStartGame(w http.ResponseWriter, req *http.Request) {
	// Create a new game
	cfg.game = NewGame()

	// Initialize world generator
	cfg.worldGen = NewWorldGenerator(&cfg.game)
	cfg.worldGen.SetChat(cfg.chat)
	cfg.worldGen.SetConfig(cfg)

	// Create a new background context for world generation
	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 5*time.Minute)

	// Start world generation in background
	go func() {
		defer cancel() // Ensure context is canceled when done
		err := cfg.worldGen.GenerateWorld(ctx)
		if err != nil {
			fmt.Printf("World generation error: %v\n", err)
		}
	}()

	type retVal struct {
		Status string `json:"status"`
	}

	RetVal := retVal{
		Status: "Game started",
	}
	dat, err := json.Marshal(RetVal)
	if err != nil {
		ErrorServer("failed to marshal response", w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func (cfg *apiConfig) HandlerMove(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		RoomID string `json:"room_id"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, nil)
		return
	}

	// Get the target room
	room, err := cfg.game.GetArea(params.RoomID)
	if err != nil {
		ErrorBadRequest("Room not found", w, err)
		return
	}

	// Get current game state
	gameState := cfg.getGameState()

	// Send move action to AI for narrative response and potential world changes
	part := genai.Part{Text: fmt.Sprintf(`Game State:
%s

Player: Moving to room %s

You can:
1. Provide a narrative response about the movement
2. Generate new areas if needed (using create_room and connect_rooms tools)
3. Add items or NPCs to the new or existing rooms
4. Modify the environment based on the player's movement

Respond with a JSON object containing:
1. A narrative response
2. Any tool calls needed to modify the world`, gameState.String(), params.RoomID)}

	result, err := cfg.chat.SendMessage(req.Context(), part)
	if err != nil {
		ErrorServer("failed to get response", w, err)
		return
	}

	// Parse the AI's response
	text := result.Text()
	pattern := `(?is)\s*(\{.*\}|\[.*\])\s*`
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)

	var narrativeResponse string
	var newAreas map[string]RoomInfo

	if len(matches) > 1 {
		// Try to parse as tool calls
		var toolCalls []struct {
			Tool      string         `json:"tool"`
			Arguments map[string]any `json:"arguments"`
		}

		err := json.Unmarshal([]byte(matches[1]), &toolCalls)
		if err != nil {
			// If array parsing fails, try single tool call
			var singleToolCall struct {
				Tool      string         `json:"tool"`
				Arguments map[string]any `json:"arguments"`
			}
			err = json.Unmarshal([]byte(matches[1]), &singleToolCall)
			if err == nil {
				toolCalls = []struct {
					Tool      string         `json:"tool"`
					Arguments map[string]any `json:"arguments"`
				}{singleToolCall}
			}
		}

		if len(toolCalls) > 0 {
			// Execute all tool calls
			var toolResults []string
			for _, toolCall := range toolCalls {
				toolResult := cfg.ExecuteTool(toolCall.Tool, toolCall.Arguments)
				toolResults = append(toolResults, fmt.Sprintf("Tool %s: %s", toolCall.Tool, toolResult))
			}

			// Get updated game state
			updatedState := cfg.getGameState()

			// Send tool results back to AI for final narrative
			part = genai.Part{Text: fmt.Sprintf(`Tool Results:
%s

Updated Game State:
%s

Please provide a narrative response about the movement and any changes to the world.`, strings.Join(toolResults, "\n"), updatedState.String())}
			result, err = cfg.chat.SendMessage(req.Context(), part)
			if err != nil {
				ErrorServer("failed to process tool results", w, err)
				return
			}
			narrativeResponse = result.Text()

			// Collect any new areas that were created
			newAreas = make(map[string]RoomInfo)
			for _, area := range cfg.game.GetAllAreas() {
				if !gameState.CurrentRoom.IsConnected(*area) {
					newAreas[area.ID] = RoomInfo{
						ID:          area.ID,
						Description: area.GetDescription(),
						Connections: area.GetConnectionIDs(),
						Items:       area.GetItemNames(),
						Occupants:   area.GetOccupantNames(),
					}
				}
			}
		} else {
			narrativeResponse = text
		}
	} else {
		narrativeResponse = text
	}

	// Update player location
	cfg.game.Player.SetLocation(room)

	type retVal struct {
		Status   string              `json:"status"`
		Response string              `json:"Response"`
		NewAreas map[string]RoomInfo `json:"NewAreas,omitempty"`
	}
	RetVal := retVal{
		Status:   fmt.Sprintf("Moved to room %s", params.RoomID),
		Response: narrativeResponse,
	}
	if len(newAreas) > 0 {
		RetVal.NewAreas = newAreas
	}

	dat, err := json.Marshal(RetVal)
	if err != nil {
		ErrorServer("failed to marshal response", w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func (cfg *apiConfig) HandlerDescribe(w http.ResponseWriter, req *http.Request) {
	// Get current room
	room := cfg.game.Player.GetLocation()

	// Create room description
	description := fmt.Sprintf("Room %s:\n", room.ID)

	// Add items
	description += "\nItems:\n"
	for _, item := range room.GetItems() {
		description += fmt.Sprintf("- %s\n", item.String())
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

	// Get all rooms and their connections for the map
	rooms := make(map[string]RoomInfo)
	for _, area := range cfg.game.GetAllAreas() {
		rooms[area.ID] = RoomInfo{
			ID:          area.ID,
			Description: area.GetDescription(),
			Connections: area.GetConnectionIDs(),
			Items:       area.GetItemNames(),
			Occupants:   area.GetOccupantNames(),
		}
	}

	type retVal struct {
		Description string              `json:"description"`
		CurrentRoom string              `json:"current_room"`
		Rooms       map[string]RoomInfo `json:"rooms"`
	}
	RetVal := retVal{
		Description: description,
		CurrentRoom: room.ID,
		Rooms:       rooms,
	}
	dat, err := json.Marshal(RetVal)
	if err != nil {
		ErrorServer("failed to marshal response", w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

// RoomInfo represents the information needed to display a room in the client
type RoomInfo struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Connections []string `json:"connections"`
	Items       []string `json:"items"`
	Occupants   []string `json:"occupants"`
}

// HandlerWorldReady checks if the world is ready
func (cfg *apiConfig) HandlerWorldReady(w http.ResponseWriter, req *http.Request) {
	if cfg.worldGen == nil {
		ErrorBadRequest("world generator not initialized", w, nil)
		return
	}

	type retVal struct {
		Ready bool `json:"ready"`
	}
	RetVal := retVal{
		Ready: cfg.worldGen.IsReady(),
	}
	dat, err := json.Marshal(RetVal)
	if err != nil {
		ErrorServer("failed to marshal response", w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
