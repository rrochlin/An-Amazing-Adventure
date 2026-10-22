package main

import (
	"context"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

func TestDialogueChoice_TracedSenderRecordsFrames(t *testing.T) {
	g, _, conn, save := makeDialogueChoiceGame(t, true)
	tr := trace.Start(trace.KindDialogueChoice, "ws-game-action", save.SessionID, g)
	wsFake := &fakeDialogueChoiceSender{}

	if _, err := handleDialogueChoice(context.Background(), &fakeDialogueChoiceDB{connections: []db.Connection{conn}},
		tracedSender{wsFake, tr}, conn, g, &save, "2"); err != nil {
		t.Fatal(err)
	}
	tr.Finish(g)

	var types []string
	for _, f := range tr.Frames {
		types = append(types, f.Type)
	}
	// What the client received is what the trace shows.
	if len(types) != len(wsFake.frames)+wsFake.deltaCount {
		t.Fatalf("trace frames %v don't match %d broadcasts + %d deltas", types, len(wsFake.frames), wsFake.deltaCount)
	}
	if types[len(types)-1] != "state_delta" {
		t.Fatalf("expected the turn to end with a state delta, got %v", types)
	}
	if !strings.Contains(string(tr.After.Campaign), `"active_node_id":"route_failure"`) {
		t.Fatalf("after snapshot should show the new node, got %s", tr.After.Campaign)
	}
}

func TestDialogueChoice_TracedSenderRecordsErrors(t *testing.T) {
	g, _, conn, save := makeDialogueChoiceGame(t, true)
	tr := trace.Start(trace.KindDialogueChoice, "ws-game-action", save.SessionID, g)
	if _, err := handleDialogueChoice(context.Background(), &fakeDialogueChoiceDB{connections: []db.Connection{conn}},
		tracedSender{&fakeDialogueChoiceSender{}, tr}, conn, g, &save, "not-a-number"); err != nil {
		t.Fatal(err)
	}
	if len(tr.Errors) != 1 || !strings.Contains(tr.Errors[0], "invalid choice ID") {
		t.Fatalf("expected the rejection in the trace, got %v", tr.Errors)
	}
}
