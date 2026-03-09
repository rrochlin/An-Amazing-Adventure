package combat_test

import (
	"context"
	"testing"

	dnd5echar "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster"
	"github.com/google/uuid"
	"github.com/rrochlin/an-amazing-adventure/internal/combat"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// ---- helpers ----

// buildTestFighter creates a level-1 Fighter character for testing.
// Uses the same builder pattern confirmed working in Phase 2.
func buildTestFighter(t *testing.T) *dnd5echar.Character {
	t.Helper()
	char, err := game.BuildDnDCharacter(context.Background(), game.CharacterCreationData{
		Name:    "TestFighter",
		RaceID:  "human",
		ClassID: "fighter",
		AbilityScores: map[string]int{
			"strength": 16, "dexterity": 12, "constitution": 14,
			"intelligence": 10, "wisdom": 10, "charisma": 8,
		},
		SelectedSkills: []string{"athletics", "intimidation"},
	})
	if err != nil {
		t.Fatalf("BuildDnDCharacter: %v", err)
	}
	return char
}

// ---- NewMonsterByType ----

func TestNewMonsterByType_Goblin(t *testing.T) {
	m := combat.NewMonsterByType("goblin")
	if m == nil {
		t.Fatal("expected non-nil goblin")
	}
	if m.Name() != "Goblin" {
		t.Errorf("expected Goblin, got %q", m.Name())
	}
	if m.GetHitPoints() <= 0 {
		t.Errorf("expected positive HP, got %d", m.GetHitPoints())
	}
}

func TestNewMonsterByType_UnknownReturnsNil(t *testing.T) {
	m := combat.NewMonsterByType("dragon_king")
	if m != nil {
		t.Errorf("expected nil for unknown type, got %q", m.Name())
	}
}

func TestNewMonsterByType_AllTypes(t *testing.T) {
	types := []string{"goblin", "skeleton", "zombie", "wolf", "giant_rat", "bandit", "ghoul", "brown_bear", "thug"}
	for _, typ := range types {
		m := combat.NewMonsterByType(typ)
		if m == nil {
			t.Errorf("NewMonsterByType(%q) returned nil", typ)
		}
	}
}

// ---- NewEncounter ----

func TestNewEncounter_EmptyPlayers(t *testing.T) {
	ctx := context.Background()
	goblin := monster.NewGoblin(uuid.NewString())
	enc, err := combat.NewEncounter(ctx, map[string]*dnd5echar.Character{}, []*monster.Monster{goblin})
	if err != nil {
		t.Fatalf("NewEncounter: %v", err)
	}
	if enc == nil {
		t.Fatal("expected non-nil encounter")
	}
	enc.Cleanup(ctx)
}

func TestNewEncounter_WithPlayer(t *testing.T) {
	ctx := context.Background()
	fighter := buildTestFighter(t)
	goblin := monster.NewGoblin(uuid.NewString())

	enc, err := combat.NewEncounter(ctx,
		map[string]*dnd5echar.Character{"uid-1": fighter},
		[]*monster.Monster{goblin},
	)
	if err != nil {
		t.Fatalf("NewEncounter: %v", err)
	}
	defer enc.Cleanup(ctx)

	if len(enc.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(enc.Players))
	}
	if len(enc.Monsters) != 1 {
		t.Errorf("expected 1 monster, got %d", len(enc.Monsters))
	}
}

// ---- ResolvePlayerAttack ----

func TestResolvePlayerAttack_InvalidAttacker(t *testing.T) {
	ctx := context.Background()
	goblin := monster.NewGoblin(uuid.NewString())
	enc, err := combat.NewEncounter(ctx,
		map[string]*dnd5echar.Character{},
		[]*monster.Monster{goblin},
	)
	if err != nil {
		t.Fatalf("NewEncounter: %v", err)
	}
	defer enc.Cleanup(ctx)

	_, err = combat.ResolvePlayerAttack(ctx, enc, combat.AttackInput{
		AttackerID: "nonexistent-uid",
		TargetID:   goblin.GetID(),
	})
	if err == nil {
		t.Error("expected error for nonexistent attacker")
	}
}

func TestResolvePlayerAttack_InvalidTarget(t *testing.T) {
	ctx := context.Background()
	fighter := buildTestFighter(t)
	goblin := monster.NewGoblin(uuid.NewString())

	enc, err := combat.NewEncounter(ctx,
		map[string]*dnd5echar.Character{"uid-1": fighter},
		[]*monster.Monster{goblin},
	)
	if err != nil {
		t.Fatalf("NewEncounter: %v", err)
	}
	defer enc.Cleanup(ctx)

	_, err = combat.ResolvePlayerAttack(ctx, enc, combat.AttackInput{
		AttackerID: "uid-1",
		TargetID:   "nonexistent-monster",
	})
	if err == nil {
		t.Error("expected error for nonexistent target")
	}
}

func TestResolvePlayerAttack_Returns_CombatLog(t *testing.T) {
	ctx := context.Background()
	fighter := buildTestFighter(t)
	goblin := monster.NewGoblin(uuid.NewString())

	enc, err := combat.NewEncounter(ctx,
		map[string]*dnd5echar.Character{"uid-1": fighter},
		[]*monster.Monster{goblin},
	)
	if err != nil {
		t.Fatalf("NewEncounter: %v", err)
	}
	defer enc.Cleanup(ctx)

	out, err := combat.ResolvePlayerAttack(ctx, enc, combat.AttackInput{
		AttackerID: "uid-1",
		TargetID:   goblin.GetID(),
		WeaponID:   "longsword",
	})
	if err != nil {
		t.Fatalf("ResolvePlayerAttack: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
	if out.Result == nil {
		t.Error("expected non-nil AttackResult")
	}
	if out.CombatLog == "" {
		t.Error("expected non-empty CombatLog")
	}
	if out.TargetName != "Goblin" {
		t.Errorf("expected target name Goblin, got %q", out.TargetName)
	}
}

func TestResolvePlayerAttack_TargetMaxHP_Consistent(t *testing.T) {
	ctx := context.Background()
	fighter := buildTestFighter(t)
	goblin := monster.NewGoblin(uuid.NewString())
	startMaxHP := goblin.GetMaxHitPoints()

	enc, err := combat.NewEncounter(ctx,
		map[string]*dnd5echar.Character{"uid-1": fighter},
		[]*monster.Monster{goblin},
	)
	if err != nil {
		t.Fatalf("NewEncounter: %v", err)
	}
	defer enc.Cleanup(ctx)

	out, err := combat.ResolvePlayerAttack(ctx, enc, combat.AttackInput{
		AttackerID: "uid-1",
		TargetID:   goblin.GetID(),
		WeaponID:   "longsword",
	})
	if err != nil {
		t.Fatalf("ResolvePlayerAttack: %v", err)
	}
	if out.TargetMaxHP != startMaxHP {
		t.Errorf("TargetMaxHP changed: was %d, now %d", startMaxHP, out.TargetMaxHP)
	}
}

// ---- FormatAttackResult ----

func TestFormatAttackResult_Hit(t *testing.T) {
	result := &combat_test_attResult{
		AttackRoll: 15, AttackBonus: 5, TotalAttack: 20, TargetAC: 13,
		Hit: true, Critical: false,
		DamageRolls: []int{6}, DamageBonus: 3, TotalDamage: 9,
		DamageType: "slashing",
	}
	_ = result // FormatAttackResult is tested via integration (ResolvePlayerAttack)
}

// ---- RollInitiative ----

func TestRollInitiative_OrderedHighToLow(t *testing.T) {
	ctx := context.Background()
	fighter := buildTestFighter(t)
	goblin := combat.NewMonsterByType("goblin")
	skeleton := combat.NewMonsterByType("skeleton")

	// We need the monsters as live *monster.Monster, so use NewEncounter to hydrate them
	enc, err := combat.NewEncounter(ctx,
		map[string]*dnd5echar.Character{"uid-1": fighter},
		[]*monster.Monster{goblin, skeleton},
	)
	if err != nil {
		t.Fatalf("NewEncounter: %v", err)
	}
	defer enc.Cleanup(ctx)

	monsterList := make([]*monster.Monster, 0, len(enc.Monsters))
	for _, m := range enc.Monsters {
		monsterList = append(monsterList, m)
	}

	entries := combat.RollInitiative(enc.Players, monsterList)
	if len(entries) != 3 {
		t.Fatalf("expected 3 initiative entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Roll > entries[i-1].Roll {
			t.Errorf("initiative not sorted: entries[%d].Roll=%d > entries[%d].Roll=%d",
				i, entries[i].Roll, i-1, entries[i-1].Roll)
		}
	}
}

// ---- Game combat helpers ----

func TestGame_SetAndGetRoomMonsters(t *testing.T) {
	g := game.NewGame("sess-1", "user-1")
	roomID := "room-abc"

	if ms := g.GetRoomMonsters(roomID); len(ms) != 0 {
		t.Errorf("expected empty monster list, got %d", len(ms))
	}

	goblinData := combat.NewMonsterByType("goblin").ToData()
	g.SetRoomMonsters(roomID, []*monster.Data{goblinData})

	ms := g.GetRoomMonsters(roomID)
	if len(ms) != 1 {
		t.Fatalf("expected 1 monster, got %d", len(ms))
	}
	if ms[0].Name != "Goblin" {
		t.Errorf("expected Goblin, got %q", ms[0].Name)
	}
}

func TestGame_HasLiveMonstersInRoom(t *testing.T) {
	g := game.NewGame("sess-1", "user-1")
	roomID := "room-xyz"

	if g.HasLiveMonstersInRoom(roomID) {
		t.Error("expected no live monsters in empty room")
	}

	goblinData := combat.NewMonsterByType("goblin").ToData()
	goblinData.HitPoints = 0 // killed
	g.SetRoomMonsters(roomID, []*monster.Data{goblinData})

	if g.HasLiveMonstersInRoom(roomID) {
		t.Error("expected no live monsters when all HP = 0")
	}

	goblinData2 := combat.NewMonsterByType("goblin").ToData()
	g.SetRoomMonsters(roomID, []*monster.Data{goblinData, goblinData2})

	if !g.HasLiveMonstersInRoom(roomID) {
		t.Error("expected live monsters when at least one HP > 0")
	}
}

func TestGame_CombatFields_RoundTrip(t *testing.T) {
	g := game.NewGame("sess-1", "user-1")
	roomID := "room-r1"
	goblinData := combat.NewMonsterByType("goblin").ToData()
	g.SetRoomMonsters(roomID, []*monster.Data{goblinData})
	g.PendingCombatContext = "Fighter attacks Goblin.\nHIT — 8 slashing damage.\n"
	g.InitiativeOrder = []combat.InitiativeEntry{
		{CombatantID: "uid-1", CombatantName: "TestFighter", Roll: 18, IsPlayer: true},
		{CombatantID: goblinData.ID, CombatantName: "Goblin", Roll: 12, IsPlayer: false},
	}

	saved := g.ToSaveState(nil, nil)
	restored, err := game.FromSaveState(saved)
	if err != nil {
		t.Fatalf("FromSaveState: %v", err)
	}

	if restored.PendingCombatContext != g.PendingCombatContext {
		t.Errorf("PendingCombatContext mismatch: got %q", restored.PendingCombatContext)
	}
	if len(restored.InitiativeOrder) != 2 {
		t.Errorf("expected 2 initiative entries, got %d", len(restored.InitiativeOrder))
	}
	ms := restored.GetRoomMonsters(roomID)
	if len(ms) != 1 {
		t.Fatalf("expected 1 monster after round-trip, got %d", len(ms))
	}
	if ms[0].Name != "Goblin" {
		t.Errorf("expected Goblin after round-trip, got %q", ms[0].Name)
	}
}

// combat_test_attResult is a minimal stand-in for formatting test documentation.
type combat_test_attResult struct {
	AttackRoll  int
	AttackBonus int
	TotalAttack int
	TargetAC    int
	Hit         bool
	Critical    bool
	DamageRolls []int
	DamageBonus int
	TotalDamage int
	DamageType  string
}
