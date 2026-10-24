// Package localstack runs the game's handlers without AWS: an in-memory
// stand-in for DynamoDB that stores saves exactly as the table does (the same
// round trip and version check), for the playtest tool and the local server.
package localstack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// Store holds sessions, connections, mutations and traces in memory. It
// implements what the ws-chat and ws-game-action handlers need from
// DynamoDB, and the trace store.
type Store struct {
	mu        sync.Mutex
	sessions  map[string]map[string]types.AttributeValue
	conns     map[string]db.Connection
	tokens    map[string]int
	mutations []game.MutationEntry
	traces    []*trace.Turn
	// OnTrace, when set, sees every turn trace as it is stored.
	OnTrace func(*trace.Turn)
	// File, when set, is where every save is written as it changes, and
	// read back by Load: sessions survive a restart.
	File string
}

func NewStore() *Store {
	return &Store{sessions: map[string]map[string]types.AttributeValue{}, conns: map[string]db.Connection{}, tokens: map[string]int{}}
}

// PutSession stores a save as is (a new game, or one loaded from disk).
func (s *Store) PutSession(save game.SaveState) error {
	item, err := db.EncodeSave(save)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[save.SessionID] = item
	return s.persist()
}

// Load reads back the sessions File holds, if any.
func (s *Store) Load() error {
	if s.File == "" {
		return nil
	}
	raw, err := os.ReadFile(s.File)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var saves []game.SaveState
	if err := json.Unmarshal(raw, &saves); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, save := range saves {
		item, err := db.EncodeSave(save)
		if err != nil {
			return err
		}
		s.sessions[save.SessionID] = item
	}
	return nil
}

// persist writes every session to File (callers hold the lock).
func (s *Store) persist() error {
	if s.File == "" {
		return nil
	}
	saves := make([]game.SaveState, 0, len(s.sessions))
	for _, item := range s.sessions {
		save, err := db.DecodeSave(item)
		if err != nil {
			return err
		}
		saves = append(saves, save)
	}
	raw, err := json.Marshal(saves)
	if err != nil {
		return err
	}
	return os.WriteFile(s.File, raw, 0o644)
}

// Sessions lists the stored session IDs.
func (s *Store) Sessions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// DeleteSession forgets a session.
func (s *Store) DeleteSession(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// Connect records a WebSocket connection to a game, as ws-connect does.
func (s *Store) Connect(connID, userID, gameID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[connID] = db.Connection{ConnectionID: connID, UserID: db.BinaryID(userID), GameID: gameID}
}

// Disconnect forgets a connection, as ws-disconnect does.
func (s *Store) Disconnect(connID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, connID)
}

// Traces returns the turn traces stored so far, and forgets them.
func (s *Store) Traces() []*trace.Turn {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.traces
	s.traces = nil
	return t
}

func (s *Store) GetConnection(_ context.Context, id string) (db.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conns[id]
	if !ok {
		return db.Connection{}, fmt.Errorf("connection not found: %s", id)
	}
	return c, nil
}

func (s *Store) GetConnectionsByGameID(_ context.Context, gameID string) ([]db.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []db.Connection
	for _, c := range s.conns {
		if c.GameID == gameID {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ConnectionID < out[j].ConnectionID })
	return out, nil
}

func (s *Store) SetStreaming(_ context.Context, id string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conns[id]
	if !ok {
		return fmt.Errorf("connection not found: %s", id)
	}
	c.Streaming = on
	s.conns[id] = c
	return nil
}

func (s *Store) DeleteConnection(_ context.Context, id string) error {
	s.Disconnect(id)
	return nil
}

func (s *Store) GetUser(_ context.Context, userID string) (*db.UserRecord, error) {
	return &db.UserRecord{UserID: db.BinaryID(userID), Role: "admin", AIEnabled: true}, nil
}

func (s *Store) UpdateUserTokens(_ context.Context, userID string, delta int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[userID] += delta
	return nil
}

func (s *Store) GetGame(_ context.Context, id string) (game.SaveState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessions[id]
	if !ok {
		return game.SaveState{}, fmt.Errorf("game not found: %s", id)
	}
	return db.DecodeSave(item)
}

// PutGame stores a save with the table's optimistic lock: the new version
// must follow the stored one.
func (s *Store) PutGame(_ context.Context, save game.SaveState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.sessions[save.SessionID]; ok {
		current, err := db.DecodeSave(item)
		if err != nil {
			return err
		}
		if save.Version != current.Version+1 {
			return fmt.Errorf("put game: ConditionalCheckFailedException (stored version %d, new version %d)", current.Version, save.Version)
		}
	}
	item, err := db.EncodeSave(save)
	if err != nil {
		return fmt.Errorf("marshal save state: %w", err)
	}
	s.sessions[save.SessionID] = item
	return s.persist()
}

func (s *Store) PutMutation(_ context.Context, e game.MutationEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mutations = append(s.mutations, e)
	return nil
}

// PutTrace keeps the turn and checks it serialises, as the table store does.
func (s *Store) PutTrace(_ context.Context, t *trace.Turn) error {
	if _, err := json.Marshal(t); err != nil {
		return fmt.Errorf("marshal trace: %w", err)
	}
	s.mu.Lock()
	s.traces = append(s.traces, t)
	hook := s.OnTrace
	s.mu.Unlock()
	if hook != nil {
		hook(t)
	}
	return nil
}
