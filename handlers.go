package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (cfg *apiConfig) HandlerStartGame(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Columns int `json:"columns"`
		Rows    int `json:"rows"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, nil)
		return
	}

	cfg.game, err = NewGame(params.Columns, params.Rows)

	type retVal struct {
		Player Position `json:"player"`
		Cols   int      `json:"cols"`
		Rows   int      `json:"rows"`
	}

	RetVal := retVal{
		Player: cfg.game.Player.Pos,
		Cols:   cfg.game.M,
		Rows:   cfg.game.N,
	}
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
	fmt.Printf("newInfo: %v\n", newInfo)
	type retVal struct {
		Positions []Position `json:"positions"`
		Values    []int      `json:"values"`
		Player    Position   `json:"player"`
	}

	keys := make([]Position, 0, len(newInfo))
	vals := make([]int, 0, len(newInfo))
	for p := range newInfo {
		keys = append(keys, p)
		vals = append(vals, newInfo[p])
	}

	RetVal := retVal{Positions: keys, Values: vals, Player: cfg.game.Player.Pos}
	dat, err := json.Marshal(RetVal)
	if err != nil {
		ErrorServer("Failed to marshal data", w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
