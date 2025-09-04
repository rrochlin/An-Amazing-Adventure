package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"google.golang.org/genai"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("warning: assuming default configuration. .env unreadable: %v", err)
	}
	log.Printf("%v", os.Getenv("AWS_SECRET_ACCESS_KEY"))

	mux := http.NewServeMux()
	awsCfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("us-west-2"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	svc := dynamodb.NewFromConfig(awsCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GCP_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal("Failed to create Gemini client:", err)
	}

	partsys := genai.Part{Text: GetSystemInstructions()}
	psys := make([]*genai.Part, 1)
	psys[0] = &partsys
	instructions := genai.Content{Role: "system", Parts: psys}

	config := &genai.GenerateContentConfig{
		Temperature:       genai.Ptr[float32](0.7),
		TopP:              genai.Ptr[float32](0.8),
		MaxOutputTokens:   20000,
		SystemInstruction: &instructions,
	}

	cfg := apiConfig{
		api: apiSettings{
			host:         os.Getenv("HOST_URL"),
			port:         os.Getenv("PORT"),
			secret:       os.Getenv("SECRET"),
			model:        "gemini-2.5-flash",
			usersTable:   os.Getenv("AWS_USERS_TABLE"),
			sessionTable: os.Getenv("AWS_SESSION_TABLE"),
			rTokensTable: os.Getenv("AWS_R_TOKENS_TABLE"),
		},
		gemini:      client,
		chatConfig:  config,
		dynamodbSvc: svc,
	}

	// game routes
	mux.HandleFunc("GET /api/describe/{uuid}", cfg.HandlerDescribe)
	mux.HandleFunc("POST /api/startgame/{uuid}", cfg.HandlerStartGame)
	mux.HandleFunc("POST /api/chat/{uuid}", cfg.HandlerChat)
	mux.HandleFunc("GET /api/worldready/{uuid}", cfg.HandlerWorldReady)

	// user routes
	mux.HandleFunc("POST /api/login", cfg.HandlerLogin)
	mux.HandleFunc("POST /api/refresh", cfg.HandlerRefresh)
	mux.HandleFunc("POST /api/revoke", cfg.HandlerRevoke)
	mux.HandleFunc("PUT /api/users", cfg.HandlerUpdateUser)
	mux.HandleFunc("POST /api/users", cfg.HandlerUsers)

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
	host         string
	port         string
	secret       string
	model        string
	usersTable   string
	sessionTable string
	rTokensTable string
}

// Main configuration struct
type apiConfig struct {
	api         apiSettings
	gemini      *genai.Client
	chatConfig  *genai.GenerateContentConfig
	dynamodbSvc *dynamodb.Client
}
