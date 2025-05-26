// Copyright 2025 Google LLC
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-2.0-flash", "gemini-2.0-flash")

// GameState represents the current state of the game for the AI
type GameState struct {
	CurrentRoom    Area
	Player         Character
	VisibleItems   []Item
	VisibleNPCs    []Character
	ConnectedRooms []Area
}

// getGameState returns the current state of the game for the AI
func (cfg *apiConfig) getGameState() GameState {
	player := cfg.game.Player
	currentRoom := player.GetLocation()

	return GameState{
		CurrentRoom:    currentRoom,
		Player:         player,
		VisibleItems:   currentRoom.GetItems(),
		VisibleNPCs:    make([]Character, 0), // TODO: Get NPCs in current room
		ConnectedRooms: currentRoom.GetConnections(),
	}
}

// formatGameState formats the game state into a string for the AI
func (state GameState) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Current Location: %s\n", state.CurrentRoom.ID))
	sb.WriteString(fmt.Sprintf("Description: %s\n", state.CurrentRoom.Description))

	sb.WriteString("\nItems in room:\n")
	for _, item := range state.VisibleItems {
		sb.WriteString(fmt.Sprintf("- %s\n", item.String()))
	}

	sb.WriteString("\nConnected rooms:\n")
	for _, room := range state.ConnectedRooms {
		sb.WriteString(fmt.Sprintf("- %s\n", room.ID))
	}

	return sb.String()
}

func (cfg *apiConfig) HandlerChat(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Message string `json:"message"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, nil)
		return
	}

	// Get current game state
	gameState := cfg.getGameState()

	// Send message to AI
	part := genai.Part{Text: fmt.Sprintf(`Game State:
%s

Player: %s

You can:
1. Provide a narrative response
2. Generate new areas if needed (using create_room and connect_rooms tools)
3. Add items or NPCs to the current or new rooms
4. Modify the environment based on the player's actions

Respond with a JSON object containing:
1. A narrative response
2. Any tool calls needed to modify the world`, gameState.String(), params.Message)}

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

Please provide a narrative response about the changes to the world.`, strings.Join(toolResults, "\n"), updatedState.String())}
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

	// Add message to chat history
	cfg.chatHistory = append(cfg.chatHistory, ChatMessage{
		Type:    "player",
		Content: params.Message,
	})
	cfg.chatHistory = append(cfg.chatHistory, ChatMessage{
		Type:    "assistant",
		Content: narrativeResponse,
	})

	type retVal struct {
		Status   string              `json:"status"`
		Response string              `json:"Response"`
		NewAreas map[string]RoomInfo `json:"NewAreas,omitempty"`
	}
	RetVal := retVal{
		Status:   "Message processed",
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
