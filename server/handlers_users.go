package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rrochlin/an-amazing-adventure/internal/auth"
)

func (cfg *apiConfig) HandlerUsers(w http.ResponseWriter, req *http.Request) {
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
	hashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		ErrorServer(fmt.Sprintf("Passowrd hash failed: %v", err), w, err)
		return
	}

	user := auth.User{
		ID:             uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Email:          params.Email,
		HashedPassword: hashedPass,
	}
	if err = cfg.CreateUser(req.Context(), user); err != nil {
		ErrorServer(fmt.Sprintf("Could not create user: %v", err), w, err)
		return
	}
	dat, err := json.Marshal(toPublicUser(user, "", ""))
	if err != nil {
		ErrorServer(fmt.Sprintf("Could not convert user to response: %v", err), w, err)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)
}

func (cfg *apiConfig) HandlerUpdateUser(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	unverifiedToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorUnauthorized(err.Error(), w, err)
		return
	}
	uuid, err := auth.ValidateJWT(unverifiedToken, cfg.api.secret)
	if err != nil {
		ErrorUnauthorized(err.Error(), w, err)
		return
	}
	var params parameters
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&params)
	if err != nil {
		ErrorServer(err.Error(), w, err)
		return
	}
	hpass, err := auth.HashPassword(params.Password)
	if err != nil {
		ErrorServer(err.Error(), w, err)
		return
	}
	user, err := cfg.GetUserByUUID(req.Context(), uuid)
	if err != nil {
		ErrorNotFound("user data not found", w, err)
		return
	}
	user.Email = params.Email
	user.HashedPassword = hpass
	user.UpdatedAt = time.Now()

	err = cfg.UpdateUser(
		req.Context(),
		user,
	)
	if err != nil {
		ErrorServer(err.Error(), w, err)
		return
	}
	dat, err := json.Marshal(toPublicUser(user, "", ""))
	if err != nil {
		ErrorServer(err.Error(), w, err)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}
