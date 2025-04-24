package main

import (
	"encoding/json"
	"net/http"
)

func (cfg *apiConfig) HandlerStartGame(w http.ResponseWriter, req *http.Request) {
	type retVal struct {
		Positions [][]int  `json:"positions"`
		Player    Position `json:"player"`
		Cols      int      `json:"cols"`
		Rows      int      `json:"rows"`
	}

	RetVal := retVal{Positions: cfg.game.Maze, Player: cfg.game.Player.Pos, Cols: cfg.game.M, Rows: cfg.game.N}
	dat, err := json.Marshal(RetVal)
	if err != nil {
		ErrorServer("failed to get maze data", w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}

func (cfg *apiConfig) HandlerMove(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Pos Position `json:"position"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, nil)
		return
	}

	err = cfg.game.Move(params.Pos)
	if err != nil {
		ErrorBadRequest("Move failed", w, err)
		return
	}

	newInfo, err := cfg.game.Describe()
	if err != nil {
		return
	}
	type retVal struct {
		Positions map[Position]int `json:"positions"`
		Player    Position         `json:"player"`
	}
	RetVal := retVal{Positions: newInfo, Player: cfg.game.Player.Pos}
	dat, err := json.Marshal(RetVal)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func (cfg *apiConfig) HandlerDescribe(w http.ResponseWriter, req *http.Request) {
	newInfo, err := cfg.game.Describe()
	if err != nil {
		return
	}
	type retVal struct {
		Positions map[Position]int `json:"positions"`
		Player    Position         `json:"player"`
	}
	RetVal := retVal{Positions: newInfo, Player: cfg.game.Player.Pos}
	dat, err := json.Marshal(RetVal)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
