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

	mux.HandleFunc("GET /", HandlerMain)

	var server = http.Server{
		Addr:    fmt.Sprintf("%v:%v", cfg.api.host, cfg.api.port),
		Handler: mux,
	}
	fmt.Println(server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("error encountered closing %v\n", err)
		return
	}

}

var handler = http.StripPrefix("/app/",
	http.FileServer(http.Dir(".")),
)

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
