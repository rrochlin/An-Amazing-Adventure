package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	mux := http.NewServeMux()
	cfg := apiConfig{
		api: apiSettings{
			host: os.Getenv("HOST_URL"),
			port: os.Getenv("PORT"),
		},
	}

	mux.Handle("GET /", handler)

	wrappedMux := NewLogger(mux)

	var server = http.Server{
		Addr:    fmt.Sprintf("%v:%v", cfg.api.host, cfg.api.port),
		Handler: wrappedMux,
	}
	fmt.Println(server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("error encountered closing %v\n", err)
		return
	}

}

var handler = http.StripPrefix("/", http.FileServer(http.Dir("static")))

// API related configuration
type apiSettings struct {
	host string
	port string
}

// Database related configuration

// Main configuration struct
type apiConfig struct {
	api apiSettings
}
