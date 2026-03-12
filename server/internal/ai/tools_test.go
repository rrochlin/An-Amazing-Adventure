package ai_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// newTestGameWithRooms sets up a minimal game with two connected rooms and a player.
func newTestGameWithRooms(t *testing.T) (*game.Game, string, string) {
	t.Helper()
	g := game.NewGame("test-session", "test-user")
	g.SetPlayerCharacter("test-user", game.NewCharacter("Hero", "The protagonist"))

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

// dispatch is a helper that calls DispatchTool and discards the WorldEvent return.
func dispatch(g *game.Game, name string, args map[string]any) (string, error) {
	result, _, err := ai.DispatchTool(context.Background(), g, name, args)
	return result, err
}

// dispatchWithEvent is a helper that returns the WorldEvent alongside result.
func dispatchWithEvent(g *game.Game, name string, args map[string]any) (string, *game.WorldEvent, error) {
	return ai.DispatchTool(context.Background(), g, name, args)
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

func TestDispatchUpdateRoom_CaseInsensitiveRoomName(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, err := dispatch(g, "update_room", map[string]any{
		"room_name":   "tavern",
		"description": "Lowercase lookup works",
	})
	if err != nil {
		t.Fatalf("update_room with lowercase room_name: %v", err)
	}
	tavern, _ := g.GetRoomByName("Tavern")
	if tavern.Description != "Lowercase lookup works" {
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
	ownerChar, _ := g.OwnerCharacter()
	if !ownerChar.HasItem(item.ID) {
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
	ownerChar, _ := g.OwnerCharacter()
	if !ownerChar.HasItem(item.ID) {
		t.Error("expected Potion in player inventory")
	}

	// Take from player (drops in current room)
	_, err = dispatch(g, "take_item_from_player", map[string]any{"item_name": "Potion"})
	if err != nil {
		t.Fatalf("take_item_from_player: %v", err)
	}
	ownerChar2, _ := g.OwnerCharacter()
	if ownerChar2.HasItem(item.ID) {
		t.Error("expected Potion removed from player inventory")
	}
	startRoom, _ := g.GetRoom(startID)
	if !startRoom.HasItem(item.ID) {
		t.Error("expected Potion dropped in player's current room")
	}
}

func TestDispatchTriggerShortRest(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	// short rest with no DnD players — should not error
	result, err := dispatch(g, "trigger_short_rest", map[string]any{
		"reason": "The party finds a quiet alcove",
	})
	if err != nil {
		t.Fatalf("trigger_short_rest: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestDispatchTriggerLongRest(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	// long rest with no DnD players — should not error
	result, err := dispatch(g, "trigger_long_rest", map[string]any{
		"reason": "The party makes camp for the night",
	})
	if err != nil {
		t.Fatalf("trigger_long_rest: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
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

// ---- Visibility tests ----

func TestGiveItemToPlayerAlwaysProducesEvent(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, _ = dispatch(g, "create_item", map[string]any{
		"name": "Sword", "description": "Sharp",
		"place_in_room": "Alley", // somewhere not the player
	})
	_, ev, err := dispatchWithEvent(g, "give_item_to_player", map[string]any{
		"item_name": "Sword",
	})
	if err != nil {
		t.Fatalf("give_item_to_player: %v", err)
	}
	if ev == nil {
		t.Fatal("expected WorldEvent for give_item_to_player, got nil")
	}
	if ev.Type != "item_gained" {
		t.Errorf("expected type 'item_gained', got %q", ev.Type)
	}
}

func TestPlaceItemInPlayerRoom_ProducesEvent(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	// Create item in Alley first, then move to Tavern (player's room)
	_, _ = dispatch(g, "create_item", map[string]any{
		"name": "Gem", "description": "Shiny",
		"place_in_room": "Alley",
	})
	_, ev, err := dispatchWithEvent(g, "place_item_in_room", map[string]any{
		"item_name": "Gem",
		"room_name": "Tavern",
	})
	if err != nil {
		t.Fatalf("place_item_in_room: %v", err)
	}
	if ev == nil {
		t.Fatal("expected WorldEvent when item placed in player's room, got nil")
	}
	if ev.Type != "item_appeared" {
		t.Errorf("expected type 'item_appeared', got %q", ev.Type)
	}
}

func TestPlaceItemInOtherRoom_NoEvent(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, _ = dispatch(g, "create_item", map[string]any{
		"name": "Rock", "description": "Heavy",
		"place_in_room": "Tavern",
	})
	_, ev, err := dispatchWithEvent(g, "place_item_in_room", map[string]any{
		"item_name": "Rock",
		"room_name": "Alley",
	})
	if err != nil {
		t.Fatalf("place_item_in_room other room: %v", err)
	}
	if ev != nil {
		t.Errorf("expected nil event when placing item in different room, got %+v", ev)
	}
}

func TestMoveCharacterArrivesAtPlayerRoom_ProducesEvent(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	// NPC starts in Alley; player is in Tavern
	_, _ = dispatch(g, "create_character", map[string]any{
		"name": "Messenger", "description": "", "backstory": "", "room_name": "Alley",
	})
	_, ev, err := dispatchWithEvent(g, "move_character", map[string]any{
		"character_name": "Messenger",
		"room_name":      "Tavern",
	})
	if err != nil {
		t.Fatalf("move_character: %v", err)
	}
	if ev == nil {
		t.Fatal("expected WorldEvent when NPC arrives at player room, got nil")
	}
	if ev.Type != "character_arrived" {
		t.Errorf("expected type 'character_arrived', got %q", ev.Type)
	}
}

func TestMoveCharacterDepartsPlayerRoom_ProducesEvent(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	// NPC starts in Tavern (same as player); moves to Alley
	_, _ = dispatch(g, "create_character", map[string]any{
		"name": "Thief", "description": "", "backstory": "", "room_name": "Tavern",
	})
	_, ev, err := dispatchWithEvent(g, "move_character", map[string]any{
		"character_name": "Thief",
		"room_name":      "Alley",
	})
	if err != nil {
		t.Fatalf("move_character departs: %v", err)
	}
	if ev == nil {
		t.Fatal("expected WorldEvent when NPC departs player room, got nil")
	}
	if ev.Type != "character_departed" {
		t.Errorf("expected type 'character_departed', got %q", ev.Type)
	}
}

func TestGetRoomInfo_NoEvent(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, ev, err := dispatchWithEvent(g, "get_room_info", map[string]any{"room_name": "Tavern"})
	if err != nil {
		t.Fatalf("get_room_info: %v", err)
	}
	if ev != nil {
		t.Errorf("expected nil event for read-only get_room_info, got %+v", ev)
	}
}

func TestDispatchMoveCharacter_PartialNameMatch(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, _ = dispatch(g, "create_character", map[string]any{
		"name":        "City Guard",
		"description": "Watchful",
		"backstory":   "Patrol duty",
		"room_name":   "Tavern",
	})

	_, err := dispatch(g, "move_character", map[string]any{
		"character_name": "guard",
		"room_name":      "alley",
	})
	if err != nil {
		t.Fatalf("move_character with partial/case-insensitive names: %v", err)
	}
	npc, err := g.GetNPCByName("City Guard")
	if err != nil {
		t.Fatalf("expected NPC to exist: %v", err)
	}
	alley, _ := g.GetRoomByName("Alley")
	if !alley.HasOccupant(npc.ID) {
		t.Error("expected City Guard in Alley after move")
	}
}

func TestDispatchRoomLookupErrorListsKnownRooms(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, err := dispatch(g, "update_room", map[string]any{
		"room_name":   "Crypt of Ash",
		"description": "irrelevant",
	})
	if err == nil {
		t.Fatal("expected error for unknown room")
	}
	msg := err.Error()
	if !strings.Contains(msg, "known rooms") {
		t.Fatalf("expected known rooms in error message, got: %s", msg)
	}
	if !strings.Contains(msg, "Tavern") || !strings.Contains(msg, "Alley") {
		t.Fatalf("expected room candidates in error message, got: %s", msg)
	}
}

func TestCreateRoom_NoEvent(t *testing.T) {
	g, _, _ := newTestGameWithRooms(t)
	_, ev, err := dispatchWithEvent(g, "create_room", map[string]any{
		"name":        "Dungeon",
		"description": "Dark",
	})
	if err != nil {
		t.Fatalf("create_room: %v", err)
	}
	if ev != nil {
		t.Errorf("expected nil event for create_room, got %+v", ev)
	}
}
