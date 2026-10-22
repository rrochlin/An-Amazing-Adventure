// Package trace records what happened during one handled turn — the model
// calls with their full prompts and raw output, the campaign state before and
// after, the resolver's verdict, and every frame sent to clients — so a
// session can be replayed turn by turn when debugging narration or state flow.
//
// Traces are written to TRACES_TABLE (one item per turn, expiring after
// Retention). They are diagnostic only: failing to record or store one never
// affects the turn.
package trace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// ErrDisabled is returned by a Store that has no traces table configured.
var ErrDisabled = errors.New("TRACES_TABLE is not set; turn tracing is disabled")

// Store holds traces. *db.Client implements it; handler test fakes may not.
type Store interface {
	PutTrace(context.Context, *Turn) error
}

var disabledOnce sync.Once

// Save finishes t against g and stores it if store can hold traces. It
// never fails the turn: problems are logged.
func Save(ctx context.Context, store any, t *Turn, g *game.Game) {
	t.Finish(g)
	s, ok := store.(Store)
	if !ok {
		return
	}
	if err := s.PutTrace(ctx, t); err != nil {
		if errors.Is(err, ErrDisabled) {
			disabledOnce.Do(func() { log.Printf("trace: %v", err) })
			return
		}
		log.Printf("trace: save %s turn for %s: %v", t.Kind, t.SessionID, err)
	}
}

// Retention is how long a trace is kept before DynamoDB TTL removes it.
const Retention = 30 * 24 * time.Hour

// Turn kinds.
const (
	KindChat           = "chat"
	KindDialogueChoice = "dialogue_choice"
	KindGameAction     = "game_action"
)

// Turn is the debugging record of one handled turn.
type Turn struct {
	SessionID    string `json:"session_id"`
	Kind         string `json:"kind"`
	Lambda       string `json:"lambda"`
	RequestID    string `json:"request_id,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	StartedAt    int64  `json:"started_at"` // unix milliseconds
	DurationMs   int64  `json:"duration_ms"`
	// Turn is the session's ConversationCount when the turn started.
	Turn int `json:"turn"`
	// Input is what the player sent: chat text, a choice, or an action payload.
	Input string `json:"input,omitempty"`

	Before *Snapshot `json:"before,omitempty"`
	After  *Snapshot `json:"after,omitempty"`

	Calls      []ModelCall       `json:"calls,omitempty"`
	Resolution any               `json:"resolution,omitempty"`
	Transition any               `json:"transition,omitempty"`
	Events     []game.WorldEvent `json:"events,omitempty"`
	Frames     []Frame           `json:"frames,omitempty"`
	Errors     []string          `json:"errors,omitempty"`

	mu sync.Mutex
}

// Snapshot is the state that decides what the story does next.
type Snapshot struct {
	PlayerRoomID   string `json:"player_room_id,omitempty"`
	PlayerRoomName string `json:"player_room_name,omitempty"`
	Version        int    `json:"version"`
	// Campaign is the full campaign runtime state (node, objectives, flags,
	// labels, dialogue), copied so later mutation can't change it.
	Campaign json.RawMessage `json:"campaign,omitempty"`
}

// ModelCall is one request to a model and what came back.
type ModelCall struct {
	Role   string `json:"role"` // narrator | engineer | resolver
	Model  string `json:"model"`
	System string `json:"system,omitempty"`
	// Messages is the conversation sent, for calls that send one (narrator).
	Messages []game.NarrativeMessage `json:"messages,omitempty"`
	// Input is the single user prompt, for calls that send one.
	Input      string `json:"input,omitempty"`
	Output     string `json:"output,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
	InputTok   int    `json:"input_tokens,omitempty"`
	OutputTok  int    `json:"output_tokens,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// Frame is a WebSocket frame sent during the turn. Streamed narrative chunks
// are folded into one entry with a count; the text is in the narrator call.
type Frame struct {
	Type    string `json:"type"`
	To      string `json:"to,omitempty"` // connection ID; empty means broadcast
	Count   int    `json:"count,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

// Start begins a trace for a turn.
func Start(kind, lambda, sessionID string, g *game.Game) *Turn {
	t := &Turn{
		SessionID: sessionID,
		Kind:      kind,
		Lambda:    lambda,
		StartedAt: time.Now().UnixMilli(),
	}
	if g != nil {
		t.Turn = g.ConversationCount
		t.Before = Snap(g)
	}
	return t
}

// Snap captures the story-relevant state of g.
func Snap(g *game.Game) *Snapshot {
	if g == nil {
		return nil
	}
	s := &Snapshot{Version: g.Version}
	if owner, ok := g.OwnerCharacter(); ok {
		s.PlayerRoomID = owner.LocationID
		if room, ok := g.Rooms[owner.LocationID]; ok {
			s.PlayerRoomName = room.Name
		}
	}
	if g.Campaign != nil {
		s.Campaign, _ = json.Marshal(g.Campaign)
	}
	return s
}

// Call records a model call.
func (t *Turn) Call(calls ...ModelCall) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Calls = append(t.Calls, calls...)
}

// Frame records a frame sent to `to` (empty for a broadcast).
func (t *Turn) Frame(frameType, to string, payload any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Frames = append(t.Frames, Frame{Type: frameType, To: to, Payload: payload})
}

// Chunk records one streamed narrative chunk, folding consecutive chunks.
func (t *Turn) Chunk(frameType string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n := len(t.Frames); n > 0 && t.Frames[n-1].Type == frameType && t.Frames[n-1].Count > 0 {
		t.Frames[n-1].Count++
		return
	}
	t.Frames = append(t.Frames, Frame{Type: frameType, Count: 1})
}

// Errorf records a problem that happened during the turn.
func (t *Turn) Errorf(format string, args ...any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Errors = append(t.Errors, fmt.Sprintf(format, args...))
}

// Finish stamps the duration and the after-state.
func (t *Turn) Finish(g *game.Game) {
	t.DurationMs = time.Now().UnixMilli() - t.StartedAt
	t.After = Snap(g)
}

// ExpiresAt is the TTL timestamp (epoch seconds) for the trace item.
func (t *Turn) ExpiresAt() int64 {
	return time.UnixMilli(t.StartedAt).Add(Retention).Unix()
}
