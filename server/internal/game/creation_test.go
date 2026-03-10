package game_test

import (
	"context"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// standardAbilityScores returns a valid standard array assignment.
func standardAbilityScores() map[string]int {
	return map[string]int{
		"str": 15, "dex": 14, "con": 13, "int": 12, "wis": 10, "cha": 8,
	}
}

// ── BuildDnDCharacter ─────────────────────────────────────────────────────────

func TestBuildDnDCharacter_Barbarian(t *testing.T) {
	ctx := context.Background()
	char, err := game.BuildDnDCharacter(ctx, game.CharacterCreationData{
		Name:           "Grak",
		RaceID:         "half-orc",
		ClassID:        "barbarian",
		AbilityScores:  standardAbilityScores(),
		SelectedSkills: []string{"athletics", "intimidation"},
	})
	if err != nil {
		t.Fatalf("BuildDnDCharacter: %v", err)
	}
	if char == nil {
		t.Fatal("expected non-nil character")
	}
	if char.GetName() != "Grak" {
		t.Errorf("expected name Grak, got %q", char.GetName())
	}
	if char.GetHitPoints() <= 0 {
		t.Errorf("expected positive HP, got %d", char.GetHitPoints())
	}
	if char.GetMaxHitPoints() <= 0 {
		t.Errorf("expected positive max HP, got %d", char.GetMaxHitPoints())
	}
}

func TestBuildDnDCharacter_Fighter(t *testing.T) {
	ctx := context.Background()
	char, err := game.BuildDnDCharacter(ctx, game.CharacterCreationData{
		Name:           "Ser Alain",
		RaceID:         "human",
		ClassID:        "fighter",
		AbilityScores:  standardAbilityScores(),
		SelectedSkills: []string{"athletics", "perception"},
	})
	if err != nil {
		t.Fatalf("BuildDnDCharacter Fighter: %v", err)
	}
	if char.GetHitPoints() <= 0 {
		t.Errorf("expected positive HP, got %d", char.GetHitPoints())
	}
}

func TestBuildDnDCharacter_Monk(t *testing.T) {
	ctx := context.Background()
	char, err := game.BuildDnDCharacter(ctx, game.CharacterCreationData{
		Name:           "Yuki",
		RaceID:         "elf",
		SubraceID:      "wood-elf",
		ClassID:        "monk",
		AbilityScores:  standardAbilityScores(),
		SelectedSkills: []string{"acrobatics", "stealth"},
	})
	if err != nil {
		t.Fatalf("BuildDnDCharacter Monk: %v", err)
	}
	if char.GetHitPoints() <= 0 {
		t.Errorf("expected positive HP, got %d", char.GetHitPoints())
	}
}

func TestBuildDnDCharacter_InvalidClass(t *testing.T) {
	ctx := context.Background()
	_, err := game.BuildDnDCharacter(ctx, game.CharacterCreationData{
		Name:           "Nobody",
		RaceID:         "human",
		ClassID:        "wizard", // not supported
		AbilityScores:  standardAbilityScores(),
		SelectedSkills: []string{},
	})
	if err == nil {
		t.Error("expected error for unsupported class wizard")
	}
}

func TestBuildDnDCharacter_InvalidRace(t *testing.T) {
	ctx := context.Background()
	_, err := game.BuildDnDCharacter(ctx, game.CharacterCreationData{
		Name:           "Alien",
		RaceID:         "martian",
		ClassID:        "fighter",
		AbilityScores:  standardAbilityScores(),
		SelectedSkills: []string{"athletics", "perception"},
	})
	if err == nil {
		t.Error("expected error for unknown race")
	}
}

func TestBuildDnDCharacter_MissingName(t *testing.T) {
	ctx := context.Background()
	_, err := game.BuildDnDCharacter(ctx, game.CharacterCreationData{
		Name:           "",
		RaceID:         "human",
		ClassID:        "fighter",
		AbilityScores:  standardAbilityScores(),
		SelectedSkills: []string{"athletics", "perception"},
	})
	if err == nil {
		t.Error("expected error for missing character name")
	}
}

// ── Schema migration ──────────────────────────────────────────────────────────

func TestFromSaveState_V1_SetsNeedsReset(t *testing.T) {
	// v1: single Player field, no Players map
	ss := game.SaveState{
		SessionID:     "sess-1",
		UserID:        "user-1",
		SchemaVersion: 1,
		Player: game.Character{
			ID:    "char-1",
			Name:  "Legolas",
			Alive: true,
		},
	}
	g, err := game.FromSaveState(ss)
	if err != nil {
		t.Fatalf("FromSaveState v1: %v", err)
	}
	if !g.NeedsCharacterReset {
		t.Error("expected NeedsCharacterReset=true for v1 migration")
	}
	// Player should have been migrated into the Players map
	c, ok := g.GetPlayerCharacter("user-1")
	if !ok {
		t.Fatal("expected player in Players map after v1 migration")
	}
	if c.Name != "Legolas" {
		t.Errorf("expected name Legolas, got %q", c.Name)
	}
}

func TestFromSaveState_V2_SetsNeedsReset(t *testing.T) {
	ss := game.SaveState{
		SessionID:     "sess-2",
		UserID:        "user-2",
		OwnerID:       "user-2",
		SchemaVersion: 2,
		Players: map[string]game.Character{
			"user-2": {ID: "char-2", Name: "Eowyn", Alive: true},
		},
	}
	g, err := game.FromSaveState(ss)
	if err != nil {
		t.Fatalf("FromSaveState v2: %v", err)
	}
	if !g.NeedsCharacterReset {
		t.Error("expected NeedsCharacterReset=true for v2 migration")
	}
}

func TestFromSaveState_V3_NoReset(t *testing.T) {
	ss := game.SaveState{
		SessionID:     "sess-3",
		UserID:        "user-3",
		OwnerID:       "user-3",
		SchemaVersion: 3,
		Players: map[string]game.Character{
			"user-3": {ID: "char-3", Name: "Aria"},
		},
	}
	g, err := game.FromSaveState(ss)
	if err != nil {
		t.Fatalf("FromSaveState v3: %v", err)
	}
	if g.NeedsCharacterReset {
		t.Error("expected NeedsCharacterReset=false for v3")
	}
}

func TestFromSaveState_FutureVersion_Errors(t *testing.T) {
	ss := game.SaveState{
		SessionID:     "sess-future",
		SchemaVersion: 99,
	}
	_, err := game.FromSaveState(ss)
	if err == nil {
		t.Error("expected error for future schema version")
	}
}

// ── ToSaveState round-trip ────────────────────────────────────────────────────

func TestToSaveState_SchemaVersion(t *testing.T) {
	g := game.NewGame("sess-rt", "user-rt")
	ss := g.ToSaveState(nil, nil)
	if ss.SchemaVersion != game.SchemaVersion {
		t.Errorf("expected schema version %d, got %d", game.SchemaVersion, ss.SchemaVersion)
	}
}

func TestToSaveState_PreservesCreationParams(t *testing.T) {
	g := game.NewGame("sess-cp", "user-cp")
	g.CreationParams = game.CharacterCreationData{
		Name:    "Thorin",
		RaceID:  "dwarf",
		ClassID: "fighter",
	}
	ss := g.ToSaveState(nil, nil)
	if ss.CreationParams.Name != "Thorin" {
		t.Errorf("expected CreationParams.Name=Thorin, got %q", ss.CreationParams.Name)
	}
	if ss.CreationParams.ClassID != "fighter" {
		t.Errorf("expected CreationParams.ClassID=fighter, got %q", ss.CreationParams.ClassID)
	}
}

// ── BuildCharacterContext ────────────────────────────────────────────────────

func TestBuildDnDCharacter_ToDataRoundTrip(t *testing.T) {
	ctx := context.Background()
	char, err := game.BuildDnDCharacter(ctx, game.CharacterCreationData{
		Name:           "Durgin",
		RaceID:         "half-orc",
		ClassID:        "fighter",
		AbilityScores:  standardAbilityScores(),
		SelectedSkills: []string{"athletics", "history"},
	})
	if err != nil {
		t.Fatalf("BuildDnDCharacter: %v", err)
	}
	data := char.ToData()
	if data == nil {
		t.Fatal("expected non-nil ToData()")
	}
	if data.Name != "Durgin" {
		t.Errorf("expected name Durgin in data, got %q", data.Name)
	}
	if string(data.ClassID) != "fighter" {
		t.Errorf("expected class fighter in data, got %q", data.ClassID)
	}
	ctx2 := game.BuildCharacterContext("Durgin", data)
	if ctx2 == "" {
		t.Error("expected non-empty character context")
	}
}
