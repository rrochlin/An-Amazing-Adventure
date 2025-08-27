package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"google.golang.org/genai"
)

// gameStateMiddleware wraps an http.HandlerFunc and saves the game state after execution
func (cfg *apiConfig) gameStateMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Create a response writer that captures the status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Execute the handler
		next(rw, req)

		// Only save game state if the request was successful
		if rw.statusCode >= 200 && rw.statusCode < 300 {
			saveState := cfg.game.SaveGameState()
			err := cfg.PutGame(req.Context(), saveState)
			if err != nil {
				ErrorServer("failed to write to dynamoDB", w, err)
			}
		}
	}
}

// responseWriter is a custom ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (cfg *apiConfig) HandlerStartGame(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Authorization string `json:"chat"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, nil)
		return
	}
	// Try to load existing game
	if cfg.game.LoadGameState() == nil {
		// Game loaded successfully
		type retVal struct {
			Status string `json:"status"`
		}
		RetVal := retVal{
			Status: "Game loaded",
		}
		dat, err := json.Marshal(RetVal)
		if err != nil {
			ErrorServer("failed to marshal response", w, err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)

		gameState := cfg.getGameState()

		cfg.chat.SendMessage(req.Context(), genai.Part{Text: fmt.Sprintf(`You have been loaded from a save state,
		you are in the room %s,
		you have the following items: %s,
		you have the following occupants: %s,
		you have the following connections: %s,
		you previously defined your narrative as: %s,
		You will receive further instructions in the next message respond with "ok"`,
			gameState.CurrentRoom.ID,
			gameState.CurrentRoom.GetItems(),
			gameState.CurrentRoom.GetOccupants(),
			cfg.game.Narrative,
			gameState.CurrentRoom.GetConnections())})

		cfg.worldGen = NewWorldGenerator(&cfg.game)
		cfg.worldGen.mu.Lock()
		cfg.worldGen.isReady = true
		cfg.worldGen.mu.Unlock()
		return
	}

	// Create a new game if loading failed
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
		// Save game state after world generation
		if err := cfg.game.SaveGameState(); err != nil {
			fmt.Printf("Failed to save game state: %v\n", err)
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
	type retVal struct {
		Ready bool `json:"ready"`
	}
	RetVal := retVal{
		Ready: cfg.worldGen != nil && cfg.worldGen.IsReady(),
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
