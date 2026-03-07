package game_test

import (
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// ── Area ─────────────────────────────────────────────────────────────────────

func TestNewArea(t *testing.T) {
	a := game.NewArea("The Tavern", "A smoky room with a bar")
	if a.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if a.Name != "The Tavern" {
		t.Errorf("expected name 'The Tavern', got %q", a.Name)
	}
	if a.Connections == nil {
		t.Fatal("expected non-nil Connections map")
	}
	if len(a.Items) != 0 || len(a.Occupants) != 0 {
		t.Error("expected empty items and occupants")
	}
}

func TestAreaConnections(t *testing.T) {
	a := game.NewArea("Room A", "desc")
	b := game.NewArea("Room B", "desc")

	if err := a.AddConnection("north", b.ID); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	if a.Connections["north"] != b.ID {
		t.Errorf("expected north -> %s", b.ID)
	}

	// Duplicate direction should error
	if err := a.AddConnection("north", b.ID); err == nil {
		t.Error("expected error on duplicate direction")
	}

	// Invalid direction
	if err := a.AddConnection("sideways", b.ID); err == nil {
		t.Error("expected error on invalid direction")
	}

	// ForceConnection overwrites
	c := game.NewArea("Room C", "desc")
	if err := a.ForceConnection("north", c.ID); err != nil {
		t.Fatalf("ForceConnection: %v", err)
	}
	if a.Connections["north"] != c.ID {
		t.Error("expected ForceConnection to overwrite")
	}

	// Remove
	if err := a.RemoveConnection("north"); err != nil {
		t.Fatalf("RemoveConnection: %v", err)
	}
	if _, ok := a.Connections["north"]; ok {
		t.Error("expected connection to be removed")
	}
	if err := a.RemoveConnection("north"); err == nil {
		t.Error("expected error removing non-existent connection")
	}
}

func TestAreaItems(t *testing.T) {
	a := game.NewArea("Room", "desc")
	if err := a.AddItemID("item-1"); err != nil {
		t.Fatalf("AddItemID: %v", err)
	}
	if !a.HasItem("item-1") {
		t.Error("expected HasItem to be true")
	}
	if err := a.AddItemID("item-1"); err == nil {
		t.Error("expected error on duplicate item")
	}
	if err := a.RemoveItemID("item-1"); err != nil {
		t.Fatalf("RemoveItemID: %v", err)
	}
	if a.HasItem("item-1") {
		t.Error("expected HasItem to be false after removal")
	}
	if err := a.RemoveItemID("item-1"); err == nil {
		t.Error("expected error removing non-existent item")
	}
}

func TestAreaOccupants(t *testing.T) {
	a := game.NewArea("Room", "desc")
	if err := a.AddOccupant("char-1"); err != nil {
		t.Fatalf("AddOccupant: %v", err)
	}
	if !a.HasOccupant("char-1") {
		t.Error("expected HasOccupant true")
	}
	if err := a.AddOccupant("char-1"); err == nil {
		t.Error("expected error on duplicate occupant")
	}
	if err := a.RemoveOccupant("char-1"); err != nil {
		t.Fatalf("RemoveOccupant: %v", err)
	}
	if a.HasOccupant("char-1") {
		t.Error("expected HasOccupant false after removal")
	}
}

// ── Character ─────────────────────────────────────────────────────────────────

func TestCharacterDamageAndHeal(t *testing.T) {
	c := game.NewCharacter("Aragorn", "A ranger")

	if err := c.TakeDamage(30); err != nil {
		t.Fatalf("TakeDamage: %v", err)
	}
	if c.Health != 70 {
		t.Errorf("expected health 70, got %d", c.Health)
	}
	if !c.Alive {
		t.Error("expected still alive")
	}

	if err := c.Heal(20); err != nil {
		t.Fatalf("Heal: %v", err)
	}
	if c.Health != 90 {
		t.Errorf("expected health 90, got %d", c.Health)
	}

	// Cap at 100
	if err := c.Heal(50); err != nil {
		t.Fatalf("Heal overshoot: %v", err)
	}
	if c.Health != 100 {
		t.Errorf("expected capped health 100, got %d", c.Health)
	}
}

func TestCharacterDeath(t *testing.T) {
	c := game.NewCharacter("Goblin", "A weak goblin")

	if err := c.TakeDamage(200); err != nil {
		t.Fatalf("TakeDamage: %v", err)
	}
	if c.Alive {
		t.Error("expected character to be dead")
	}
	if c.Health != 0 {
		t.Errorf("expected health 0, got %d", c.Health)
	}

	// Can't damage dead character
	if err := c.TakeDamage(1); err == nil {
		t.Error("expected error damaging dead character")
	}

	// Can't heal dead character
	if err := c.Heal(50); err == nil {
		t.Error("expected error healing dead character")
	}

	// Revive
	if err := c.Revive(50); err != nil {
		t.Fatalf("Revive: %v", err)
	}
	if !c.Alive || c.Health != 50 {
		t.Error("expected character alive with 50 health after revive")
	}

	// Can't revive living character
	if err := c.Revive(50); err == nil {
		t.Error("expected error reviving living character")
	}
}

func TestCharacterInventory(t *testing.T) {
	c := game.NewCharacter("Player", "The hero")
	if err := c.AddItemID("sword"); err != nil {
		t.Fatalf("AddItemID: %v", err)
	}
	if !c.HasItem("sword") {
		t.Error("expected HasItem true")
	}
	if err := c.AddItemID("sword"); err == nil {
		t.Error("expected error on duplicate item in inventory")
	}
	if err := c.RemoveItemID("sword"); err != nil {
		t.Fatalf("RemoveItemID: %v", err)
	}
	if c.HasItem("sword") {
		t.Error("expected HasItem false after removal")
	}
}

// ── Game engine ───────────────────────────────────────────────────────────────

func newTestGame() *game.Game {
	g := game.NewGame("session-1", "user-1")
	g.Player = game.NewCharacter("Hero", "The player character")
	return g
}

func TestGameRoomCRUD(t *testing.T) {
	g := newTestGame()

	tavern := game.NewArea("Tavern", "A smoky tavern")
	if err := g.AddRoom(tavern); err != nil {
		t.Fatalf("AddRoom: %v", err)
	}

	// Duplicate ID should fail
	if err := g.AddRoom(tavern); err == nil {
		t.Error("expected error on duplicate room ID")
	}

	got, err := g.GetRoom(tavern.ID)
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if got.Name != "Tavern" {
		t.Errorf("expected name Tavern, got %q", got.Name)
	}

	// GetRoomByName
	byName, err := g.GetRoomByName("Tavern")
	if err != nil {
		t.Fatalf("GetRoomByName: %v", err)
	}
	if byName.ID != tavern.ID {
		t.Error("GetRoomByName returned wrong room")
	}

	// Not found
	if _, err := g.GetRoomByName("Nonexistent"); err == nil {
		t.Error("expected error for unknown room name")
	}
}

func TestConnectRoomsAndCoordinates(t *testing.T) {
	g := newTestGame()

	a := game.NewArea("Start", "The start")
	b := game.NewArea("North Room", "To the north")
	_ = g.AddRoom(a)
	_ = g.AddRoom(b)

	if err := g.ConnectRooms(a.ID, b.ID, "north"); err != nil {
		t.Fatalf("ConnectRooms: %v", err)
	}

	aUpdated, _ := g.GetRoom(a.ID)
	bUpdated, _ := g.GetRoom(b.ID)

	if aUpdated.Connections["north"] != b.ID {
		t.Errorf("expected a.north = b.ID")
	}
	if bUpdated.Connections["south"] != a.ID {
		t.Errorf("expected b.south = a.ID (bidirectional)")
	}
	if bUpdated.Coordinates.Y >= 0 {
		t.Errorf("expected north room to have negative Y coordinate, got %f", bUpdated.Coordinates.Y)
	}
}

func TestDeleteRoomCleansConnections(t *testing.T) {
	g := newTestGame()
	a := game.NewArea("A", "")
	b := game.NewArea("B", "")
	_ = g.AddRoom(a)
	_ = g.AddRoom(b)
	_ = g.ConnectRooms(a.ID, b.ID, "east")

	if err := g.DeleteRoom(b.ID); err != nil {
		t.Fatalf("DeleteRoom: %v", err)
	}
	aUpdated, _ := g.GetRoom(a.ID)
	if _, ok := aUpdated.Connections["east"]; ok {
		t.Error("expected back-reference to be cleaned up on DeleteRoom")
	}
}

func TestPlaceItemInRoom(t *testing.T) {
	g := newTestGame()
	room := game.NewArea("Room", "")
	_ = g.AddRoom(room)

	item := game.NewItem("Sword", "A sharp sword")
	_ = g.AddItem(item)

	if err := g.PlaceItemInRoom(item.ID, room.ID); err != nil {
		t.Fatalf("PlaceItemInRoom: %v", err)
	}
	r, _ := g.GetRoom(room.ID)
	if !r.HasItem(item.ID) {
		t.Error("expected item to be in room")
	}

	// Moving it to player should remove from room
	if err := g.GiveItemToPlayer(item.ID); err != nil {
		t.Fatalf("GiveItemToPlayer: %v", err)
	}
	rAfter, _ := g.GetRoom(room.ID)
	if rAfter.HasItem(item.ID) {
		t.Error("expected item removed from room after given to player")
	}
	if !g.Player.HasItem(item.ID) {
		t.Error("expected item in player inventory")
	}
}

func TestMovePlayer(t *testing.T) {
	g := newTestGame()
	start := game.NewArea("Start", "")
	north := game.NewArea("North", "")
	_ = g.AddRoom(start)
	_ = g.AddRoom(north)
	_ = g.ConnectRooms(start.ID, north.ID, "north")
	_ = g.PlacePlayer(start.ID)

	dest, err := g.MovePlayer("north")
	if err != nil {
		t.Fatalf("MovePlayer: %v", err)
	}
	if dest.ID != north.ID {
		t.Errorf("expected to be in north room, got %s", dest.ID)
	}
	if g.Player.LocationID != north.ID {
		t.Error("expected Player.LocationID to update")
	}

	// No exit to the south from north should fail (well, south is connected back)
	// Try an unconnected direction
	if _, err := g.MovePlayer("west"); err == nil {
		t.Error("expected error moving in direction with no exit")
	}
}

func TestNPCLifecycle(t *testing.T) {
	g := newTestGame()
	room := game.NewArea("Room", "")
	_ = g.AddRoom(room)

	npc := game.NewCharacter("Barkeep", "A tired barkeep")
	_ = g.AddNPC(npc)

	if err := g.MoveNPC(npc.ID, room.ID); err != nil {
		t.Fatalf("MoveNPC: %v", err)
	}
	r, _ := g.GetRoom(room.ID)
	if !r.HasOccupant(npc.ID) {
		t.Error("expected NPC to be in room")
	}

	// Move to another room
	room2 := game.NewArea("Room 2", "")
	_ = g.AddRoom(room2)
	if err := g.MoveNPC(npc.ID, room2.ID); err != nil {
		t.Fatalf("MoveNPC to room2: %v", err)
	}
	r1After, _ := g.GetRoom(room.ID)
	r2After, _ := g.GetRoom(room2.ID)
	if r1After.HasOccupant(npc.ID) {
		t.Error("expected NPC removed from old room")
	}
	if !r2After.HasOccupant(npc.ID) {
		t.Error("expected NPC in new room")
	}
}

func TestCalculateRoomCoordinates(t *testing.T) {
	g := newTestGame()
	start := game.NewArea("Start", "")
	east := game.NewArea("East", "")
	north := game.NewArea("North", "")
	_ = g.AddRoom(start)
	_ = g.AddRoom(east)
	_ = g.AddRoom(north)
	_ = g.ConnectRooms(start.ID, east.ID, "east")
	_ = g.ConnectRooms(start.ID, north.ID, "north")
	_ = g.PlacePlayer(start.ID)

	g.CalculateRoomCoordinates()

	startRoom, _ := g.GetRoom(start.ID)
	eastRoom, _ := g.GetRoom(east.ID)
	northRoom, _ := g.GetRoom(north.ID)

	if startRoom.Coordinates.X != 0 || startRoom.Coordinates.Y != 0 {
		t.Errorf("expected start at (0,0), got (%f,%f)", startRoom.Coordinates.X, startRoom.Coordinates.Y)
	}
	if eastRoom.Coordinates.X <= startRoom.Coordinates.X {
		t.Error("expected east room to have greater X coordinate")
	}
	if northRoom.Coordinates.Y >= startRoom.Coordinates.Y {
		t.Error("expected north room to have lesser Y coordinate")
	}
}

// ── Serialisation ─────────────────────────────────────────────────────────────

func TestSaveStateRoundtrip(t *testing.T) {
	g := newTestGame()
	room := game.NewArea("Tavern", "A smoky tavern")
	_ = g.AddRoom(room)
	_ = g.PlacePlayer(room.ID)
	item := game.NewItem("Sword", "Sharp")
	_ = g.AddItem(item)
	_ = g.GiveItemToPlayer(item.ID)
	npc := game.NewCharacter("Goblin", "Green and mean")
	_ = g.AddNPC(npc)
	_ = g.MoveNPC(npc.ID, room.ID)
	g.Ready = true
	g.Version = 3

	history := []game.ChatMessage{{Type: "player", Content: "Hello"}}
	narrative := []game.NarrativeMessage{{Role: "assistant", Content: []game.NarrativeBlock{{Type: "text", Text: "You enter the tavern."}}}}

	saved := g.ToSaveState(narrative, history)
	if saved.SchemaVersion != game.SchemaVersion {
		t.Errorf("expected schema version %d, got %d", game.SchemaVersion, saved.SchemaVersion)
	}
	if saved.Version != 3 {
		t.Errorf("expected version 3, got %d", saved.Version)
	}

	restored, err := game.FromSaveState(saved)
	if err != nil {
		t.Fatalf("FromSaveState: %v", err)
	}
	if restored.Player.LocationID != room.ID {
		t.Error("player location not preserved")
	}
	if !restored.Player.HasItem(item.ID) {
		t.Error("player inventory not preserved")
	}
	if _, err := restored.GetNPC(npc.ID); err != nil {
		t.Error("NPC not preserved")
	}
	if !restored.Ready {
		t.Error("Ready flag not preserved")
	}
}

func TestFromSaveStateSchemaVersionMismatch(t *testing.T) {
	bad := game.SaveState{SchemaVersion: 999}
	if _, err := game.FromSaveState(bad); err == nil {
		t.Error("expected error for incompatible schema version")
	}
}

// ── BuildGameStateView ────────────────────────────────────────────────────────

func TestBuildGameStateView(t *testing.T) {
	g := newTestGame()
	room := game.NewArea("Tavern", "A cozy tavern")
	_ = g.AddRoom(room)
	_ = g.PlacePlayer(room.ID)
	item := game.NewItem("Dagger", "A small blade")
	_ = g.AddItem(item)
	_ = g.GiveItemToPlayer(item.ID)

	history := []game.ChatMessage{{Type: "player", Content: "hi"}}
	view := g.BuildGameStateView(history)

	if view.CurrentRoom.Name != "Tavern" {
		t.Errorf("expected current room name 'Tavern', got %q", view.CurrentRoom.Name)
	}
	if len(view.Player.Inventory) != 1 {
		t.Errorf("expected 1 inventory item, got %d", len(view.Player.Inventory))
	}
	if view.Player.Inventory[0].Name != "Dagger" {
		t.Errorf("expected item 'Dagger', got %q", view.Player.Inventory[0].Name)
	}
	if len(view.ChatHistory) != 1 {
		t.Error("expected 1 chat message in view")
	}
}
