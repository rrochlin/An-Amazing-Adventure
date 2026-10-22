// Command sessiondump prints one game session as a turn-by-turn debugging
// report: the player-visible transcript, then each traced turn with the
// campaign state change, model calls, resolver verdict, events and frames,
// then the engineer's mutation log. Read-only.
//
// Usage (from server/, with read access to the tables):
//
//	SESSIONS_TABLE=amazing-adventure-prod-sessions \
//	MUTATIONS_TABLE=amazing-adventure-prod-mutations \
//	TRACES_TABLE=amazing-adventure-prod-traces \
//	AWS_REGION=us-west-2 go run ./tools/sessiondump [-full] [-json] SESSION_ID
//
// -full adds every model call's system prompt, messages and raw output.
// -json prints the raw session, traces and mutations instead of a report.
// SESSION_ID may be a bare UUID or the route form game-{uuid}.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

func main() {
	full := flag.Bool("full", false, "include full prompts, messages and model output")
	asJSON := flag.Bool("json", false, "print raw JSON instead of a report")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: sessiondump [-full] [-json] SESSION_ID")
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
	if err != nil && !errors.Is(err, trace.ErrDisabled) {
		fail(fmt.Errorf("list traces: %w", err))
	}
	var mutations []game.MutationEntry
	if os.Getenv("MUTATIONS_TABLE") != "" {
		if mutations, err = client.ListMutations(ctx, sessionID); err != nil {
			fail(fmt.Errorf("list mutations: %w", err))
		}
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{"session": save, "traces": traces, "mutations": mutations}); err != nil {
			fail(err)
		}
		return
	}
	render(os.Stdout, save, traces, mutations, *full)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "sessiondump:", err)
	os.Exit(1)
}
