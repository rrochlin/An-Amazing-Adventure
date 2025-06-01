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
	Narrative      string
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
		Narrative:      cfg.game.Narrative,
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
	part := genai.Part{Text: fmt.Sprintf(`
	Player Says: %s
	respond ONLY with a plan for your next actions and how these will help build 
	and reinforce the narrative you are building for the player.`,
		params.Chat,
		gameState.Player.GetInventoryNames())}

	result, err := cfg.chat.SendMessage(req.Context(), part)
	if err != nil {
		ErrorServer("failed to get response", w, err)
		return
	}
	fmt.Printf("planning result: %v\n", result.Text())

	part = genai.Part{Text: "Execute the plan you've outlined and provide engaging narrative to the player"}

	result, err = cfg.chat.SendMessage(req.Context(), part)
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

	jsonStr := codeBlockMatches[1]

	// Parse the response as JSON
	type AIResponse struct {
		Narrative string `json:"narrative"`
		ToolCalls []struct {
			Tool      string         `json:"tool"`
			Arguments map[string]any `json:"arguments"`
		} `json:"tool_calls"`
	}

	var aiResponse AIResponse

	err = json.Unmarshal([]byte(jsonStr), &aiResponse)
	if err != nil {
		ErrorServer("failed to parse AI response", w, err)
	}

	// Execute any tool calls
	var toolResults []string
	if len(aiResponse.ToolCalls) > 0 {
		for _, toolCall := range aiResponse.ToolCalls {
			toolResult := cfg.ExecuteTool(toolCall.Tool, toolCall.Arguments)
			toolResults = append(toolResults, fmt.Sprintf("Tool %s: %s", toolCall.Tool, toolResult))
		}
	}

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
