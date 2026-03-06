// Package ai provides the Bedrock client, tool definitions, and tool dispatch
// for the game's AI Dungeon Master.
package ai

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	brDocument "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// ---- Tool definitions sent to Bedrock in toolConfig ----

// NarratorTools are the tools available to the Narrator during gameplay chat.
// The AI calls these to mutate the game world in response to player actions.
func NarratorTools() []types.Tool {
	return []types.Tool{
		tool("create_room",
			"Create a new room connected to an existing room. Returns the new room's ID.",
			props(
				req("name", "string", "Display name of the room, e.g. 'The Rusty Tavern'"),
				req("description", "string", "Descriptive text the player sees when entering"),
				req("connect_to_room_name", "string", "Name of an existing room to connect this one to"),
				req("direction", "string", "Direction from connect_to_room_name to the new room (north/south/east/west/northeast/northwest/southeast/southwest/up/down)"),
			),
			[]string{"name", "description", "connect_to_room_name", "direction"},
		),
		tool("update_room",
			"Update an existing room's description.",
			props(
				req("room_name", "string", "Name of the room to update"),
				req("description", "string", "New description text"),
			),
			[]string{"room_name", "description"},
		),
		tool("create_item",
			"Create a new item and optionally place it in a room or the player's inventory.",
			props(
				req("name", "string", "Item name, e.g. 'Rusty Dagger'"),
				req("description", "string", "Item description"),
				opt("weight", "number", "Weight in kg (default 1.0)"),
				opt("place_in_room", "string", "Room name to place item in (leave empty for player inventory)"),
			),
			[]string{"name", "description"},
		),
		tool("create_character",
			"Create a new NPC and place them in a room.",
			props(
				req("name", "string", "Character name"),
				req("description", "string", "Physical description"),
				req("backstory", "string", "Brief backstory for the DM's use"),
				req("room_name", "string", "Name of room to place them in"),
				opt("friendly", "boolean", "Whether the character is friendly (default true)"),
				opt("health", "integer", "Starting health 1-100 (default 100)"),
			),
			[]string{"name", "description", "backstory", "room_name"},
		),
		tool("move_character",
			"Move an NPC from their current room to another room.",
			props(
				req("character_name", "string", "Name of the character to move"),
				req("room_name", "string", "Destination room name"),
			),
			[]string{"character_name", "room_name"},
		),
		tool("give_item_to_player",
			"Move an item from anywhere into the player's inventory.",
			props(
				req("item_name", "string", "Name of the item to give"),
			),
			[]string{"item_name"},
		),
		tool("take_item_from_player",
			"Remove an item from the player's inventory and drop it in their current room.",
			props(
				req("item_name", "string", "Name of the item to take"),
			),
			[]string{"item_name"},
		),
		tool("place_item_in_room",
			"Move an item (from anywhere) into a specific room.",
			props(
				req("item_name", "string", "Name of the item to move"),
				req("room_name", "string", "Destination room name"),
			),
			[]string{"item_name", "room_name"},
		),
		tool("damage_character",
			"Reduce a character's health by an amount. Use for NPCs and the player.",
			props(
				req("character_name", "string", "Character name, or 'player' for the player"),
				req("amount", "integer", "Damage amount (positive integer)"),
			),
			[]string{"character_name", "amount"},
		),
		tool("heal_character",
			"Restore a character's health by an amount.",
			props(
				req("character_name", "string", "Character name, or 'player' for the player"),
				req("amount", "integer", "Heal amount (positive integer)"),
			),
			[]string{"character_name", "amount"},
		),
		tool("set_character_alive",
			"Set a character's alive status (kill or revive).",
			props(
				req("character_name", "string", "Character name, or 'player' for the player"),
				req("alive", "boolean", "true to revive, false to kill"),
			),
			[]string{"character_name", "alive"},
		),
		tool("get_room_info",
			"Get full details about a room including items, occupants, and exits.",
			props(
				req("room_name", "string", "Name of the room"),
			),
			[]string{"room_name"},
		),
	}
}

// WorldBuilderTools are the minimal set used during world generation.
func WorldBuilderTools() []types.Tool {
	return NarratorTools() // world gen uses the same tool set
}

// ---- Tool dispatch ----

// DispatchTool executes a single tool call against the game and returns a
// result string to feed back to the model as a tool_result block.
func DispatchTool(g *game.Game, name string, input map[string]any) (string, error) {
	switch name {
	case "create_room":
		return execCreateRoom(g, input)
	case "update_room":
		return execUpdateRoom(g, input)
	case "create_item":
		return execCreateItem(g, input)
	case "create_character":
		return execCreateCharacter(g, input)
	case "move_character":
		return execMoveCharacter(g, input)
	case "give_item_to_player":
		return execGiveItemToPlayer(g, input)
	case "take_item_from_player":
		return execTakeItemFromPlayer(g, input)
	case "place_item_in_room":
		return execPlaceItemInRoom(g, input)
	case "damage_character":
		return execDamageCharacter(g, input)
	case "heal_character":
		return execHealCharacter(g, input)
	case "set_character_alive":
		return execSetCharacterAlive(g, input)
	case "get_room_info":
		return execGetRoomInfo(g, input)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// ---- Implementations ----

func execCreateRoom(g *game.Game, in map[string]any) (string, error) {
	name := strArg(in, "name")
	desc := strArg(in, "description")
	connectTo := strArg(in, "connect_to_room_name")
	direction := strArg(in, "direction")
	if name == "" || desc == "" {
		return "", fmt.Errorf("name and description are required")
	}
	room := game.NewArea(name, desc)
	if err := g.AddRoom(room); err != nil {
		return "", err
	}
	if connectTo != "" && direction != "" {
		fromRoom, err := g.GetRoomByName(connectTo)
		if err != nil {
			return "", fmt.Errorf("connect_to_room_name: %w", err)
		}
		if err := g.ConnectRooms(fromRoom.ID, room.ID, direction); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("Created room %q (ID: %s)", name, room.ID), nil
}

func execUpdateRoom(g *game.Game, in map[string]any) (string, error) {
	roomName := strArg(in, "room_name")
	desc := strArg(in, "description")
	room, err := g.GetRoomByName(roomName)
	if err != nil {
		return "", err
	}
	room.Description = desc
	g.UpdateRoom(room)
	return fmt.Sprintf("Updated room %q", roomName), nil
}

func execCreateItem(g *game.Game, in map[string]any) (string, error) {
	name := strArg(in, "name")
	desc := strArg(in, "description")
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	item := game.NewItem(name, desc)
	if w, ok := in["weight"].(float64); ok {
		item.Weight = w
	}
	if err := g.AddItem(item); err != nil {
		return "", err
	}
	if roomName := strArg(in, "place_in_room"); roomName != "" {
		room, err := g.GetRoomByName(roomName)
		if err != nil {
			return "", err
		}
		if err := g.PlaceItemInRoom(item.ID, room.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Created item %q and placed in %q", name, roomName), nil
	}
	if err := g.GiveItemToPlayer(item.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Created item %q and placed in player inventory", name), nil
}

func execCreateCharacter(g *game.Game, in map[string]any) (string, error) {
	name := strArg(in, "name")
	desc := strArg(in, "description")
	backstory := strArg(in, "backstory")
	roomName := strArg(in, "room_name")
	if name == "" || roomName == "" {
		return "", fmt.Errorf("name and room_name are required")
	}
	c := game.NewCharacter(name, desc)
	c.Backstory = backstory
	if friendly, ok := in["friendly"].(bool); ok {
		c.Friendly = friendly
	}
	if health, ok := in["health"].(float64); ok {
		c.Health = int(health)
	}
	if err := g.AddNPC(c); err != nil {
		return "", err
	}
	room, err := g.GetRoomByName(roomName)
	if err != nil {
		return "", err
	}
	if err := g.MoveNPC(c.ID, room.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Created character %q in room %q", name, roomName), nil
}

func execMoveCharacter(g *game.Game, in map[string]any) (string, error) {
	charName := strArg(in, "character_name")
	roomName := strArg(in, "room_name")
	c, err := g.GetNPCByName(charName)
	if err != nil {
		return "", err
	}
	room, err := g.GetRoomByName(roomName)
	if err != nil {
		return "", err
	}
	if err := g.MoveNPC(c.ID, room.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Moved %q to %q", charName, roomName), nil
}

func execGiveItemToPlayer(g *game.Game, in map[string]any) (string, error) {
	itemName := strArg(in, "item_name")
	item, err := g.GetItemByName(itemName)
	if err != nil {
		return "", err
	}
	if err := g.GiveItemToPlayer(item.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Gave %q to player", itemName), nil
}

func execTakeItemFromPlayer(g *game.Game, in map[string]any) (string, error) {
	itemName := strArg(in, "item_name")
	item, err := g.GetItemByName(itemName)
	if err != nil {
		return "", err
	}
	room, err := g.GetRoom(g.Player.LocationID)
	if err != nil {
		return "", err
	}
	if err := g.TakeItemFromPlayer(item.ID, room.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Took %q from player", itemName), nil
}

func execPlaceItemInRoom(g *game.Game, in map[string]any) (string, error) {
	itemName := strArg(in, "item_name")
	roomName := strArg(in, "room_name")
	item, err := g.GetItemByName(itemName)
	if err != nil {
		return "", err
	}
	room, err := g.GetRoomByName(roomName)
	if err != nil {
		return "", err
	}
	if err := g.PlaceItemInRoom(item.ID, room.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Placed %q in %q", itemName, roomName), nil
}

func execDamageCharacter(g *game.Game, in map[string]any) (string, error) {
	charName := strArg(in, "character_name")
	amount := int(numArg(in, "amount"))
	if charName == "player" {
		return fmt.Sprintf("Player health: %d", g.Player.Health-amount),
			g.Player.TakeDamage(amount)
	}
	c, err := g.GetNPCByName(charName)
	if err != nil {
		return "", err
	}
	if err := c.TakeDamage(amount); err != nil {
		return "", err
	}
	g.NPCs[c.ID] = c
	return fmt.Sprintf("%q health: %d", charName, c.Health), nil
}

func execHealCharacter(g *game.Game, in map[string]any) (string, error) {
	charName := strArg(in, "character_name")
	amount := int(numArg(in, "amount"))
	if charName == "player" {
		return fmt.Sprintf("Player health: %d", g.Player.Health+amount),
			g.Player.Heal(amount)
	}
	c, err := g.GetNPCByName(charName)
	if err != nil {
		return "", err
	}
	if err := c.Heal(amount); err != nil {
		return "", err
	}
	g.NPCs[c.ID] = c
	return fmt.Sprintf("%q health: %d", charName, c.Health), nil
}

func execSetCharacterAlive(g *game.Game, in map[string]any) (string, error) {
	charName := strArg(in, "character_name")
	alive, _ := in["alive"].(bool)
	if charName == "player" {
		if alive {
			return "Player revived", g.Player.Revive(50)
		}
		g.Player.Alive = false
		g.Player.Health = 0
		return "Player killed", nil
	}
	c, err := g.GetNPCByName(charName)
	if err != nil {
		return "", err
	}
	if alive {
		err = c.Revive(50)
	} else {
		c.Alive = false
		c.Health = 0
	}
	g.NPCs[c.ID] = c
	return fmt.Sprintf("%q alive=%v", charName, alive), err
}

func execGetRoomInfo(g *game.Game, in map[string]any) (string, error) {
	roomName := strArg(in, "room_name")
	room, err := g.GetRoomByName(roomName)
	if err != nil {
		return "", err
	}
	info := map[string]any{
		"name":        room.Name,
		"description": room.Description,
		"exits":       room.Connections,
		"item_count":  len(room.Items),
		"occupants":   len(room.Occupants),
	}
	b, _ := json.Marshal(info)
	return string(b), nil
}

// ---- helpers for building tool definitions ----

func tool(name, desc string, inputSchema map[string]any, required []string) types.Tool {
	inputSchema["required"] = required
	inputSchema["type"] = "object"
	return &types.ToolMemberToolSpec{
		Value: types.ToolSpecification{
			Name:        aws.String(name),
			Description: aws.String(desc),
			InputSchema: &types.ToolInputSchemaMemberJson{
				Value: brDocument.NewLazyDocument(inputSchema),
			},
		},
	}
}

func props(fields ...map[string]any) map[string]any {
	properties := make(map[string]any)
	for _, f := range fields {
		for k, v := range f {
			properties[k] = v
		}
	}
	return map[string]any{"properties": properties}
}

func req(name, typ, desc string) map[string]any {
	return map[string]any{name: map[string]string{"type": typ, "description": desc}}
}

func opt(name, typ, desc string) map[string]any {
	return map[string]any{name: map[string]string{"type": typ, "description": desc}}
}

func strArg(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func numArg(in map[string]any, key string) float64 {
	v, _ := in[key].(float64)
	return v
}
