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
	"github.com/rrochlin/an-amazing-adventure/internal/story"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

func main() {
	pipeline := flag.String("pipeline", "current", "candidate pipeline: current or recorded")
	only := flag.Int("turn", 0, "replay only this traced turn (1-based)")
	outPath := flag.String("o", "", "write the report here instead of stdout")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: replay [-pipeline current|recorded] [-turn N] [-o FILE] SESSION_ID")
		os.Exit(2)
	}
	sessionID := strings.TrimPrefix(flag.Arg(0), "game-")
	ctx := context.Background()

	client, err := db.New(ctx)
	if err != nil {
		fail(err)
	}
	save, err := client.GetGame(ctx, sessionID)
	if err != nil {
		fail(fmt.Errorf("get session: %w", err))
	}
	traces, err := client.ListTraces(ctx, sessionID)
	if err != nil {
		fail(fmt.Errorf("list traces: %w", err))
	}
	if *only > 0 {
		if *only > len(traces) {
			fail(fmt.Errorf("session has %d traced turns", len(traces)))
		}
		traces = traces[*only-1 : *only]
	}
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		fail(err)
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
	render(w, save, *pipeline, rn.replay(ctx, save, traces))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "replay:", err)
	os.Exit(1)
}
