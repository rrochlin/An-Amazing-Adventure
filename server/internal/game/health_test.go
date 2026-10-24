package game

import (
	"context"
	"testing"
)

func quickStartGame(t *testing.T) *Game {
	t.Helper()
	g := NewGame("s", "u")
	g.SetPlayerCharacter("u", NewCharacter("Pip", ""))
	dnd, err := BuildDnDCharacter(context.Background(), CharacterCreationData{
		Name: "Pip", RaceID: "human", ClassID: "fighter",
		AbilityScores:  map[string]int{"str": 13, "dex": 15, "con": 14, "int": 10, "wis": 12, "cha": 8},
		SelectedSkills: []string{"athletics", "perception"},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.SetDnDCharacter("u", dnd)
	return g
}

// A save opened without loading its characters (http-invites does this)
// and saved again must keep them.
func TestSaveKeepsCharactersItDidNotLoad(t *testing.T) {
	g := quickStartGame(t)
	if _, _, err := g.DamagePlayer(context.Background(), "u", 2); err != nil {
		t.Fatal(err)
	}
	saved := g.ToSaveState(nil, nil)
	reopened, err := FromSaveState(saved)
	if err != nil {
		t.Fatal(err)
	}
	reopened.InviteCode = "ABC123"
	again := reopened.ToSaveState(nil, nil)
	data := again.PlayersData["u"]
	if data == nil || data.HitPoints != saved.PlayersData["u"].HitPoints {
		t.Fatalf("the character should survive a save without loading it, got %+v", data)
	}
}

func TestPlayerHealthIsTheCharactersHitPoints(t *testing.T) {
	g := quickStartGame(t)
	hp, max := g.PlayerHitPoints("u")
	if max < 1 || hp != max {
		t.Fatalf("hit points %d/%d", hp, max)
	}
	hp, _, _ = g.DamagePlayer(context.Background(), "u", max+5)
	if hp != 0 || g.Players["u"].Alive || !g.PartyWiped() {
		t.Fatalf("at 0 hit points the player is down: hp %d alive %v", hp, g.Players["u"].Alive)
	}
	hp, _, _ = g.HealPlayer(context.Background(), "u", 3)
	if hp != 3 || !g.Players["u"].Alive {
		t.Fatalf("healed to %d, alive %v", hp, g.Players["u"].Alive)
	}
}
