package wsgameaction

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

// fakeActionDB is the whole of DynamoDB the handler sees: one session.
type fakeActionDB struct {
	fakeDialogueChoiceDB
	save game.SaveState
	conn db.Connection
}

func (f *fakeActionDB) GetConnection(context.Context, string) (db.Connection, error) {
	return f.conn, nil
}

func (f *fakeActionDB) GetGame(context.Context, string) (game.SaveState, error) { return f.save, nil }

func (f *fakeActionDB) PutGame(ctx context.Context, s game.SaveState) error {
	f.save = s
	return f.fakeDialogueChoiceDB.PutGame(ctx, s)
}

type fakeActionSender struct {
	fakeDialogueChoiceSender
	states int
}

func (f *fakeActionSender) Send(_ context.Context, _ string, frame wsutil.Frame) error {
	f.frames = append(f.frames, frame)
	return nil
}

func (f *fakeActionSender) SendFullState(context.Context, string, any) error {
	f.states++
	return nil
}

// walkingSession is a Briarwell session past Father's question, standing in
// the cottage, served by fakes (no AWS).
func walkingSession(t *testing.T, def *campaigns.CampaignDefinition) (*fakeActionDB, *fakeActionSender) {
	t.Helper()
	creation, err := def.CharacterCreation.Apply(game.CharacterCreationData{Name: "Pip"})
	if err != nil {
		t.Fatal(err)
	}
	g, history, err := campaigns.StartSession(context.Background(), def, "s-walk", "u-walk", creation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := campaigns.ResumeDialogueWithChoice(def, g, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
		t.Fatal(err)
	}
	conn := db.Connection{ConnectionID: "conn-walk", UserID: db.BinaryID("u-walk"), GameID: "s-walk"}
	store := &fakeActionDB{save: g.ToSaveState(nil, history), conn: conn}
	store.connections = []db.Connection{conn}
	sender := &fakeActionSender{}
	orig, origWS, origReg := newDBClient, newWSSender, getCampaignRegistry
	newDBClient = func(context.Context) (dbClient, error) { return store, nil }
	newWSSender = func(context.Context) (wsSender, error) { return sender, nil }
	getCampaignRegistry = func() (*campaigns.Registry, error) { return campaigns.NewRegistry(def), nil }
	t.Cleanup(func() { newDBClient, newWSSender, getCampaignRegistry = orig, origWS, origReg })
	return store, sender
}

func walk(t *testing.T, room string) {
	t.Helper()
	body, _ := json.Marshal(actionRequest{Action: "game_action", SubAction: "move_to_room", Payload: room})
	if _, err := handler(context.Background(), makeActionReq("conn-walk", string(body))); err != nil {
		t.Fatal(err)
	}
}

func playerRoom(t *testing.T, s game.SaveState) string {
	t.Helper()
	g, err := game.FromSaveState(s)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := g.OwnerCharacter()
	return owner.LocationID
}

func briarwell(t *testing.T) *campaigns.CampaignDefinition {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := reg.Get("briarwell")
	return def
}

func TestMoveToRoom_WalksToAConnectedRoom(t *testing.T) {
	store, sender := walkingSession(t, briarwell(t))
	walk(t, "village_square")
	if got := playerRoom(t, store.save); got != "village_square" {
		t.Fatalf("player in %q", got)
	}
	if len(sender.errors) != 0 || sender.states != 1 {
		t.Fatalf("errors %v, state updates %d", sender.errors, sender.states)
	}
	if !store.save.Campaign.DiscoveredRoomIDs["village_square"] {
		t.Fatal("the room walked into should be discovered")
	}
}

func TestMoveToRoom_RefusesRoomsThatDontConnect(t *testing.T) {
	store, sender := walkingSession(t, briarwell(t))
	walk(t, "old_mill") // the mill is across the square
	if got := playerRoom(t, store.save); got != "family_cottage" {
		t.Fatalf("player moved to %q", got)
	}
	if len(sender.errors) != 1 || !strings.Contains(sender.errors[0], "doesn't lead to") {
		t.Fatalf("errors %v", sender.errors)
	}
}

func TestMoveToRoom_WaitsForAPendingChoice(t *testing.T) {
	def := briarwell(t)
	store, sender := walkingSession(t, def)
	g, _ := game.FromSaveState(store.save)
	g.Campaign.ActiveDialogue = &game.DialogueRuntimeState{AwaitingChoice: true}
	store.save = g.ToSaveState(store.save.Narrative, store.save.ChatHistory)
	walk(t, "village_square")
	if len(sender.errors) != 1 || sender.errors[0] != "dialogue_choice_required" || playerRoom(t, store.save) != "family_cottage" {
		t.Fatalf("errors %v, player in %q", sender.errors, playerRoom(t, store.save))
	}
}

// Walking into a room can start a scene: here the stall opens when the
// player arrives in Market Lane with the deeds done.
func TestMoveToRoom_ArrivingCanStartAScene(t *testing.T) {
	def := *briarwell(t)
	def.StoryNodes = map[string]campaigns.StoryNode{}
	for id, n := range briarwell(t).StoryNodes {
		def.StoryNodes[id] = n
	}
	deeds := def.StoryNodes["village_deeds"]
	deeds.CompletionRules = []campaigns.TransitionRule{{
		Conditions: []campaigns.Condition{{Type: "player_in_room", RoomID: "market_lane"}},
		NextNodeID: "gift_stall",
	}}
	def.StoryNodes["village_deeds"] = deeds
	if err := def.Validate(); err != nil {
		t.Fatal(err)
	}
	store, sender := walkingSession(t, &def)

	walk(t, "village_square")
	if store.save.Campaign.ActiveNodeID != "village_deeds" {
		t.Fatalf("the square shouldn't start anything, node %s", store.save.Campaign.ActiveNodeID)
	}
	walk(t, "market_lane")
	if store.save.Campaign.ActiveNodeID != "gift_stall" {
		t.Fatalf("arriving in Market Lane should open the stall, node %s", store.save.Campaign.ActiveNodeID)
	}
	if d := sender.deltas[len(sender.deltas)-1]; d.Campaign == nil || d.Campaign.ActiveNodeID != "gift_stall" {
		t.Fatalf("the client should get the new scene, got %+v", d.Campaign)
	}
	var streamed strings.Builder
	for _, f := range sender.frames {
		if f.Type == wsutil.FrameNarrativeChunk {
			streamed.WriteString(f.Payload.(map[string]string)["content"])
		}
	}
	if !strings.Contains(streamed.String(), "Garrick") {
		t.Fatalf("the stall's opening should be broadcast, got %q", streamed.String())
	}
	last := store.save.ChatHistory[len(store.save.ChatHistory)-1]
	if !strings.Contains(last.Content, "Garrick") || len(last.Events) == 0 {
		t.Fatalf("the scene should join the log with its events, got %+v", last)
	}
}

// Walking somewhere with no scene still updates the story: an arrival line
// joins the log (not broadcast as narration, so it doesn't hold the player)
// and the game master's history.
func TestMoveToRoom_ArrivalUpdatesTheStory(t *testing.T) {
	store, sender := walkingSession(t, briarwell(t))
	before := len(store.save.Narrative)
	walk(t, "village_square")
	last := store.save.ChatHistory[len(store.save.ChatHistory)-1]
	if last.Type != "arrival" || !strings.HasPrefix(last.Content, "Village Square.") || !strings.Contains(last.Content, "Little Bess is here.") {
		t.Fatalf("arrival line: %+v", last)
	}
	if len(store.save.Narrative) != before+1 || store.save.Narrative[before].Content[0].Text != last.Content {
		t.Fatal("the game master's history should learn where the player went")
	}
	for _, f := range sender.frames {
		if f.Type == wsutil.FrameNarrativeChunk {
			t.Fatalf("an arrival isn't streamed as narration: %+v", f)
		}
	}
}

func TestConversation_LocksWalkingUntilThePlayerLeaves(t *testing.T) {
	store, sender := walkingSession(t, briarwell(t))
	g, _ := game.FromSaveState(store.save)
	g.Campaign.Conversation = &game.ConversationView{With: "father", Name: "Tomas"}
	store.save = g.ToSaveState(store.save.Narrative, store.save.ChatHistory)

	walk(t, "village_square")
	if len(sender.errors) != 1 || sender.errors[0] != "in_conversation" || playerRoom(t, store.save) != "family_cottage" {
		t.Fatalf("errors %v, player in %q", sender.errors, playerRoom(t, store.save))
	}

	body, _ := json.Marshal(actionRequest{Action: "game_action", SubAction: "end_conversation"})
	if _, err := handler(context.Background(), makeActionReq("conn-walk", string(body))); err != nil {
		t.Fatal(err)
	}
	if store.save.Campaign.Conversation != nil {
		t.Fatal("leaving should close the conversation")
	}
	// The client learns it too: the campaign goes out as a delta.
	last := sender.deltas[len(sender.deltas)-1]
	if last.Campaign == nil || last.Campaign.Conversation != nil {
		t.Fatalf("the client should be told the conversation closed, got %+v", last.Campaign)
	}
	walk(t, "village_square")
	if playerRoom(t, store.save) != "village_square" {
		t.Fatal("free roam again: the player can walk")
	}
}
