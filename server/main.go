package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"google.golang.org/genai"
)

func main() {
	godotenv.Load()

	mux := http.NewServeMux()

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  os.Getenv("GCP_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	partsys := genai.Part{Text: GetSystemInstructions()}
	psys := make([]*genai.Part, 1)
	psys[0] = &partsys
	instructions := genai.Content{Role: "system", Parts: psys}

	var config *genai.GenerateContentConfig = &genai.GenerateContentConfig{
		Temperature:       genai.Ptr[float32](0.5),
		SystemInstruction: &instructions,
	}

	// Create a new Chat.
	chat, err := client.Chats.Create(context.Background(), *model, config, nil)
	if err != nil {
		log.Fatal(err)
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

	mux.HandleFunc("POST /api/move", cfg.HandlerMove)
	mux.HandleFunc("GET /api/describe", cfg.HandlerDescribe)
	mux.HandleFunc("POST /api/startgame", cfg.HandlerStartGame)
	mux.HandleFunc("POST /api/chat", cfg.HandlerChat)

	wrappedMux := cors.Default().Handler(NewLogger(mux))

	var server = http.Server{
		Addr:    fmt.Sprintf("%v:%v", cfg.api.host, cfg.api.port),
		Handler: wrappedMux,
	}
	fmt.Println(server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("error encountered closing %v\n", err)
		return
	}

}

// API related configuration
type apiSettings struct {
	host string
	port string
}

// Database related configuration

// Main configuration struct
type apiConfig struct {
	api    apiSettings
	game   Game
	gemini *genai.Client
	chat   *genai.Chat
}
