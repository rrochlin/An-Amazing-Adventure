// Package ai provides the Bedrock client, tool definitions, and tool dispatch
// for the game's AI Dungeon Master.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

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
		tool("trigger_short_rest",
			"Trigger a short rest for all players. Restores Fighter Second Wind, Action Surge, Monk Ki. Does NOT restore hit points.",
			props(
				req("reason", "string", "Brief in-world reason for the rest (e.g. 'The party finds a quiet alcove')"),
			),
			[]string{"reason"},
		),
		tool("trigger_long_rest",
			"Trigger a long rest for all players. Restores all resources, spell slots, and full hit points.",
			props(
				req("reason", "string", "Brief in-world reason for the rest (e.g. 'The party makes camp for the night')"),
			),
			[]string{"reason"},
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

// DispatchTool executes a single tool call against the game.
// Returns:
//   - result: string to feed back to the model as a tool_result block
//   - event: player-visible WorldEvent if the player can observe the mutation, otherwise nil
//   - err: non-nil if the tool call failed
func DispatchTool(ctx context.Context, g *game.Game, name string, input map[string]any) (result string, event *game.WorldEvent, err error) {
	switch name {
	case "create_room":
		result, event, err = execCreateRoom(g, input)
	case "update_room":
		result, event, err = execUpdateRoom(g, input)
	case "create_item":
		result, event, err = execCreateItem(g, input)
	case "create_character":
		result, event, err = execCreateCharacter(g, input)
	case "move_character":
		result, event, err = execMoveCharacter(g, input)
	case "give_item_to_player":
		result, event, err = execGiveItemToPlayer(g, input)
	case "take_item_from_player":
		result, event, err = execTakeItemFromPlayer(g, input)
	case "place_item_in_room":
		result, event, err = execPlaceItemInRoom(g, input)
	case "trigger_short_rest":
		result, event, err = execTriggerShortRest(ctx, g, input)
	case "trigger_long_rest":
		result, event, err = execTriggerLongRest(ctx, g, input)
	case "get_room_info":
		result, event, err = execGetRoomInfo(g, input)
	default:
		err = fmt.Errorf("unknown tool: %s", name)
	}
	return
}

// ---- Implementations ----
// Each exec* function returns (result, event, err).
// event is non-nil only when the player can observe the mutation.

func execCreateRoom(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	name := strArg(in, "name")
	desc := strArg(in, "description")
	connectTo := strArg(in, "connect_to_room_name")
	direction := strArg(in, "direction")
	if name == "" || desc == "" {
		return "", nil, fmt.Errorf("name and description are required")
	}
	room := game.NewArea(name, desc)
	if err := g.AddRoom(room); err != nil {
		return "", nil, err
	}
	if connectTo != "" && direction != "" {
		fromRoom, err := resolveRoomByName(g, connectTo)
		if err != nil {
			return "", nil, fmt.Errorf("connect_to_room_name: %w", err)
		}
		if err := g.ConnectRooms(fromRoom.ID, room.ID, direction); err != nil {
			return "", nil, err
		}
	}
	// create_room: player visibility — never (description visible in next narrative)
	return fmt.Sprintf("Created room %q (ID: %s)", name, room.ID), nil, nil
}

func execUpdateRoom(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	roomName := strArg(in, "room_name")
	desc := strArg(in, "description")
	room, err := resolveRoomByName(g, roomName)
	if err != nil {
		return "", nil, err
	}
	room.Description = desc
	g.UpdateRoom(room)
	// update_room: player visibility — never (description change visible in next narrative)
	return fmt.Sprintf("Updated room %q", roomName), nil, nil
}

func execCreateItem(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	name := strArg(in, "name")
	desc := strArg(in, "description")
	if name == "" {
		return "", nil, fmt.Errorf("name is required")
	}
	item := game.NewItem(name, desc)
	if w, ok := in["weight"].(float64); ok {
		item.Weight = w
	}
	if err := g.AddItem(item); err != nil {
		return "", nil, err
	}
	if roomName := strArg(in, "place_in_room"); roomName != "" {
		room, err := resolveRoomByName(g, roomName)
		if err != nil {
			return "", nil, err
		}
		if err := g.PlaceItemInRoom(item.ID, room.ID); err != nil {
			return "", nil, err
		}
		// Visible if placed in player's current room
		var ev *game.WorldEvent
		owner, _ := g.OwnerCharacter()
		if room.ID == owner.LocationID {
			ev = &game.WorldEvent{Type: "item_appeared", Message: fmt.Sprintf("A %s appears nearby.", name)}
		}
		return fmt.Sprintf("Created item %q and placed in %q", name, roomName), ev, nil
	}
	if err := g.GiveItemToPlayer(item.ID); err != nil {
		return "", nil, err
	}
	// Placed in player inventory — always visible
	ev := &game.WorldEvent{Type: "item_gained", Message: fmt.Sprintf("%s added to your inventory.", name)}
	return fmt.Sprintf("Created item %q and placed in player inventory", name), ev, nil
}

func execCreateCharacter(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	name := strArg(in, "name")
	desc := strArg(in, "description")
	backstory := strArg(in, "backstory")
	roomName := strArg(in, "room_name")
	if name == "" || roomName == "" {
		return "", nil, fmt.Errorf("name and room_name are required")
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
		return "", nil, err
	}
	room, err := resolveRoomByName(g, roomName)
	if err != nil {
		return "", nil, err
	}
	if err := g.MoveNPC(c.ID, room.ID); err != nil {
		return "", nil, err
	}
	// Visible if placed in player's current room
	var ev *game.WorldEvent
	owner, _ := g.OwnerCharacter()
	if room.ID == owner.LocationID {
		ev = &game.WorldEvent{Type: "character_arrived", Message: fmt.Sprintf("%s appears.", name)}
	}
	return fmt.Sprintf("Created character %q in room %q", name, roomName), ev, nil
}

func execMoveCharacter(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	charName := strArg(in, "character_name")
	roomName := strArg(in, "room_name")
	c, err := resolveNPCByName(g, charName)
	if err != nil {
		return "", nil, err
	}
	fromRoomID := c.LocationID
	room, err := resolveRoomByName(g, roomName)
	if err != nil {
		return "", nil, err
	}
	if err := g.MoveNPC(c.ID, room.ID); err != nil {
		return "", nil, err
	}
	// Visible if NPC arrives at or departs from player's current room
	owner, _ := g.OwnerCharacter()
	playerRoom := owner.LocationID
	var ev *game.WorldEvent
	if room.ID == playerRoom {
		ev = &game.WorldEvent{Type: "character_arrived", Message: fmt.Sprintf("%s arrives.", charName)}
	} else if fromRoomID == playerRoom {
		ev = &game.WorldEvent{Type: "character_departed", Message: fmt.Sprintf("%s leaves.", charName)}
	}
	return fmt.Sprintf("Moved %q to %q", charName, roomName), ev, nil
}

func execGiveItemToPlayer(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	itemName := strArg(in, "item_name")
	item, err := resolveItemByName(g, itemName)
	if err != nil {
		return "", nil, err
	}
	if err := g.GiveItemToPlayer(item.ID); err != nil {
		return "", nil, err
	}
	// Always visible — player receives item
	ev := &game.WorldEvent{Type: "item_gained", Message: fmt.Sprintf("%s added to your inventory.", itemName)}
	return fmt.Sprintf("Gave %q to player", itemName), ev, nil
}

func execTakeItemFromPlayer(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	itemName := strArg(in, "item_name")
	item, err := resolveItemByName(g, itemName)
	if err != nil {
		return "", nil, err
	}
	owner, _ := g.OwnerCharacter()
	room, err := g.GetRoom(owner.LocationID)
	if err != nil {
		return "", nil, err
	}
	if err := g.TakeItemFromPlayer(item.ID, room.ID); err != nil {
		return "", nil, err
	}
	// Always visible — item removed from player
	ev := &game.WorldEvent{Type: "item_lost", Message: fmt.Sprintf("%s removed from your inventory.", itemName)}
	return fmt.Sprintf("Took %q from player", itemName), ev, nil
}

func execPlaceItemInRoom(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	itemName := strArg(in, "item_name")
	roomName := strArg(in, "room_name")
	item, err := resolveItemByName(g, itemName)
	if err != nil {
		return "", nil, err
	}
	room, err := resolveRoomByName(g, roomName)
	if err != nil {
		return "", nil, err
	}
	if err := g.PlaceItemInRoom(item.ID, room.ID); err != nil {
		return "", nil, err
	}
	// Visible only if placed in player's current room
	owner, _ := g.OwnerCharacter()
	var ev *game.WorldEvent
	if room.ID == owner.LocationID {
		ev = &game.WorldEvent{Type: "item_appeared", Message: fmt.Sprintf("A %s appears nearby.", itemName)}
	}
	return fmt.Sprintf("Placed %q in %q", itemName, roomName), ev, nil
}

func execTriggerShortRest(ctx context.Context, g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	reason := strArg(in, "reason")
	var errs []string
	restored := 0
	for uid, dndChar := range g.DnDPlayers {
		if dndChar == nil {
			continue
		}
		if err := dndChar.ShortRest(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", uid, err))
		} else {
			restored++
		}
	}
	result := fmt.Sprintf("Short rest taken (%s). %d characters restored resources.", reason, restored)
	if len(errs) > 0 {
		result += " Errors: " + fmt.Sprint(errs)
	}
	ev := &game.WorldEvent{Type: "heal", Message: "The party takes a short rest and recovers their resources."}
	return result, ev, nil
}

func execTriggerLongRest(ctx context.Context, g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	reason := strArg(in, "reason")
	var errs []string
	restored := 0
	for uid, dndChar := range g.DnDPlayers {
		if dndChar == nil {
			continue
		}
		if err := dndChar.LongRest(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", uid, err))
		} else {
			restored++
		}
	}
	result := fmt.Sprintf("Long rest taken (%s). %d characters fully restored.", reason, restored)
	if len(errs) > 0 {
		result += " Errors: " + fmt.Sprint(errs)
	}
	ev := &game.WorldEvent{Type: "heal", Message: "The party takes a long rest and recovers fully."}
	return result, ev, nil
}

func execGetRoomInfo(g *game.Game, in map[string]any) (string, *game.WorldEvent, error) {
	roomName := strArg(in, "room_name")
	room, err := resolveRoomByName(g, roomName)
	if err != nil {
		return "", nil, err
	}
	info := map[string]any{
		"name":        room.Name,
		"description": room.Description,
		"exits":       room.Connections,
		"item_count":  len(room.Items),
		"occupants":   len(room.Occupants),
	}
	b, _ := json.Marshal(info)
	// get_room_info: read-only, no player event
	return string(b), nil, nil
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

func resolveRoomByName(g *game.Game, name string) (game.Area, error) {
	if room, err := g.GetRoomByName(name); err == nil {
		return room, nil
	}
	normTarget := normalizeLookupKey(name)
	if normTarget == "" {
		return game.Area{}, fmt.Errorf("room name is required")
	}

	var partial []game.Area
	for _, room := range g.Rooms {
		normRoom := normalizeLookupKey(room.Name)
		if normRoom == normTarget {
			return room, nil
		}
		if strings.Contains(normRoom, normTarget) || strings.Contains(normTarget, normRoom) {
			partial = append(partial, room)
		}
	}
	if len(partial) == 1 {
		return partial[0], nil
	}
	if len(partial) > 1 {
		return game.Area{}, fmt.Errorf("room name %q is ambiguous; candidates: %s", name, strings.Join(roomNames(g), ", "))
	}
	return game.Area{}, fmt.Errorf("room named %q not found; known rooms: %s", name, strings.Join(roomNames(g), ", "))
}

func resolveItemByName(g *game.Game, name string) (game.Item, error) {
	if item, err := g.GetItemByName(name); err == nil {
		return item, nil
	}
	normTarget := normalizeLookupKey(name)
	if normTarget == "" {
		return game.Item{}, fmt.Errorf("item name is required")
	}

	var partial []game.Item
	for _, item := range g.Items {
		normItem := normalizeLookupKey(item.Name)
		if normItem == normTarget {
			return item, nil
		}
		if strings.Contains(normItem, normTarget) || strings.Contains(normTarget, normItem) {
			partial = append(partial, item)
		}
	}
	if len(partial) == 1 {
		return partial[0], nil
	}
	if len(partial) > 1 {
		return game.Item{}, fmt.Errorf("item name %q is ambiguous; candidates: %s", name, strings.Join(itemNames(g), ", "))
	}
	return game.Item{}, fmt.Errorf("item named %q not found; known items: %s", name, strings.Join(itemNames(g), ", "))
}

func resolveNPCByName(g *game.Game, name string) (game.Character, error) {
	if npc, err := g.GetNPCByName(name); err == nil {
		return npc, nil
	}
	normTarget := normalizeLookupKey(name)
	if normTarget == "" {
		return game.Character{}, fmt.Errorf("character name is required")
	}

	var partial []game.Character
	for _, npc := range g.NPCs {
		normNPC := normalizeLookupKey(npc.Name)
		if normNPC == normTarget {
			return npc, nil
		}
		if strings.Contains(normNPC, normTarget) || strings.Contains(normTarget, normNPC) {
			partial = append(partial, npc)
		}
	}
	if len(partial) == 1 {
		return partial[0], nil
	}
	if len(partial) > 1 {
		return game.Character{}, fmt.Errorf("character name %q is ambiguous; candidates: %s", name, strings.Join(npcNames(g), ", "))
	}
	return game.Character{}, fmt.Errorf("NPC named %q not found; known NPCs: %s", name, strings.Join(npcNames(g), ", "))
}

func normalizeLookupKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	lastSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '_' || r == '\'' {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func roomNames(g *game.Game) []string {
	names := make([]string, 0, len(g.Rooms))
	for _, room := range g.Rooms {
		names = append(names, room.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return []string{"(none)"}
	}
	return names
}

func itemNames(g *game.Game) []string {
	names := make([]string, 0, len(g.Items))
	for _, item := range g.Items {
		names = append(names, item.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return []string{"(none)"}
	}
	return names
}

func npcNames(g *game.Game) []string {
	names := make([]string, 0, len(g.NPCs))
	for _, npc := range g.NPCs {
		names = append(names, npc.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return []string{"(none)"}
	}
	return names
}
