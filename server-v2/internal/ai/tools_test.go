package ai_test

import (
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// newTestGameWithRooms sets up a minimal game with two connected rooms and a player.
func newTestGameWithRooms(t *testing.T) (*game.Game, string, string) {
	t.Helper()
	g := game.NewGame("test-session", "test-user")
	g.Player = game.NewCharacter("Hero", "The protagonist")

	start := game.NewArea("Tavern", "A smoky tavern")
	north := game.NewArea("Alley", "A dark alley")
	if err := g.AddRoom(start); err != nil {
		t.Fatal(err)
	}
	if err := g.AddRoom(north); err != nil {
		t.Fatal(err)
	}
	if err := g.ConnectRooms(start.ID, north.ID, "north"); err != nil {
		t.Fatal(err)
	}
	if err := g.PlacePlayer(start.ID); err != nil {
		t.Fatal(err)
	}
	return g, start.ID, north.ID
}

func dispatch(g *game.Game, name string, args map[string]any) (string, error) {
	return ai.DispatchTool(g, name, args)
}

func TestDispatchCreateRoom(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)

	result, err := dispatch(g, "create_room", map[string]any{
		"name":                 "Cellar",
		"description":          "A damp cellar",
		"connect_to_room_name": "Tavern",
		"direction":            "down",
	})
	if err != nil {
		t.Fatalf("create_room: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	if _, err := g.GetRoomByName("Cellar"); err != nil {
		t.Error("expected Cellar room to exist after create_room")
	}

	// Bidirectional connection
	tavern, _ := g.GetRoomByName("Tavern")
	if tavern.Connections["down"] == "" {
		t.Error("expected down connection from Tavern to Cellar")
	}
}

func TestDispatchCreateRoomMissingConnect(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	// Omitting connect_to_room_name — room should still be created without connection
	_, err := dispatch(g, "create_room", map[string]any{
		"name":        "Vault",
		"description": "A hidden vault",
	})
	if err != nil {
		t.Fatalf("create_room without connection: %v", err)
	}
	if _, err := g.GetRoomByName("Vault"); err != nil {
		t.Error("expected Vault to exist")
	}
}

func TestDispatchUpdateRoom(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, err := dispatch(g, "update_room", map[string]any{
		"room_name":   "Tavern",
		"description": "A cleaner tavern now",
	})
	if err != nil {
		t.Fatalf("update_room: %v", err)
	}
	tavern, _ := g.GetRoomByName("Tavern")
	if tavern.Description != "A cleaner tavern now" {
		t.Errorf("expected updated description, got %q", tavern.Description)
	}
}

func TestDispatchCreateItemInRoom(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, err := dispatch(g, "create_item", map[string]any{
		"name":          "Rusty Sword",
		"description":   "A dull blade",
		"weight":        float64(2),
		"place_in_room": "Tavern",
	})
	if err != nil {
		t.Fatalf("create_item in room: %v", err)
	}
	item, err := g.GetItemByName("Rusty Sword")
	if err != nil {
		t.Fatal("item not found in registry")
	}
	tavern, _ := g.GetRoomByName("Tavern")
	if !tavern.HasItem(item.ID) {
		t.Error("expected item to be in Tavern")
	}
}

func TestDispatchCreateItemForPlayer(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, err := dispatch(g, "create_item", map[string]any{
		"name":        "Key",
		"description": "A brass key",
	})
	if err != nil {
		t.Fatalf("create_item for player: %v", err)
	}
	item, err := g.GetItemByName("Key")
	if err != nil {
		t.Fatal("item not in registry")
	}
	if !g.Player.HasItem(item.ID) {
		t.Error("expected item in player inventory when no place_in_room given")
	}
}

func TestDispatchCreateCharacter(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, err := dispatch(g, "create_character", map[string]any{
		"name":        "Barkeep",
		"description": "A tired man",
		"backstory":   "Worked here for years",
		"room_name":   "Tavern",
		"friendly":    true,
		"health":      float64(80),
	})
	if err != nil {
		t.Fatalf("create_character: %v", err)
	}
	npc, err := g.GetNPCByName("Barkeep")
	if err != nil {
		t.Fatal("NPC not found")
	}
	if npc.Health != 80 {
		t.Errorf("expected health 80, got %d", npc.Health)
	}
	tavern, _ := g.GetRoomByName("Tavern")
	if !tavern.HasOccupant(npc.ID) {
		t.Error("expected NPC in Tavern")
	}
}

func TestDispatchMoveCharacter(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	// First create the NPC
	_, _ = dispatch(g, "create_character", map[string]any{
		"name": "Guard", "description": "A guard", "backstory": "", "room_name": "Tavern",
	})
	_, err := dispatch(g, "move_character", map[string]any{
		"character_name": "Guard",
		"room_name":      "Alley",
	})
	if err != nil {
		t.Fatalf("move_character: %v", err)
	}
	guard, _ := g.GetNPCByName("Guard")
	alley, _ := g.GetRoomByName("Alley")
	if !alley.HasOccupant(guard.ID) {
		t.Error("expected guard to be in Alley after move")
	}
	tavern, _ := g.GetRoomByName("Tavern")
	if tavern.HasOccupant(guard.ID) {
		t.Error("expected guard removed from Tavern")
	}
}

func TestDispatchGiveAndTakeItem(t *testing.T) {
	g, startID, _ := newTestGameWithRooms(t)
	_, _ = dispatch(g, "create_item", map[string]any{
		"name": "Potion", "description": "Healing potion",
	})
	item, _ := g.GetItemByName("Potion")

	// Give to player
	_, err := dispatch(g, "give_item_to_player", map[string]any{"item_name": "Potion"})
	if err != nil {
		t.Fatalf("give_item_to_player: %v", err)
	}
	if !g.Player.HasItem(item.ID) {
		t.Error("expected Potion in player inventory")
	}

	// Take from player (drops in current room)
	_, err = dispatch(g, "take_item_from_player", map[string]any{"item_name": "Potion"})
	if err != nil {
		t.Fatalf("take_item_from_player: %v", err)
	}
	if g.Player.HasItem(item.ID) {
		t.Error("expected Potion removed from player inventory")
	}
	startRoom, _ := g.GetRoom(startID)
	if !startRoom.HasItem(item.ID) {
		t.Error("expected Potion dropped in player's current room")
	}
}

func TestDispatchDamageAndHealPlayer(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)

	_, err := dispatch(g, "damage_character", map[string]any{
		"character_name": "player",
		"amount":         float64(40),
	})
	if err != nil {
		t.Fatalf("damage player: %v", err)
	}
	if g.Player.Health != 60 {
		t.Errorf("expected player health 60, got %d", g.Player.Health)
	}

	_, err = dispatch(g, "heal_character", map[string]any{
		"character_name": "player",
		"amount":         float64(20),
	})
	if err != nil {
		t.Fatalf("heal player: %v", err)
	}
	if g.Player.Health != 80 {
		t.Errorf("expected player health 80, got %d", g.Player.Health)
	}
}

func TestDispatchDamageAndHealNPC(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, _ = dispatch(g, "create_character", map[string]any{
		"name": "Troll", "description": "Big", "backstory": "", "room_name": "Tavern",
		"health": float64(100),
	})

	_, err := dispatch(g, "damage_character", map[string]any{
		"character_name": "Troll",
		"amount":         float64(60),
	})
	if err != nil {
		t.Fatalf("damage NPC: %v", err)
	}
	troll, _ := g.GetNPCByName("Troll")
	if troll.Health != 40 {
		t.Errorf("expected Troll health 40, got %d", troll.Health)
	}

	_, err = dispatch(g, "heal_character", map[string]any{
		"character_name": "Troll",
		"amount":         float64(10),
	})
	if err != nil {
		t.Fatalf("heal NPC: %v", err)
	}
	troll, _ = g.GetNPCByName("Troll")
	if troll.Health != 50 {
		t.Errorf("expected Troll health 50 after heal, got %d", troll.Health)
	}
}

func TestDispatchSetCharacterAlive(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, _ = dispatch(g, "create_character", map[string]any{
		"name": "Bandit", "description": "", "backstory": "", "room_name": "Tavern",
	})

	_, err := dispatch(g, "set_character_alive", map[string]any{
		"character_name": "Bandit",
		"alive":          false,
	})
	if err != nil {
		t.Fatalf("kill NPC: %v", err)
	}
	bandit, _ := g.GetNPCByName("Bandit")
	if bandit.Alive {
		t.Error("expected Bandit to be dead")
	}

	_, err = dispatch(g, "set_character_alive", map[string]any{
		"character_name": "Bandit",
		"alive":          true,
	})
	if err != nil {
		t.Fatalf("revive NPC: %v", err)
	}
	bandit, _ = g.GetNPCByName("Bandit")
	if !bandit.Alive {
		t.Error("expected Bandit to be alive after revive")
	}
}

func TestDispatchGetRoomInfo(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	result, err := dispatch(g, "get_room_info", map[string]any{"room_name": "Tavern"})
	if err != nil {
		t.Fatalf("get_room_info: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty room info JSON")
	}
}

func TestDispatchUnknownTool(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	if _, err := dispatch(g, "summon_dragon", map[string]any{}); err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestNarratorToolsDefined(t *testing.T) {
	tools := ai.NarratorTools()
	if len(tools) == 0 {
		t.Error("expected non-empty narrator tools")
	}
	// Verify all tools have names
	for _, tool := range tools {
		if tool == nil {
			t.Error("got nil tool in NarratorTools")
		}
	}
}
