package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// referenceTurn is one traced chat turn saved in the repo, so it can be
// replayed after its trace has expired and without AWS access (with the
// recorded pipeline). The note says what the turn is a test of.
type referenceTurn struct {
	SessionID    string      `json:"session_id"`
	SessionTitle string      `json:"session_title"`
	Turn         int         `json:"turn"` // 1-based index among the session's traces
	Note         string      `json:"note,omitempty"`
	Trace        *trace.Turn `json:"trace"`
}

func referenceFile(dir string, r referenceTurn) string {
	return filepath.Join(dir, fmt.Sprintf("%s-%02d.json", r.SessionID[:8], r.Turn))
}

// saveReference writes the given traces (1-based turn numbers into all) as
// reference turns. A turn without a starting world can't be rebuilt exactly
// later, so it is refused.
func saveReference(dir string, save game.SaveState, all []*trace.Turn, turns []int, note string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var written []string
	for _, n := range turns {
		if n < 1 || n > len(all) {
			return written, fmt.Errorf("turn %d: session has %d traced turns", n, len(all))
		}
		t := all[n-1]
		if t.Kind != trace.KindChat {
			return written, fmt.Errorf("turn %d is a %s turn, not a chat turn", n, t.Kind)
		}
		if t.Before == nil || len(t.Before.World) == 0 {
			return written, fmt.Errorf("turn %d has no starting world; it predates world snapshots and can't be saved", n)
		}
		r := referenceTurn{SessionID: save.SessionID, SessionTitle: save.Title, Turn: n, Note: note, Trace: t}
		data, err := scrub(r)
		if err != nil {
			return written, err
		}
		path := referenceFile(dir, r)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return written, err
		}
		written = append(written, path)
	}
	return written, nil
}

// loadReference reads every reference turn in dir, in file order.
func loadReference(dir string) ([]referenceTurn, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []referenceTurn
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var r referenceTurn
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if r.Trace == nil {
			return nil, fmt.Errorf("%s: no trace", e.Name())
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SessionID != out[j].SessionID {
			return out[i].SessionID < out[j].SessionID
		}
		return out[i].Turn < out[j].Turn
	})
	return out, nil
}

// saveFor is the minimal session record a reference turn replays against:
// the starting world lives in the trace.
func (r referenceTurn) saveFor() game.SaveState {
	return game.SaveState{SessionID: r.SessionID, Title: r.SessionTitle}
}

// scrub serialises a reference turn without who played it or on which
// socket. The repo is public. The player's id also names the owner and keys
// the players map inside the starting world, so it is replaced everywhere,
// consistently, by "player"; the turn then rebuilds exactly as before.
func scrub(r referenceTurn) ([]byte, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	if uid := r.Trace.UserID; uid != "" {
		raw = bytes.ReplaceAll(raw, []byte(uid), []byte("player"))
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if tr, ok := doc["trace"].(map[string]any); ok {
		for _, k := range []string{"user_id", "connection_id", "request_id"} {
			delete(tr, k)
		}
		if frames, ok := tr["frames"].([]any); ok {
			for _, f := range frames {
				if fm, ok := f.(map[string]any); ok {
					delete(fm, "to")
				}
			}
		}
	}
	return json.MarshalIndent(doc, "", " ")
}
