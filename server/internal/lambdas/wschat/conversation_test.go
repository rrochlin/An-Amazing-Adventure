package wschat

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

// A Briarwell session past Father's question (no choice open), in the cottage
// with the family.
func roamingSave(t *testing.T) game.SaveState {
	t.Helper()
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, _ := reg.Get("briarwell")
	g, history, err := campaigns.BootstrapGame(def, "game-roam", "user-1", game.NewCharacter("Pip", ""), game.CharacterCreationData{Name: "Pip"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := campaigns.ResumeDialogueWithChoice(def, g, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := campaigns.OrchestrateTurn(def, g, campaigns.ConditionResolution{}); err != nil {
		t.Fatal(err)
	}
	return g.ToSaveState(historyToNarrative(history), history)
}

func talk(t *testing.T, save game.SaveState, req chatRequest, gmDecision ...string) (*fakeDBClient, *fakeWSSender, *fakeAIClient) {
	t.Helper()
	t.Setenv("MUTATIONS_TABLE", "mutations-test")
	conn := db.Connection{ConnectionID: "conn-1", GameID: save.SessionID, UserID: db.BinaryID("user-1")}
	dbFake := &fakeDBClient{connection: conn, connectionsByID: []db.Connection{conn}, user: &db.UserRecord{UserID: db.BinaryID("user-1"), AIEnabled: true}, save: save}
	wsFake, aiFake := &fakeWSSender{}, &fakeAIClient{}
	if len(gmDecision) > 0 {
		aiFake.gmDecision = gmDecision[0]
	}
	orig, origWS, origAI := newDBClient, newWSSender, newAIClient
	newDBClient = func(context.Context) (dbClient, error) { return dbFake, nil }
	newWSSender = func(context.Context) (wsSender, error) { return wsFake, nil }
	newAIClient = func(context.Context) (aiClient, error) { return aiFake, nil }
	t.Cleanup(func() { newDBClient, newWSSender, newAIClient = orig, origWS, origAI })
	body, _ := json.Marshal(req)
	if _, err := handler(context.Background(), makeWSChatReq("conn-1", string(body))); err != nil {
		t.Fatal(err)
	}
	return dbFake, wsFake, aiFake
}

func TestTalkTo_OpensAConversationTheGameMasterKnowsAbout(t *testing.T) {
	dbFake, wsFake, aiFake := talk(t, roamingSave(t), chatRequest{Action: "chat", TalkTo: "father"})
	if len(wsFake.errors) != 0 {
		t.Fatalf("errors %v", wsFake.errors)
	}
	c := dbFake.putGameState.Campaign.Conversation
	if c == nil || c.With != "father" || c.Name != "Tomas" {
		t.Fatalf("conversation %+v", c)
	}
	if len(aiFake.gmRequests) != 1 {
		t.Fatalf("game master calls %d", len(aiFake.gmRequests))
	}
	req := aiFake.gmRequests[0]
	if !strings.Contains(req.PlayerInput, "Tomas") || !strings.Contains(req.System, "in a conversation with Tomas") {
		t.Fatalf("input %q; brief should name the conversation", req.PlayerInput)
	}
}

func TestTalkTo_OnlySomeoneInTheRoom(t *testing.T) {
	dbFake, wsFake, aiFake := talk(t, roamingSave(t), chatRequest{Action: "chat", TalkTo: "bess"}) // Bess is in the square
	if len(wsFake.errors) != 1 || wsFake.errors[0] != "not_here" || len(aiFake.gmRequests) != 0 {
		t.Fatalf("errors %v, model calls %d", wsFake.errors, len(aiFake.gmRequests))
	}
	if dbFake.putGameState.Campaign != nil && dbFake.putGameState.Campaign.Conversation != nil {
		t.Fatal("no conversation should open")
	}
}

// Playtest 7: Dunstan walked off mid-conversation and the text box still
// spoke to him.
func TestConversation_EndsWhenTheOtherPersonLeaves(t *testing.T) {
	leaves := `{"world_events":[{"type":"move_character","input":{"character_name":"Tomas","room_name":"Village Square"}}],"beats":{}}`
	dbFake, wsFake, _ := talk(t, roamingSave(t), chatRequest{Action: "chat", TalkTo: "father"}, leaves)
	if len(wsFake.errors) != 0 {
		t.Fatalf("errors %v", wsFake.errors)
	}
	if c := dbFake.putGameState.Campaign.Conversation; c != nil {
		t.Fatalf("Tomas left, so the conversation should end, got %+v", c)
	}
}

// joiningDB is a game whose player reloads mid-turn: a second connection
// appears after the turn has started streaming.
type joiningDB struct {
	*fakeDBClient
	calls int
}

func (j *joiningDB) GetConnectionsByGameID(ctx context.Context, id string) ([]db.Connection, error) {
	j.calls++
	conns, err := j.fakeDBClient.GetConnectionsByGameID(ctx, id)
	if j.calls > 2 { // the first lookups happen before streaming
		conns = append(conns, db.Connection{ConnectionID: "conn-reloaded", GameID: id, UserID: db.BinaryID("user-1")})
	}
	return conns, err
}

type recordingWS struct {
	*fakeWSSender
	sent map[string][]wsutil.Frame
}

func (r *recordingWS) Send(_ context.Context, id string, f wsutil.Frame) error {
	r.sent[id] = append(r.sent[id], f)
	return nil
}

// Playtest 8: reloading during a turn left the page without that turn's
// narration (it was streamed to the old connection only).
func TestReloadMidTurn_TheNewConnectionGetsTheStory(t *testing.T) {
	t.Setenv("MUTATIONS_TABLE", "mutations-test")
	save := roamingSave(t)
	conn := db.Connection{ConnectionID: "conn-1", GameID: save.SessionID, UserID: db.BinaryID("user-1")}
	dbFake := &joiningDB{fakeDBClient: &fakeDBClient{connection: conn, connectionsByID: []db.Connection{conn}, user: &db.UserRecord{UserID: db.BinaryID("user-1"), AIEnabled: true}, save: save}}
	wsFake := &recordingWS{fakeWSSender: &fakeWSSender{}, sent: map[string][]wsutil.Frame{}}
	aiFake := &fakeAIClient{}
	orig, origWS, origAI := newDBClient, newWSSender, newAIClient
	newDBClient = func(context.Context) (dbClient, error) { return dbFake, nil }
	newWSSender = func(context.Context) (wsSender, error) { return wsFake, nil }
	newAIClient = func(context.Context) (aiClient, error) { return aiFake, nil }
	t.Cleanup(func() { newDBClient, newWSSender, newAIClient = orig, origWS, origAI })

	body, _ := json.Marshal(chatRequest{Action: "chat", Content: "I look around"})
	if _, err := handler(context.Background(), makeWSChatReq("conn-1", string(body))); err != nil {
		t.Fatal(err)
	}
	frames := wsFake.sent["conn-reloaded"]
	if len(frames) != 1 || frames[0].Type != wsutil.FrameGameStateUpdate {
		t.Fatalf("the reloaded page should get the whole state, got %+v", frames)
	}
	view := frames[0].Payload.(game.GameStateView)
	last := view.ChatHistory[len(view.ChatHistory)-1]
	if last.Content != "Tomas looks up from the table." {
		t.Fatalf("with the turn's narration in it, got %+v", last)
	}
	if len(wsFake.sent["conn-1"]) != 0 {
		t.Fatal("the connection that saw the stream needs no full state")
	}
}
