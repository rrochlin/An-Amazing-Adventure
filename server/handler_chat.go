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

	// Get NPCs in current room
	var visibleNPCs []Character
	for _, occupant := range currentRoom.GetOccupants() {
		if npc, err := cfg.game.GetNPC(occupant); err == nil {
			visibleNPCs = append(visibleNPCs, npc)
		}
	}

	return GameState{
		CurrentRoom:    currentRoom,
		Player:         player,
		VisibleItems:   currentRoom.GetItems(),
		VisibleNPCs:    visibleNPCs,
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

	sb.WriteString("\nInventory:\n")
	for _, item := range state.Player.Inventory {
		sb.WriteString(fmt.Sprintf("- %s\n", item.String()))
	}

	sb.WriteString("\nConnected rooms:\n")
	for _, room := range state.ConnectedRooms {
		sb.WriteString(fmt.Sprintf("- %s\n", room.ID))
	}

	return sb.String()
}

// HandlerChat handles chat messages from the player
func (cfg *apiConfig) HandlerChat(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Chat string `json:"chat"`
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

	// Send message to AI for processing
	part := genai.Part{Text: fmt.Sprintf(`Game State:
Player: %s

You can either:
- Provide a narrative response
- or - 
Generate new areas if needed (using create_room and connect_rooms tools)
Add items or NPCs to the new or existing rooms
Modify the environment based on the player's actions


Respond with a JSON object containing:
1. A narrative response or an empty string if you're calling tools
2. Any tool calls needed to modify the world or an empty array if you're providing a narrative response`, params.Chat)}

	result, err := cfg.chat.SendMessage(req.Context(), part)
	if err != nil {
		ErrorServer("failed to get response", w, err)
		return
	}

	// Parse the AI's response
	text := result.Text()
	fmt.Printf("AI Response: %s\n", text) // Debug log

	// Find JSON between triple backticks
	codeBlockPattern := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```")
	codeBlockMatches := codeBlockPattern.FindStringSubmatch(text)

	var jsonStr string
	if len(codeBlockMatches) > 1 {
		jsonStr = strings.TrimSpace(codeBlockMatches[1])
	} else {
		jsonStr = text
	}

	// Parse the response as JSON
	type AIResponse struct {
		Narrative string `json:"narrative"`
		ToolCalls []struct {
			Tool      string         `json:"tool"`
			Arguments map[string]any `json:"arguments"`
		} `json:"tool_calls"`
	}

	var aiResponse AIResponse
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = json.Unmarshal([]byte(jsonStr), &aiResponse)
		if err == nil {
			break
		}

		if attempt == maxRetries {
			ErrorServer("failed to parse AI response after multiple attempts", w, err)
			return
		}

		// Ask AI to fix the JSON format
		part = genai.Part{Text: fmt.Sprintf(`Your response was not valid JSON. Error: %v`, err)}

		result, err = cfg.chat.SendMessage(req.Context(), part)
		if err != nil {
			ErrorServer("failed to get retry response", w, err)
			return
		}

		text = result.Text()
		fmt.Printf("AI Retry Response (attempt %d): %s\n", attempt, text)

		// Try to find JSON in the new response
		codeBlockMatches = codeBlockPattern.FindStringSubmatch(text)
		if len(codeBlockMatches) > 1 {
			jsonStr = strings.TrimSpace(codeBlockMatches[1])
		} else {
			jsonStr = text
		}
	}

	// Execute any tool calls
	var toolResults []string
	if len(aiResponse.ToolCalls) > 0 {
		for _, toolCall := range aiResponse.ToolCalls {
			toolResult := cfg.ExecuteTool(toolCall.Tool, toolCall.Arguments)
			toolResults = append(toolResults, fmt.Sprintf("Tool %s: %s", toolCall.Tool, toolResult))
		}

		// Get updated game state
		updatedState := cfg.getGameState()

		// Send tool results back to AI for final narrative
		part = genai.Part{Text: fmt.Sprintf(`I executed the following tool calls:
%s

Updated Game State:
%s

Please provide a narrative response about what happened and what the player sees now.`, strings.Join(toolResults, "\n"), updatedState.String())}
		fmt.Printf("Sending to AI: %s\n", part.Text) // Debug log
		result, err = cfg.chat.SendMessage(req.Context(), part)
		if err != nil {
			ErrorServer("failed to process tool results", w, err)
			return
		}
		text = result.Text()
		fmt.Printf("AI Response: %s\n", text) // Debug log

		// Parse the final narrative response
		parsedText := codeBlockPattern.FindStringSubmatch(text)
		if len(parsedText) < 1 {
			ErrorServer("failed to parse AI response", w, err)
			return
		}
		err = json.Unmarshal([]byte(parsedText[1]), &aiResponse)
		if err != nil {
			fmt.Printf("err hit parsing the json response: %v\n", err)
			ErrorServer("failed to parse AI response", w, err)
			return
		}
	}

	// At this point, aiResponse.Narrative contains the final narrative
	narrativeResponse := aiResponse.Narrative

	// Collect any new areas that were created
	newAreas := make(map[string]RoomInfo)
	for _, area := range cfg.game.GetAllAreas() {
		if !gameState.CurrentRoom.IsConnected(area) {
			newAreas[area.ID] = RoomInfo{
				ID:          area.ID,
				Description: area.GetDescription(),
				Connections: area.GetConnectionIDs(),
				Items:       area.GetItemNames(),
				Occupants:   area.GetOccupantNames(),
			}
		}
	}

	type retVal struct {
		Response  string              `json:"Response"`
		NewAreas  map[string]RoomInfo `json:"NewAreas,omitempty"`
		GameState struct {
			CurrentRoom string              `json:"current_room"`
			Inventory   []string            `json:"inventory"`
			Rooms       map[string]RoomInfo `json:"rooms"`
		} `json:"game_state"`
	}
	RetVal := retVal{
		Response: narrativeResponse,
	}
	if len(newAreas) > 0 {
		RetVal.NewAreas = newAreas
	}

	// Add game state updates
	RetVal.GameState.CurrentRoom = cfg.game.Player.GetLocation().ID
	RetVal.GameState.Inventory = cfg.game.Player.GetInventoryNames()
	RetVal.GameState.Rooms = make(map[string]RoomInfo)
	for _, area := range cfg.game.GetAllAreas() {
		RetVal.GameState.Rooms[area.ID] = RoomInfo{
			ID:          area.ID,
			Description: area.GetDescription(),
			Connections: area.GetConnectionIDs(),
			Items:       area.GetItemNames(),
			Occupants:   area.GetOccupantNames(),
		}
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
