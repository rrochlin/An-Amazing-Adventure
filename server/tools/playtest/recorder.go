package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

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
