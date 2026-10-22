package trace

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

type fakeStore struct {
	got []*Turn
	err error
}

func (s *fakeStore) PutTrace(_ context.Context, t *Turn) error {
	s.got = append(s.got, t)
	return s.err
}

func TestChunkFoldsConsecutiveChunks(t *testing.T) {
	tr := &Turn{}
	tr.Chunk("narrative_chunk")
	tr.Chunk("narrative_chunk")
	tr.Frame("narrative_end", "", nil)
	tr.Chunk("narrative_chunk")
	if len(tr.Frames) != 3 || tr.Frames[0].Count != 2 || tr.Frames[2].Count != 1 {
		t.Fatalf("unexpected frames %+v", tr.Frames)
	}
}

func TestSnapCopiesCampaignState(t *testing.T) {
	g := &game.Game{Campaign: &game.CampaignRuntimeState{ActiveNodeID: "intro"}}
	s := Snap(g)
	g.Campaign.ActiveNodeID = "later"
	if !strings.Contains(string(s.Campaign), `"active_node_id":"intro"`) {
		t.Fatalf("snapshot changed with the game: %s", s.Campaign)
	}
}

func TestSaveStoresFinishedTrace(t *testing.T) {
	g := &game.Game{Campaign: &game.CampaignRuntimeState{ActiveNodeID: "intro"}}
	tr := Start(KindChat, "ws-chat", "s1", g)
	g.Campaign.ActiveNodeID = "next"
	store := &fakeStore{}
	Save(context.Background(), store, tr, g)
	if len(store.got) != 1 || !strings.Contains(string(store.got[0].After.Campaign), `"next"`) {
		t.Fatalf("expected a finished trace with the after-state, got %+v", store.got)
	}
	if tr.ExpiresAt() <= tr.StartedAt/1000 {
		t.Fatalf("expiry should be after the start")
	}
}

func TestSaveNeverFailsTheTurn(t *testing.T) {
	g := &game.Game{}
	// Stores without PutTrace, without a table, or failing are all tolerated.
	Save(context.Background(), struct{}{}, Start(KindChat, "ws-chat", "s1", g), g)
	Save(context.Background(), &fakeStore{err: ErrDisabled}, Start(KindChat, "ws-chat", "s1", g), g)
	Save(context.Background(), &fakeStore{err: errors.New("boom")}, Start(KindChat, "ws-chat", "s1", g), g)
}
