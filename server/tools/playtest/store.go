package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

// memStore stands in for DynamoDB: one session, one connection, its user,
// mutations and traces. The save makes the same round trip it makes through
// the table, and the version check is the table's optimistic lock.
type memStore struct {
	mu        sync.Mutex
	item      map[string]types.AttributeValue
	sessionID string
	streaming bool
	tokens    int
	mutations []game.MutationEntry
	traces    []*trace.Turn
}

func newMemStore(s game.SaveState) *memStore {
	item, err := db.EncodeSave(s)
	must(err)
	return &memStore{item: item, sessionID: s.SessionID}
}

func (m *memStore) conn() db.Connection {
	return db.Connection{ConnectionID: connID, UserID: db.BinaryID(userID), GameID: m.sessionID, Streaming: m.streaming}
}

func (m *memStore) GetConnection(_ context.Context, id string) (db.Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id != connID {
		return db.Connection{}, fmt.Errorf("connection not found: %s", id)
	}
	return m.conn(), nil
}

func (m *memStore) GetConnectionsByGameID(_ context.Context, gameID string) ([]db.Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gameID != m.sessionID {
		return nil, nil
	}
	return []db.Connection{m.conn()}, nil
}

func (m *memStore) SetStreaming(_ context.Context, _ string, on bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streaming = on
	return nil
}

func (m *memStore) DeleteConnection(context.Context, string) error {
	return fmt.Errorf("playtest: the handler tried to delete the player's connection")
}

func (m *memStore) GetUser(context.Context, string) (*db.UserRecord, error) {
	return &db.UserRecord{UserID: db.BinaryID(userID), Role: "admin", AIEnabled: true}, nil
}

func (m *memStore) UpdateUserTokens(_ context.Context, _ string, delta int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens += delta
	return nil
}

func (m *memStore) GetGame(context.Context, string) (game.SaveState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return db.DecodeSave(m.item)
}

func (m *memStore) PutGame(_ context.Context, s game.SaveState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, err := db.DecodeSave(m.item)
	if err != nil {
		return err
	}
	if s.Version != current.Version+1 {
		return fmt.Errorf("put game: ConditionalCheckFailedException (stored version %d, new version %d)", current.Version, s.Version)
	}
	item, err := db.EncodeSave(s)
	if err != nil {
		return fmt.Errorf("marshal save state: %w", err)
	}
	m.item = item
	return nil
}

func (m *memStore) PutMutation(_ context.Context, e game.MutationEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mutations = append(m.mutations, e)
	return nil
}

// PutTrace keeps the turn and checks it serialises, as the table store does.
func (m *memStore) PutTrace(_ context.Context, t *trace.Turn) error {
	if _, err := json.Marshal(t); err != nil {
		return fmt.Errorf("marshal trace: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traces = append(m.traces, t)
	return nil
}

// recorder stands in for the WebSocket API: it keeps every frame.
type recorder struct {
	mu     sync.Mutex
	frames []wsutil.Frame
	errors []string
}

func (r *recorder) add(f wsutil.Frame) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = append(r.frames, f)
	if f.Type == wsutil.FrameError {
		r.errors = append(r.errors, fmt.Sprint(f.Payload))
	}
}

func (r *recorder) Send(_ context.Context, _ string, f wsutil.Frame) error {
	r.add(f)
	return nil
}

func (r *recorder) Broadcast(_ context.Context, _ []string, f wsutil.Frame) ([]string, error) {
	r.add(f)
	return nil, nil
}

func (r *recorder) SendDelta(_ context.Context, _ string, delta any) error {
	r.add(wsutil.Frame{Type: wsutil.FrameStateDelta, Payload: delta})
	return nil
}

func (r *recorder) SendError(_ context.Context, _ string, message string) error {
	r.add(wsutil.Frame{Type: wsutil.FrameError, Payload: message})
	return nil
}

func (r *recorder) SendFullState(_ context.Context, _ string, state any) error {
	r.add(wsutil.Frame{Type: wsutil.FrameGameStateUpdate, Payload: state})
	return nil
}

// summary counts frames by type, in order of first appearance.
func (r *recorder) summary() string {
	var order []string
	counts := map[string]int{}
	for _, f := range r.frames {
		t := string(f.Type)
		if counts[t] == 0 {
			order = append(order, t)
		}
		counts[t]++
	}
	parts := make([]string, 0, len(order))
	for _, t := range order {
		parts = append(parts, fmt.Sprintf("%s×%d", t, counts[t]))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// streamed is the text the narrative chunks spelled out, as the client
// would show it; separate streams are separated by a blank line.
func (r *recorder) streamed() string {
	var b strings.Builder
	open := false
	for _, f := range r.frames {
		switch f.Type {
		case wsutil.FrameNarrativeChunk:
			b.WriteString(chunkText(f.Payload))
			open = true
		case wsutil.FrameNarrativeEnd:
			if open {
				b.WriteString("\n\n")
			}
			open = false
		}
	}
	return strings.TrimSpace(b.String())
}

func chunkText(payload any) string {
	switch v := payload.(type) {
	case string:
		return v
	case map[string]any:
		if s, ok := v["content"].(string); ok {
			return s
		}
		if s, ok := v["text"].(string); ok {
			return s
		}
	}
	raw, _ := json.Marshal(payload)
	var m map[string]any
	if json.Unmarshal(raw, &m) == nil {
		for _, k := range []string{"content", "text", "chunk"} {
			if s, ok := m[k].(string); ok {
				return s
			}
		}
	}
	return string(raw)
}
