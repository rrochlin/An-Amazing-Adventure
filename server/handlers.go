package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rrochlin/an-amazing-adventure/internal/auth"
	"google.golang.org/genai"
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
		ErrorUnauthorized("invalid token provided by client", w, err)
		return
	}

	type parameters struct {
		PlayerName string `json:"playerName"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, nil)
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
			Ready: game.Ready,
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
	worldGen := NewWorldGenerator(&game, cfg.gemini, cfg.s3Client)
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
		err := worldGen.GenerateWorld(ctx, params.PlayerName)
		if err != nil {
			fmt.Printf("World generation error: %v\n", err)
		}
		game.Ready = true
		asyncChat, err := cfg.CreateChat(
			ctx,
			sessionUUID,
			game.Narrative,
		)
		if err != nil {
			fmt.Printf("Failed to make secondary chat: %v\n", err)
		}

		introduction := genai.Part{Text: `Please provide an introductory narrative to the player introducing them to the world and the adventure.

You must structure your response as a valid JSON object with the following format:
{
    "narrative": "Your narrative response describing what happens, what the player sees, etc.",
    "tool_calls": []
}

The narrative field should contain your descriptive text about the world and adventure.
The tool_calls field should be an empty array for this introduction.`}
		response, err := asyncChat.SendMessage(ctx, introduction)
		if err != nil {
			fmt.Printf("Failed to get chat intro: %v\n", err)
		}

		game.Narrative = asyncChat.History(false)

		// Parse the JSON response to extract just the narrative
		text := response.Text()
		codeBlockPattern := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```")
		codeBlockMatches := codeBlockPattern.FindStringSubmatch(text)

		var narrative string
		if len(codeBlockMatches) >= 2 {
			jsonStr := codeBlockMatches[1]
			type IntroResponse struct {
				Narrative string `json:"narrative"`
			}
			var introResponse IntroResponse
			if err := json.Unmarshal([]byte(jsonStr), &introResponse); err == nil {
				narrative = introResponse.Narrative
			} else {
				fmt.Printf("Failed to parse intro JSON: %v\n", err)
				narrative = text
			}
		} else {
			fmt.Printf("No JSON code block found in intro response\n")
			narrative = text
		}

		game.ChatHistory = append(game.ChatHistory, ChatMessage{Type: "narrative", Content: narrative})
		// Save game state after world generation
		saveState := game.SaveGameState()
		if err = cfg.PutGame(ctx, saveState); err != nil {
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

func (cfg *apiConfig) HandlerListGames(w http.ResponseWriter, req *http.Request) {
	type GameInfo struct {
		SessionId  uuid.UUID `json:"sessionId"`
		PlayerName string    `json:"playerName"`
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorBadRequest("unable to parse auth header", w, err)
		return
	}

	uuid, err := auth.ValidateJWT(token, cfg.api.secret)
	if err != nil {
		ErrorUnauthorized("invalid token provided by client", w, err)
		return
	}

	saves, err := cfg.GetUsersSaves(req.Context(), uuid)
	if err != nil {
		ErrorServer("unable to query games", w, err)
		return
	}

	retVal := make([]GameInfo, 0)
	for _, save := range saves {
		retVal = append(retVal, GameInfo{SessionId: save.SessionID, PlayerName: save.Player.Name})
	}

	dat, err := json.Marshal(retVal)
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
		ErrorUnauthorized("invalid token provided by client", w, err)
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
		description += fmt.Sprintf("- Room %s\n", conn)
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
	fmt.Printf("%v", RetVal)
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
		ErrorUnauthorized("invalid token provided by client", w, err)
		return
	}
	sessionUUID, err := uuid.Parse(req.PathValue("uuid"))
	if err != nil {
		ErrorBadRequest("valid uuid not provided in path params", w, err)
		return
	}

	game, err := cfg.GetGamePartial(req.Context(), sessionUUID, "Ready")
	if err != nil && !strings.HasPrefix(err.Error(), "no game found") {
		ErrorServer("Failed to fetch game from server", w, err)
		return
	}

	if err == nil && game.Ready {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(204)
	}

	w.Header().Add("Content-Type", "application/json")
}

func (cfg *apiConfig) HandlerDeleteGame(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorBadRequest("unable to parse auth header", w, err)
		return
	}

	_, err = auth.ValidateJWT(token, cfg.api.secret)
	if err != nil {
		ErrorUnauthorized("invalid token provided by client", w, err)
		return
	}

	sessionUUID, err := uuid.Parse(req.PathValue("uuid"))
	if err != nil {
		ErrorBadRequest("valid uuid not provided in path params", w, err)
		return
	}

	// Delete S3 images before deleting from DynamoDB
	if err := cfg.s3Client.DeleteMapImages(req.Context(), sessionUUID); err != nil {
		fmt.Printf("Warning: Failed to delete S3 images for session %s: %v\n", sessionUUID, err)
		// Continue with deletion even if S3 cleanup fails
	}

	cfg.DeleteGame(req.Context(), sessionUUID)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)

}
