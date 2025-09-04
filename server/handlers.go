package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rrochlin/an-amazing-adventure/internal/auth"
)

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
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorBadRequest("unable to parse auth header", w, err)
		return
	}

	userUUID, err := auth.ValidateJWT(token, cfg.api.secret)
	if err != nil {
		ErrorBadRequest("invalid token provided by client", w, err)
		return
	}

	sessionUUID, err := uuid.Parse(req.PathValue("uuid"))
	if err != nil {
		ErrorBadRequest("valid uuid not provided in path params", w, err)
		return
	}

	game, err := cfg.GetGame(req.Context(), sessionUUID)
	if err != nil && !strings.HasPrefix(err.Error(), "no game found") {
		ErrorServer("Failed to fetch game from server", w, err)
		return
	}

	type retVal struct {
		Ready bool `json:"ready"`
	}

	// there was a game found
	if err == nil {
		// Game loaded successfully
		RetVal := retVal{
			Ready: true,
		}
		dat, err := json.Marshal(RetVal)
		if err != nil {
			ErrorServer("failed to marshal response", w, err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
		return
	}

	// Create a new game if loading failed
	game = NewGame(sessionUUID, userUUID)

	// Initialize world generator
	worldGen := NewWorldGenerator(&game)
	chat, err := cfg.CreateChat(
		req.Context(),
		sessionUUID,
		nil,
	)
	if err != nil {
		ErrorServer("Failed to create chat session for game creation", w, err)
		return
	}

	worldGen.SetChat(chat)

	// Create a new background context for world generation
	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 5*time.Minute)

	// Start world generation in background
	go func() {
		defer cancel() // Ensure context is canceled when done
		err := worldGen.GenerateWorld(ctx)
		if err != nil {
			fmt.Printf("World generation error: %v\n", err)
		}
		// Save game state after world generation
		saveState := game.SaveGameState()
		if err = cfg.PutGame(req.Context(), saveState); err != nil {
			fmt.Printf("Failed to save game state: %v\n", err)
		}

	}()

	RetVal := retVal{
		Ready: false,
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
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorBadRequest("unable to parse auth header", w, err)
		return
	}

	_, err = auth.ValidateJWT(token, cfg.api.secret)
	if err != nil {
		ErrorBadRequest("invalid token provided by client", w, err)
		return
	}

	sessionUUID, err := uuid.Parse(req.PathValue("uuid"))
	if err != nil {
		ErrorBadRequest("valid uuid not provided in path params", w, err)
		return
	}

	game, err := cfg.GetGame(req.Context(), sessionUUID)
	if err != nil && !strings.HasPrefix(err.Error(), "no game found") {
		ErrorServer("Failed to fetch game from server", w, err)
		return
	}

	room := game.Player.GetLocation()

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
	for _, area := range game.GetAllAreas() {
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
		State       GameState           `json:"game_state"`
	}
	RetVal := retVal{
		Description: description,
		CurrentRoom: room.ID,
		Rooms:       rooms,
		State:       game.getGameState(),
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
	// Get current room
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorBadRequest("unable to parse auth header", w, err)
		return
	}

	_, err = auth.ValidateJWT(token, cfg.api.secret)
	if err != nil {
		ErrorBadRequest("invalid token provided by client", w, err)
		return
	}
	sessionUUID, err := uuid.Parse(req.PathValue("uuid"))
	if err != nil {
		ErrorBadRequest("valid uuid not provided in path params", w, err)
		return
	}

	_, err = cfg.GetGame(req.Context(), sessionUUID)
	if err != nil && !strings.HasPrefix(err.Error(), "no game found") {
		ErrorServer("Failed to fetch game from server", w, err)
		return
	}

	if err == nil {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(204)
	}

	w.Header().Add("Content-Type", "application/json")
}
