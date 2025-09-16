package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
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
		ctx,
		&dynamodb.PutItemInput{
			TableName: aws.String(cfg.api.sessionTable),
			Item:      item,
		},
	)

	if err != nil {
		return err
	}
	return nil
}

func (cfg *apiConfig) GetUsersSaves(ctx context.Context, userId uuid.UUID) ([]SaveState, error) {
	keyEx := expression.Key("user_id").Equal(expression.Value(userId))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, err
	}
	out, err := cfg.dynamodbSvc.Query(
		ctx,
		&dynamodb.QueryInput{
			TableName:                 aws.String(cfg.api.sessionTable),
			IndexName:                 aws.String("user-sessions-index"),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			KeyConditionExpression:    expr.KeyCondition(),
		},
	)
	if err != nil {
		return nil, err
	}

	var saves []SaveState
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &saves); err != nil {
		return nil, err
	}

	return saves, nil

}

func (cfg *apiConfig) GetGame(ctx context.Context, sessionId uuid.UUID) (Game, error) {
	rawUuid, err := sessionId.MarshalBinary()
	if err != nil {
		return Game{}, err
	}

	key := map[string]types.AttributeValue{
		"session_id": &types.AttributeValueMemberB{Value: rawUuid},
	}
	out, err := cfg.dynamodbSvc.GetItem(
		ctx,
		&dynamodb.GetItemInput{
			Key:       key,
			TableName: aws.String(cfg.api.sessionTable),
		},
	)
	if err != nil {
		return Game{}, err
	}

	if out.Item == nil {
		return Game{}, fmt.Errorf("no game found for session_id %s", sessionId)
	}

	var save SaveState
	if err := attributevalue.UnmarshalMap(out.Item, &save); err != nil {
		return Game{}, err
	}

	fmt.Printf("CURRENT SAVE STATE\n%v", save)
	return save.LoadGame(), nil
}

func (cfg *apiConfig) GetGamePartial(ctx context.Context, sessionId uuid.UUID, projectionExpression string) (Game, error) {
	rawUuid, err := sessionId.MarshalBinary()
	if err != nil {
		return Game{}, err
	}

	key := map[string]types.AttributeValue{
		"session_id": &types.AttributeValueMemberB{Value: rawUuid},
	}
	out, err := cfg.dynamodbSvc.GetItem(
		ctx,
		&dynamodb.GetItemInput{
			Key:                  key,
			TableName:            aws.String(cfg.api.sessionTable),
			ProjectionExpression: &projectionExpression,
		},
	)
	if err != nil {
		return Game{}, err
	}

	if out.Item == nil {
		return Game{}, fmt.Errorf("no game found for session_id %s", sessionId)
	}

	var save SaveState
	if err := attributevalue.UnmarshalMap(out.Item, &save); err != nil {
		return Game{}, err
	}

	fmt.Printf("CURRENT SAVE STATE\n%v", save)
	return save.LoadGame(), nil

}

func (cfg *apiConfig) CreateUser(ctx context.Context, user auth.User) error {
	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		return err
	}
	_, err = cfg.dynamodbSvc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &cfg.api.usersTable,
		Item:      item,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cfg *apiConfig) UpdateUser(ctx context.Context, user auth.User) error {
	// we're just calling create user to update
	return cfg.CreateUser(ctx, user)
}

func (cfg *apiConfig) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	keyEx := expression.Key("email").Equal(expression.Value(email))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return auth.User{}, err
	}
	out, err := cfg.dynamodbSvc.Query(
		ctx,
		&dynamodb.QueryInput{
			TableName:                 aws.String(cfg.api.usersTable),
			IndexName:                 aws.String("email-index"),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			KeyConditionExpression:    expr.KeyCondition(),
		},
	)
	if err != nil {
		return auth.User{}, err
	}

	var users []auth.User
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &users); err != nil {
		return auth.User{}, err
	}

	if len(users) != 1 {
		return auth.User{}, fmt.Errorf("found %v number of users with email", len(users))
	}

	return users[0], nil
}

func (cfg *apiConfig) GetUserByUUID(ctx context.Context, userUUID uuid.UUID) (auth.User, error) {
	rawUuid, err := userUUID.MarshalBinary()
	if err != nil {
		return auth.User{}, err
	}

	key := map[string]types.AttributeValue{
		"user_id": &types.AttributeValueMemberB{Value: rawUuid},
	}
	out, err := cfg.dynamodbSvc.GetItem(
		ctx,
		&dynamodb.GetItemInput{
			TableName: aws.String(cfg.api.usersTable),
			Key:       key,
		},
	)
	if err != nil {
		return auth.User{}, err
	}

	var user auth.User
	if err = attributevalue.UnmarshalMap(out.Item, &user); err != nil {
		return auth.User{}, fmt.Errorf("unable to parse user from database %v", err)
	}
	return user, nil
}

type CreateRTokenParams struct {
	Token  string    `json:"token"`
	UserID uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) CreateRToken(ctx context.Context, arg CreateRTokenParams) (auth.RefreshToken, error) {
	token := auth.RefreshToken{
		Token:     arg.Token,
		UserID:    arg.UserID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	item, err := attributevalue.MarshalMap(token)
	if err != nil {
		return auth.RefreshToken{}, err
	}
	_, err = cfg.dynamodbSvc.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &cfg.api.rTokensTable,
		Item:      item,
	})
	if err != nil {
		return auth.RefreshToken{}, err
	}

	return token, err
}

func (cfg *apiConfig) GetRToken(ctx context.Context, token string) (auth.RefreshToken, error) {
	key := map[string]types.AttributeValue{
		"token": &types.AttributeValueMemberS{Value: token},
	}
	out, err := cfg.dynamodbSvc.GetItem(
		ctx,
		&dynamodb.GetItemInput{
			Key:       key,
			TableName: aws.String(cfg.api.rTokensTable),
		},
	)
	if err != nil {
		return auth.RefreshToken{}, err
	}

	if out.Item == nil {
		return auth.RefreshToken{}, fmt.Errorf("refresh token not found")
	}

	var rToken auth.RefreshToken
	if err := attributevalue.UnmarshalMap(out.Item, &rToken); err != nil {
		return auth.RefreshToken{}, err
	}

	return rToken, err
}

func (cfg *apiConfig) RevokeToken(ctx context.Context, token string) error {
	key := map[string]types.AttributeValue{
		"token": &types.AttributeValueMemberS{Value: token},
	}
	var err error
	var response *dynamodb.UpdateItemOutput
	var attributeMap map[string]map[string]any
	update := expression.Set(expression.Name("revoked_at"), expression.Value(time.Now()))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		log.Printf("Couldn't build expression for update. Here's why: %v\n", err)
	} else {
		response, err = cfg.dynamodbSvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName:                 aws.String(cfg.api.rTokensTable),
			Key:                       key,
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			UpdateExpression:          expr.Update(),
			ReturnValues:              types.ReturnValueUpdatedNew,
		})
		if err != nil {
			log.Printf("Couldn't revoke token %v. Here's why: %v\n", token, err)
		} else {
			err = attributevalue.UnmarshalMap(response.Attributes, &attributeMap)
			if err != nil {
				log.Printf("Couldn't unmarshall update response. Here's why: %v\n", err)
			}
		}
	}
	return err
}

func (cfg *apiConfig) RefreshToken(ctx context.Context, token string) error {
	key := map[string]types.AttributeValue{
		"token": &types.AttributeValueMemberS{Value: token},
	}
	var err error
	var response *dynamodb.UpdateItemOutput
	var attributeMap map[string]map[string]any
	update := expression.Set(expression.Name("expires_at"), expression.Value(time.Now().Add(time.Hour)))
	update.Add(expression.Name("updated_at"), expression.Value(time.Now()))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		log.Printf("Couldn't build expression for update. Here's why: %v\n", err)
	} else {
		response, err = cfg.dynamodbSvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName:                 aws.String(cfg.api.rTokensTable),
			Key:                       key,
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			UpdateExpression:          expr.Update(),
			ReturnValues:              types.ReturnValueUpdatedNew,
		})
		if err != nil {
			log.Printf("Couldn't revoke token %v. Here's why: %v\n", token, err)
		} else {
			err = attributevalue.UnmarshalMap(response.Attributes, &attributeMap)
			if err != nil {
				log.Printf("Couldn't unmarshall update response. Here's why: %v\n", err)
			}
		}
	}
	return err
}
