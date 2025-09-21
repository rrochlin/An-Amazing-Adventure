package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rrochlin/an-amazing-adventure/internal/auth"
)

func (cfg *apiConfig) HandlerRefresh(w http.ResponseWriter, req *http.Request) {
	untrustedToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorBadRequest(err.Error(), w, err)
		return
	}
	token, err := cfg.GetRToken(req.Context(), untrustedToken)
	if err != nil {
		ErrorUnauthorized(err.Error(), w, err)
		return
	}
	if token.RevokedAt != nil {
		msg := fmt.Sprintf("Refresh Token has been revoked at %v", token.RevokedAt)
		ErrorUnauthorized(msg, w, nil)
		return
	}

	err = cfg.RefreshToken(req.Context(), token.Token)

	jtoken, err := auth.MakeJWT(token.UserID, cfg.api.secret)
	if err != nil {
		ErrorServer(err.Error(), w, err)
		return
	}
	type response struct {
		Token string `json:"token"`
	}
	res := response{Token: jtoken}

	dat, err := json.Marshal(res)
	if err != nil {
		ErrorServer(fmt.Sprintf("failed to encode token for response %v", err), w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}

func (cfg *apiConfig) HandlerRevoke(w http.ResponseWriter, req *http.Request) {
	untrustedToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		ErrorBadRequest(err.Error(), w, err)
		return
	}
	err = cfg.RevokeToken(req.Context(), untrustedToken)
	if err != nil {
		ErrorUnauthorized(err.Error(), w, err)
		return
	}
	w.WriteHeader(204)

}
