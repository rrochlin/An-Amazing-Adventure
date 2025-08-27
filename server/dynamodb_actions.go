package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/rrochlin/an-amazing-adventure/internal/auth"
)

func (cfg *apiConfig) PutGame(ctx context.Context, saveState SaveState) error {
	item, err := attributevalue.MarshalMap(saveState)
	if err != nil {
		return err
	}
	_, err = cfg.dynamodbSvc.PutItem(
		context.Background(),
		&dynamodb.PutItemInput{
			TableName: aws.String(os.Getenv("AWS_TABLE_NAME")),
			Item:      item,
		},
	)

	if err != nil {
		return err
	}
	return nil
}

func (cfg *apiConfig) GetGame(ctx context.Context, sessionId uuid.UUID) (GameState, error) {
	key := map[string]types.AttributeValue{
		"session_id": &types.AttributeValueMemberS{Value: sessionId.String()},
	}
	out, err := cfg.dynamodbSvc.GetItem(
		ctx,
		&dynamodb.GetItemInput{
			Key:       key,
			TableName: aws.String(os.Getenv("AWS_TABLE_NAME")),
		},
	)
	if err != nil {
		return GameState{}, err
	}

	if out.Item == nil {
		return GameState{}, fmt.Errorf("no game found for session_id %s", sessionId)
	}

	var save SaveState
	if err := attributevalue.UnmarshalMap(out.Item, &save); err != nil {
		return GameState{}, err
	}

	return GameState{}, nil
}

func (cfg *apiConfig) CreateUser(ctx context.Context, user auth.User) error {
	return nil
}

func (cfg *apiConfig) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	return auth.User{}, nil
}

func (cfg *apiConfig) GetUserByJWT(ctx context.Context, jwt auth.RefreshToken) (auth.User, error) {
	return auth.User{}, nil
}
