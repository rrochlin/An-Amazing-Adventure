package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// tracingDBClient is a fakeDBClient that can also store traces.
type tracingDBClient struct {
	*fakeDBClient
	traces []*trace.Turn
}

func (f *tracingDBClient) PutTrace(_ context.Context, t *trace.Turn) error {
	f.traces = append(f.traces, t)
	return nil
}

func TestHandlerChat_RecordsTurnTrace(t *testing.T) {
	t.Setenv("MUTATIONS_TABLE", "mutations-test")
	save := makeBootstrappedCampaignSaveState(t, "lichs-labyrinth")
	conn := db.Connection{ConnectionID: "conn-2", GameID: save.SessionID, UserID: db.BinaryID("user-1")}
	dbFake := &tracingDBClient{fakeDBClient: &fakeDBClient{
		connection:      conn,
		connectionsByID: []db.Connection{conn},
		user:            &db.UserRecord{UserID: db.BinaryID("user-1"), AIEnabled: true},
		save:            save,
	}}
	wsFake := &fakeWSSender{}
	aiFake := &fakeAIClient{
		narrator: ai.NarratorResult{
			Narrative: "The briefing settles into a plan.",
			Calls:     []trace.ModelCall{{Role: "narrator", System: "narrator prompt", Output: "The briefing settles into a plan."}},
		},
		engineer: ai.EngineerResult{Calls: []trace.ModelCall{{Role: "engineer"}}},
		resolver: ai.CampaignConditionResolutionResult{
			Resolution: campaigns.ConditionResolution{BoolFlags: map[string]bool{"briefing_complete": true}},
			Calls:      []trace.ModelCall{{Role: "resolver", Output: `{"bool_flags":{"briefing_complete":true}}`}},
		},
	}

	origNewDBClient, origNewWSSender, origNewAIClient := newDBClient, newWSSender, newAIClient
	newDBClient = func(context.Context) (dbClient, error) { return dbFake, nil }
	newWSSender = func(context.Context) (wsSender, error) { return wsFake, nil }
	newAIClient = func(context.Context) (aiClient, error) { return aiFake, nil }
	t.Cleanup(func() {
		newDBClient = origNewDBClient
		newWSSender = origNewWSSender
		newAIClient = origNewAIClient
	})

	body, _ := json.Marshal(chatRequest{Action: "chat", Content: "we are ready"})
	if _, err := handler(context.Background(), makeWSChatReq("conn-2", string(body))); err != nil {
		t.Fatalf("handler err: %v", err)
	}

	if len(dbFake.traces) != 1 {
		t.Fatalf("expected one trace, got %d", len(dbFake.traces))
	}
	tr := dbFake.traces[0]
	if tr.Kind != trace.KindChat || tr.SessionID != save.SessionID || tr.Input != "we are ready" || tr.UserID != "user-1" {
		t.Fatalf("unexpected trace header %+v", tr)
	}
	var roles []string
	for _, c := range tr.Calls {
		roles = append(roles, c.Role)
	}
	if strings.Join(roles, ",") != "narrator,engineer,resolver" {
		t.Fatalf("expected narrator, engineer, resolver calls, got %v", roles)
	}
	if !strings.Contains(string(tr.Before.Campaign), `"active_node_id":"camp_briefing"`) {
		t.Fatalf("before snapshot should be on the briefing node, got %s", tr.Before.Campaign)
	}
	if !strings.Contains(string(tr.After.Campaign), `"active_node_id":"choose_entry_plan"`) {
		t.Fatalf("after snapshot should be on the next node, got %s", tr.After.Campaign)
	}
	if tr.Resolution == nil || tr.Transition == nil {
		t.Fatalf("expected resolution and transition in trace")
	}
	var frames []string
	for _, f := range tr.Frames {
		frames = append(frames, f.Type)
	}
	want := "narrative_chunk,narrative_end,narrative_chunk,narrative_end,state_delta"
	if strings.Join(frames, ",") != want {
		t.Fatalf("frames = %v, want %s", frames, want)
	}
	if tr.Frames[0].Count != 1 {
		t.Fatalf("streamed chunks should be folded with a count, got %+v", tr.Frames[0])
	}
	if len(tr.Events) != 2 {
		t.Fatalf("expected the turn's events in the trace, got %+v", tr.Events)
	}
}

func TestHandlerChat_TracesRejectedTurn(t *testing.T) {
	t.Setenv("MUTATIONS_TABLE", "mutations-test")
	conn := db.Connection{ConnectionID: "conn-1", GameID: "game-choice-blocked", UserID: db.BinaryID("user-1")}
	dbFake := &tracingDBClient{fakeDBClient: &fakeDBClient{
		connection: conn,
		user:       &db.UserRecord{UserID: db.BinaryID("user-1"), AIEnabled: true},
		save:       makeAwaitingChoiceCampaignSaveState(t),
	}}
	origNewDBClient, origNewWSSender, origNewAIClient := newDBClient, newWSSender, newAIClient
	newDBClient = func(context.Context) (dbClient, error) { return dbFake, nil }
	newWSSender = func(context.Context) (wsSender, error) { return &fakeWSSender{}, nil }
	newAIClient = func(context.Context) (aiClient, error) { return &fakeAIClient{}, nil }
	t.Cleanup(func() {
		newDBClient = origNewDBClient
		newWSSender = origNewWSSender
		newAIClient = origNewAIClient
	})

	body, _ := json.Marshal(chatRequest{Action: "chat", Content: "I rush in"})
	if _, err := handler(context.Background(), makeWSChatReq("conn-1", string(body))); err != nil {
		t.Fatalf("handler err: %v", err)
	}
	if len(dbFake.traces) != 1 {
		t.Fatalf("expected the rejected turn to be traced, got %d traces", len(dbFake.traces))
	}
	if errs := dbFake.traces[0].Errors; len(errs) != 1 || errs[0] != "sent error: dialogue_choice_required" {
		t.Fatalf("unexpected trace errors %v", errs)
	}
}
