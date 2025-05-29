package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"google.golang.org/genai"
)

func main() {
	godotenv.Load()

	mux := http.NewServeMux()

	// Create a context with timeout for API client initialization
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize Gemini API client with proper configuration
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GCP_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal("Failed to create Gemini client:", err)
	}

	// Create system instructions
	partsys := genai.Part{Text: GetSystemInstructions()}
	psys := make([]*genai.Part, 1)
	psys[0] = &partsys
	instructions := genai.Content{Role: "system", Parts: psys}

	// Configure model parameters
	config := &genai.GenerateContentConfig{
		Temperature:       genai.Ptr[float32](0.7),
		TopP:              genai.Ptr[float32](0.8),
		MaxOutputTokens:   20000,
		SystemInstruction: &instructions,
	}

	// Create a new Chat.
	chat, err := client.Chats.Create(context.Background(), *model, config, nil)
	if err != nil {
		log.Fatal("Failed to create chat session after retries:", err)
	}

	cfg := apiConfig{
		api: apiSettings{
			host: os.Getenv("HOST_URL"),
			port: os.Getenv("PORT"),
		},
		game:   Game{},
		gemini: client,
		chat:   chat,
	}

	// Set up routes
	mux.HandleFunc("GET /api/describe", cfg.HandlerDescribe)
	mux.HandleFunc("POST /api/startgame", cfg.gameStateMiddleware(cfg.HandlerStartGame))
	mux.HandleFunc("POST /api/chat", cfg.gameStateMiddleware(cfg.HandlerChat))
	mux.HandleFunc("GET /api/worldready", cfg.HandlerWorldReady)

	wrappedMux := cors.Default().Handler(NewLogger(mux))

	server := &http.Server{
		Addr:         fmt.Sprintf("%v:%v", cfg.api.host, cfg.api.port),
		Handler:      wrappedMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// API related configuration
type apiSettings struct {
	host string
	port string
}

// Main configuration struct
type apiConfig struct {
	api      apiSettings
	game     Game
	worldGen *WorldGenerator
	gemini   *genai.Client
	chat     *genai.Chat
}
