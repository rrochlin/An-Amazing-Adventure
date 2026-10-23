// Command replay reruns a session's traced chat turns through a story
// pipeline and writes a side-by-side report against what originally
// happened: narration, campaign state changes, model calls, tokens, and
// latency against the turn budget. It reads only; nothing is written.
//
// Usage (from server/, with read access to the tables):
//
//	SESSIONS_TABLE=amazing-adventure-prod-sessions \
//	TRACES_TABLE=amazing-adventure-prod-traces \
//	AWS_PROFILE=aa-debug AWS_REGION=us-west-2 \
//	go run ./tools/replay [-pipeline current|recorded] [-turn N] [-o report.md] SESSION_ID
//
// -pipeline current (the default) reruns the game's own pipeline, which calls
// Bedrock and needs credentials that can invoke the models. -pipeline recorded
// replays what each turn originally said without calling a model, to check
// the rebuild and the report. -turn N replays only the Nth traced turn
// (1-based, as numbered in the report).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/story"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

func main() {
	pipeline := flag.String("pipeline", "current", "candidate pipeline: current or recorded")
	only := flag.String("turn", "", "replay (or save) only these traced turns, 1-based, e.g. 3 or 2,3,5")
	outPath := flag.String("o", "", "write the report here instead of stdout")
	saveDir := flag.String("save", "", "save the selected turns as reference turns in this directory instead of replaying")
	note := flag.String("note", "", "with -save: what the saved turns are a test of")
	fromDir := flag.String("from", "", "replay the reference turns in this directory instead of a session")
	flag.Parse()
	if (flag.NArg() != 1) == (*fromDir == "") {
		fmt.Fprintln(os.Stderr, "usage: replay [-pipeline current|recorded] [-turn N[,N]] [-o FILE] SESSION_ID\n"+
			"       replay -save DIR [-turn N[,N]] [-note TEXT] SESSION_ID\n"+
			"       replay -from DIR [-pipeline current|recorded] [-o FILE]")
		os.Exit(2)
	}
	ctx := context.Background()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		fail(err)
	}

	var runs []struct {
		title  string
		save   game.SaveState
		traces []*trace.Turn
	}
	if *fromDir != "" {
		refs, err := loadReference(*fromDir)
		if err != nil {
			fail(fmt.Errorf("load reference turns: %w", err))
		}
		for _, r := range refs {
			runs = append(runs, struct {
				title  string
				save   game.SaveState
				traces []*trace.Turn
			}{fmt.Sprintf("%s turn %d: %s", r.SessionID[:8], r.Turn, r.Note), r.saveFor(), []*trace.Turn{r.Trace}})
		}
	} else {
		sessionID := strings.TrimPrefix(flag.Arg(0), "game-")
		client, err := db.New(ctx)
		if err != nil {
			fail(err)
		}
		traces, err := client.ListTraces(ctx, sessionID)
		if err != nil {
			fail(fmt.Errorf("list traces: %w", err))
		}
		// A deleted session leaves its traces behind (30-day TTL); each
		// chat trace carries its starting world, so they still replay.
		save, err := client.GetGame(ctx, sessionID)
		if err != nil {
			if len(traces) == 0 {
				fail(fmt.Errorf("get session: %w", err))
			}
			fmt.Fprintf(os.Stderr, "replay: session record not found (%v); using its %d traces\n", err, len(traces))
			save = game.SaveState{SessionID: sessionID}
		}
		turns, err := parseTurns(*only, len(traces))
		if err != nil {
			fail(err)
		}
		if *saveDir != "" {
			written, err := saveReference(*saveDir, save, traces, turns, *note)
			for _, w := range written {
				fmt.Println("saved", w)
			}
			if err != nil {
				fail(err)
			}
			return
		}
		if len(turns) > 0 {
			picked := make([]*trace.Turn, 0, len(turns))
			for _, n := range turns {
				picked = append(picked, traces[n-1])
			}
			traces = picked
		}
		runs = append(runs, struct {
			title  string
			save   game.SaveState
			traces []*trace.Turn
		}{"", save, traces})
	}

	rn := runner{Registry: reg}
	switch *pipeline {
	case "current":
		models, err := ai.New(ctx)
		if err != nil {
			fail(fmt.Errorf("bedrock client: %w", err))
		}
		rn.Pipeline = func(*trace.Turn) story.Pipeline { return story.Current{Models: models} }
	case "recorded":
		rn.Pipeline = func(t *trace.Turn) story.Pipeline { return recorded{t} }
	default:
		fail(fmt.Errorf("unknown pipeline %q", *pipeline))
	}

	var w io.Writer = os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fail(err)
		}
		defer f.Close()
		w = f
	}
	for _, run := range runs {
		if run.title != "" {
			fmt.Fprintf(w, "# Reference: %s\n\n", run.title)
		}
		render(w, run.save, *pipeline, rn.replay(ctx, run.save, run.traces))
	}
}

// parseTurns reads "2,3,5" into 1-based turn numbers, all within n.
func parseTurns(spec string, n int) ([]int, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	var out []int
	for _, part := range strings.Split(spec, ",") {
		var t int
		if _, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &t); err != nil || t < 1 {
			return nil, fmt.Errorf("bad turn %q", part)
		}
		if t > n {
			return nil, fmt.Errorf("turn %d: session has %d traced turns", t, n)
		}
		out = append(out, t)
	}
	return out, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "replay:", err)
	os.Exit(1)
}
