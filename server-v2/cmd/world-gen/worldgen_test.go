package main

import (
	"context"
	"testing"

	"github.com/KirkDiggler/rpg-toolkit/tools/environments"
	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// ---- generateDungeonLayout ----

func TestGenerateDungeonLayout_RoomCount(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 42, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}
	if len(data.Zones) == 0 {
		t.Error("expected at least 1 room, got 0")
	}
	// We request 8 rooms; the generator may produce slightly fewer due to
	// branching constraints — accept any count >= 3.
	if len(data.Zones) < 3 {
		t.Errorf("expected >= 3 rooms, got %d", len(data.Zones))
	}
}

func TestGenerateDungeonLayout_HasPassages(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 99, "dark cave")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}
	if len(data.Passages) == 0 {
		t.Error("expected at least 1 passage connecting rooms")
	}
}

func TestGenerateDungeonLayout_Deterministic(t *testing.T) {
	ctx := context.Background()
	d1, err := generateDungeonLayout(ctx, 7777, "forest")
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}
	d2, err := generateDungeonLayout(ctx, 7777, "forest")
	if err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if len(d1.Zones) != len(d2.Zones) {
		t.Errorf("deterministic seed produced different room counts: %d vs %d",
			len(d1.Zones), len(d2.Zones))
	}
}

// ---- populateEncounters ----

func TestPopulateEncounters_EntranceEmpty(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 12345, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}

	monsters := populateEncounters(data, 12345)

	for _, zone := range data.Zones {
		if zone.Type == environments.RoomTypeEntrance {
			if ms, ok := monsters[zone.ID]; ok && len(ms) > 0 {
				t.Errorf("entrance room %s should have no monsters, got %d", zone.ID, len(ms))
			}
		}
	}
}

func TestPopulateEncounters_BossRoomPopulated(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 55555, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}

	monsters := populateEncounters(data, 55555)

	for _, zone := range data.Zones {
		if zone.Type == environments.RoomTypeBoss {
			ms, ok := monsters[zone.ID]
			if !ok || len(ms) == 0 {
				t.Errorf("boss room %s should have monsters", zone.ID)
			}
			// Boss room should have 3 monsters: 1 brown bear + 2 ghouls.
			if len(ms) != 3 {
				t.Errorf("boss room expected 3 monsters, got %d", len(ms))
			}
		}
	}
}

// ---- buildDungeonData ----

func TestBuildDungeonData_AllRoomsPresent(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 8888, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}

	framing := ai.NarrativeFraming{
		Title:        "The Cursed Keep",
		Theme:        "Dark gothic fortress",
		QuestGoal:    "Slay the boss",
		OpeningScene: "You stand at the entrance...",
		RoomNames:    make(map[string]string),
	}
	// Give each room a name in the framing.
	for _, z := range data.Zones {
		framing.RoomNames[z.ID] = "Room " + z.ID
	}

	dd := buildDungeonData(data, framing, 8888)

	if dd == nil {
		t.Fatal("buildDungeonData returned nil")
	}
	if len(dd.Rooms) != len(data.Zones) {
		t.Errorf("expected %d rooms in DungeonData, got %d", len(data.Zones), len(dd.Rooms))
	}
	if dd.StartRoomID == "" {
		t.Error("StartRoomID should not be empty")
	}
	if dd.BossRoomID == "" {
		t.Error("BossRoomID should not be empty")
	}
	if dd.State != game.DungeonStateActive {
		t.Errorf("expected DungeonStateActive, got %v", dd.State)
	}
}

func TestBuildDungeonData_StartRoomRevealed(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 11111, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}

	framing := ai.NarrativeFraming{
		Title:     "Test Dungeon",
		RoomNames: map[string]string{},
	}
	dd := buildDungeonData(data, framing, 11111)

	if !dd.RevealedRooms[dd.StartRoomID] {
		t.Error("starting room should be revealed in fog-of-war map")
	}
}

func TestBuildDungeonData_FallbackRoomNames(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 22222, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}

	// Pass framing with empty room_names — should fall back to deterministic names.
	framing := ai.NarrativeFraming{
		Title:     "No Names",
		RoomNames: map[string]string{},
	}
	dd := buildDungeonData(data, framing, 22222)

	for id, room := range dd.Rooms {
		if room.Name == "" {
			t.Errorf("room %s has empty name after fallback", id)
		}
	}
}

// ---- buildLegacyRooms ----

func TestBuildLegacyRooms_PopulatesGameRooms(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 33333, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}
	framing := ai.NarrativeFraming{Title: "Legacy", RoomNames: map[string]string{}}
	dd := buildDungeonData(data, framing, 33333)

	g := game.NewGame("sess-test", "user-test")
	buildLegacyRooms(g, dd)

	if len(g.Rooms) != len(dd.Rooms) {
		t.Errorf("expected %d legacy rooms, got %d", len(dd.Rooms), len(g.Rooms))
	}
	for id := range dd.Rooms {
		if _, err := g.GetRoom(id); err != nil {
			t.Errorf("room %s missing from legacy map: %v", id, err)
		}
	}
}

// ---- mapRoomType / fallbackRoomName ----

func TestMapRoomType(t *testing.T) {
	cases := []struct {
		input    string
		expected game.DungeonRoomType
	}{
		{environments.RoomTypeEntrance, game.DungeonRoomTypeEntrance},
		{environments.RoomTypeBoss, game.DungeonRoomTypeBoss},
		{environments.RoomTypeTreasure, game.DungeonRoomTypeTreasure},
		{environments.RoomTypeCorridor, game.DungeonRoomTypeCorridor},
		{environments.RoomTypeJunction, game.DungeonRoomTypeJunction},
		{environments.RoomTypeChamber, game.DungeonRoomTypeChamber},
		{"unknown_type", game.DungeonRoomTypeChamber},
	}
	for _, c := range cases {
		got := mapRoomType(c.input)
		if got != c.expected {
			t.Errorf("mapRoomType(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestFallbackRoomName(t *testing.T) {
	names := []string{
		fallbackRoomName(environments.RoomTypeEntrance),
		fallbackRoomName(environments.RoomTypeBoss),
		fallbackRoomName(environments.RoomTypeTreasure),
		fallbackRoomName(environments.RoomTypeCorridor),
		fallbackRoomName(environments.RoomTypeChamber),
		fallbackRoomName("unknown"),
	}
	for _, n := range names {
		if n == "" {
			t.Error("fallbackRoomName returned empty string")
		}
	}
}

// ---- DungeonData round-trip through SaveState ----

func TestDungeonData_SaveStateRoundTrip(t *testing.T) {
	ctx := context.Background()
	data, err := generateDungeonLayout(ctx, 44444, "")
	if err != nil {
		t.Fatalf("generateDungeonLayout: %v", err)
	}
	framing := ai.NarrativeFraming{Title: "Round Trip", RoomNames: map[string]string{}}
	dd := buildDungeonData(data, framing, 44444)

	g := game.NewGame("sess-rt", "user-rt")
	g.DungeonData = dd
	buildLegacyRooms(g, dd)

	saved := g.ToSaveState(nil, nil)
	if saved.DungeonData == nil {
		t.Fatal("DungeonData missing from SaveState")
	}
	if saved.DungeonData.StartRoomID != dd.StartRoomID {
		t.Errorf("StartRoomID mismatch after round-trip: %q vs %q",
			saved.DungeonData.StartRoomID, dd.StartRoomID)
	}
	if saved.SchemaVersion != game.SchemaVersion {
		t.Errorf("SchemaVersion should be %d, got %d", game.SchemaVersion, saved.SchemaVersion)
	}

	restored, err := game.FromSaveState(saved)
	if err != nil {
		t.Fatalf("FromSaveState: %v", err)
	}
	if restored.DungeonData == nil {
		t.Fatal("DungeonData missing after FromSaveState")
	}
	if restored.DungeonData.BossRoomID != dd.BossRoomID {
		t.Errorf("BossRoomID mismatch after FromSaveState: %q vs %q",
			restored.DungeonData.BossRoomID, dd.BossRoomID)
	}
}
