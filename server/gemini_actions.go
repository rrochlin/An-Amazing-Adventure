package main

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

func (cfg *apiConfig) CreateChat(
	ctx context.Context,
	sessionUUID uuid.UUID,
	history []*genai.Content,
) (*genai.Chat, error) {
	chat, err := cfg.gemini.Chats.Create(
		ctx,
		cfg.api.model,
		cfg.chatConfig,
		history,
	)
	if err != nil {
		return &genai.Chat{}, err
	}
	return chat, nil

}
