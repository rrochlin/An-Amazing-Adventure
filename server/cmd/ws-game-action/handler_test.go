package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

func assertPanicsWithEnvAbsent(t *testing.T, envVar string, fn func()) {
	t.Helper()
	t.Setenv(envVar, "")
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic for missing %s, but handler did not panic", envVar)
			return
		}
		msg := ""
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(msg, envVar) {
			t.Errorf("panic message %q does not mention %s", msg, envVar)
		}
	}()
	fn()
}

func makeActionReq(connID, body string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		Body: body,
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connID,
		},
	}
}

func TestHandlerAction_InvalidJSON(t *testing.T) {
	req := makeActionReq("conn-1", "bad-json")
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestHandlerAction_ValidBody_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")

	body, _ := json.Marshal(actionRequest{
		Action:    "game_action",
		SubAction: "move",
		Payload:   "north",
	})
	req := makeActionReq("conn-abc", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	// Should fail at DB layer (410 Gone or 500), not at parse layer (400)
	if resp.StatusCode == 400 {
		t.Errorf("routing/parse failure — expected to reach DB layer, got 400")
	}
}

func TestActionRequest_SubActions(t *testing.T) {
	cases := []struct {
		subAction string
		payload   string
	}{
		{"move", "north"},
		{"pick_up", "Rusty Dagger"},
		{"drop", "Heavy Shield"},
		{"equip", "Iron Helm"},
		{"unequip", "head"},
	}
	for _, c := range cases {
		body, _ := json.Marshal(actionRequest{
			Action:    "game_action",
			SubAction: c.subAction,
			Payload:   c.payload,
		})
		var parsed actionRequest
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("failed to parse action %q: %v", c.subAction, err)
		}
		if parsed.SubAction != c.subAction {
			t.Errorf("expected sub_action=%q, got %q", c.subAction, parsed.SubAction)
		}
		if parsed.Payload != c.payload {
			t.Errorf("expected payload=%q, got %q", c.payload, parsed.Payload)
		}
	}
}

// ---- Required env var tests ----
// ws-game-action calls GetConnection first, so CONNECTIONS_TABLE panics immediately.
// SESSIONS_TABLE is required later (GetGame) but unreachable without real DynamoDB.
// Both vars are set in Terraform — see modules/lambdas/main.tf.

func TestHandlerAction_MissingCONNECTIONS_TABLE_Panics(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")
	body, _ := json.Marshal(actionRequest{Action: "game_action", SubAction: "move", Payload: "north"})
	assertPanicsWithEnvAbsent(t, "CONNECTIONS_TABLE", func() {
		handler(context.Background(), makeActionReq("conn-1", string(body))) //nolint:errcheck
	})
}

func TestHandlerAction_Equip_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")

	body, _ := json.Marshal(actionRequest{
		Action:    "game_action",
		SubAction: "equip",
		Payload:   "Iron Helm",
	})
	req := makeActionReq("conn-equip", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode == 400 {
		t.Errorf("parse failure — expected to reach DB layer, got 400")
	}
}

func TestHandlerAction_Unequip_ReachesDB(t *testing.T) {
	t.Setenv("CONNECTIONS_TABLE", "test-connections")
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("WEBSOCKET_API_ENDPOINT", "https://test.execute-api.us-west-2.amazonaws.com/prod")

	body, _ := json.Marshal(actionRequest{
		Action:    "game_action",
		SubAction: "unequip",
		Payload:   "head",
	})
	req := makeActionReq("conn-unequip", string(body))
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected lambda error: %v", err)
	}
	if resp.StatusCode == 400 {
		t.Errorf("parse failure — expected to reach DB layer, got 400")
	}
}

type fakeDialogueChoiceDB struct {
	connections []db.Connection
	putGame     game.SaveState
}

func (f *fakeDialogueChoiceDB) GetConnectionsByGameID(context.Context, string) ([]db.Connection, error) {
	if len(f.connections) == 0 {
		return nil, nil
	}
	return f.connections, nil
}

func (f *fakeDialogueChoiceDB) DeleteConnection(context.Context, string) error { return nil }

func (f *fakeDialogueChoiceDB) PutGame(_ context.Context, save game.SaveState) error {
	f.putGame = save
	return nil
}

type fakeDialogueChoiceSender struct {
	errors     []string
	frames     []wsutil.Frame
	deltaCount int
}

func (f *fakeDialogueChoiceSender) SendError(_ context.Context, _ string, message string) error {
	f.errors = append(f.errors, message)
	return nil
}

func (f *fakeDialogueChoiceSender) Broadcast(_ context.Context, _ []string, frame wsutil.Frame) ([]string, error) {
	f.frames = append(f.frames, frame)
	return nil, nil
}

func (f *fakeDialogueChoiceSender) SendDelta(context.Context, string, any) error {
	f.deltaCount++
	return nil
}

func makeDialogueChoiceGame(t *testing.T, awaiting bool) (*game.Game, *campaigns.CampaignDefinition, db.Connection, game.SaveState) {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load campaigns: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	g := game.NewGame("game-dialogue-choice", "user-1")
	g.SetPlayerCharacter("user-1", game.NewCharacter("Hero", ""))
	room := game.NewArea("Camp", "Camp")
	if err := g.AddRoom(room); err != nil {
		t.Fatalf("add room: %v", err)
	}
	if err := g.PlacePlayer(room.ID); err != nil {
		t.Fatalf("place player: %v", err)
	}
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      "test",
		CampaignVersion: "1",
		ActiveNodeID:    "route_choice",
		ActiveDialogue: &game.DialogueRuntimeState{
			AssetID:        "route_dialogue",
			CurrentNode:    "RouteChoice",
			AwaitingChoice: awaiting,
			PendingChoices: []game.DialogueChoice{
				{ID: 0, Text: "Take the Hidden Route."},
				{ID: 1, Text: "Take the Target Route."},
				{ID: 2, Text: "Take the Novice Route."},
			},
		},
		Labels:          map[string]string{},
		BoolFlags:       map[string]bool{},
		Counters:        map[string]int{},
		ActorStates:     map[string]game.ActorRuntimeState{},
		CompanionStates: map[string]game.CompanionState{},
	}
	conn := db.Connection{ConnectionID: "conn-1", GameID: g.ID, UserID: db.BinaryID("user-1")}
	save := g.ToSaveState(nil, nil)
	return g, def, conn, save
}

func TestHandleDialogueChoice_Success(t *testing.T) {
	g, _, conn, save := makeDialogueChoiceGame(t, true)
	dbFake := &fakeDialogueChoiceDB{connections: []db.Connection{conn}}
	wsFake := &fakeDialogueChoiceSender{}

	resp, err := handleDialogueChoice(context.Background(), dbFake, wsFake, conn, g, &save, "0")
	if err != nil {
		t.Fatalf("handleDialogueChoice: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if len(wsFake.errors) != 0 {
		t.Fatalf("expected no ws errors, got %#v", wsFake.errors)
	}
	if g.Campaign.ActiveNodeID != "hidden_probe" {
		t.Fatalf("expected campaign to advance to hidden_probe, got %q", g.Campaign.ActiveNodeID)
	}
	if g.Campaign.ActiveDialogue != nil {
		t.Fatalf("expected active dialogue to be cleared after hidden_probe transition, got %#v", g.Campaign.ActiveDialogue)
	}
	if g.Campaign.Labels["entry_route"] != "hidden" {
		t.Fatalf("expected dialogue command to set hidden route, got %#v", g.Campaign.Labels)
	}
	if dbFake.putGame.SessionID == "" {
		t.Fatal("expected updated game state to be persisted")
	}
	if wsFake.deltaCount == 0 {
		t.Fatal("expected at least one state delta")
	}
}

func TestHandleDialogueChoice_ErrorPaths(t *testing.T) {
	g, _, conn, save := makeDialogueChoiceGame(t, true)
	dbFake := &fakeDialogueChoiceDB{connections: []db.Connection{conn}}
	wsFake := &fakeDialogueChoiceSender{}

	resp, err := handleDialogueChoice(context.Background(), dbFake, wsFake, conn, g, &save, "not-a-number")
	if err != nil {
		t.Fatalf("handleDialogueChoice invalid payload: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200 for invalid payload, got %d", resp.StatusCode)
	}
	if len(wsFake.errors) == 0 || wsFake.errors[0] != "dialogue_choice: invalid choice ID" {
		t.Fatalf("expected invalid choice ID error, got %#v", wsFake.errors)
	}

	wsFake.errors = nil
	g.Campaign.ActiveDialogue.AwaitingChoice = false
	resp, err = handleDialogueChoice(context.Background(), dbFake, wsFake, conn, g, &save, "0")
	if err != nil {
		t.Fatalf("handleDialogueChoice not awaiting: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200 for not-awaiting case, got %d", resp.StatusCode)
	}
	if len(wsFake.errors) == 0 || wsFake.errors[0] != "dialogue_choice: not awaiting a choice" {
		t.Fatalf("expected not-awaiting error, got %#v", wsFake.errors)
	}
}
