package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rrochlin/an-amazing-adventure/internal/auth"
)

func (cfg *apiConfig) HandlerLogin(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, err)
		return
	}

	user, err := cfg.GetUserByEmail(req.Context(), params.Email)
	if err != nil {
		ErrorNotFound(fmt.Sprintf("%v", err), w, err)
		return
	}
	err = auth.CheckPasswordHash(user.HashedPassword, params.Password)
	if err != nil {
		ErrorUnauthorized("Incorrect Password", w)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.api.secret)
	if err != nil {
		ErrorServer(fmt.Sprintf("failed to construct JWT %v", err), w)
		return
	}
	refToken, _ := auth.MakeRefreshToken()
	_, err = cfg.db.query.CreateRToken(
		req.Context(),
		database.CreateRTokenParams{
			Token:  refToken,
			UserID: user.ID,
		})
	if err != nil {
		ErrorServer(fmt.Sprintf("Could not create refresh token: %v", err), w)
		return
	}

	dat, err := json.Marshal(toPublicUser(user, token, refToken))
	if err != nil {
		ErrorServer(fmt.Sprintf("failed to encode user for response %v", err), w)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}

type PublicUser struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
}

func toPublicUser(user auth.User, token, refresh_token string) PublicUser {
	return PublicUser{
		ID:           user.ID,
		Email:        user.Email,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Token:        token,
		RefreshToken: refresh_token,
	}
}
