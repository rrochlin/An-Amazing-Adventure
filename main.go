package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	game, err := NewGame(40, 40)
	if err != nil {
		fmt.Printf("fatal error, game object creation failed: %v", err)
		return
	}

	mux := http.NewServeMux()
	cfg := apiConfig{
		api: apiSettings{
			host: os.Getenv("HOST_URL"),
			port: os.Getenv("PORT"),
		},
		game: game,
	}

	mux.HandleFunc("POST /api/move", cfg.HandlerMove)
	mux.HandleFunc("GET /api/describe", cfg.HandlerDescribe)
	mux.HandleFunc("GET /api/startgame", cfg.HandlerStartGame)

	wrappedMux := NewLogger(mux)

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
	api  apiSettings
	game Game
}
