// ws-game-action handles direct player actions that mutate game state without AI:
// move, pick_up, drop, equip, unequip, attack.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	rpgevents "github.com/KirkDiggler/rpg-toolkit/events"
	dnd5echar "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster/actions"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rrochlin/an-amazing-adventure/internal/combat"
	"github.com/rrochlin/an-amazing-adventure/internal/db"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/wsutil"
)

type actionRequest struct {
	Action    string `json:"action"`
	SubAction string `json:"sub_action"` // "move" | "pick_up" | "drop" | "equip" | "unequip" | "attack"
	Payload   string `json:"payload"`    // direction, item name, or target monster ID
	// WeaponID is optional — used only for "attack" sub_action.
	// If empty the character's equipped main-hand weapon is used.
	WeaponID string `json:"weapon_id,omitempty"`
}

func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connID := req.RequestContext.ConnectionID
	reqID := req.RequestContext.RequestID

	var msg actionRequest
	if err := json.Unmarshal([]byte(req.Body), &msg); err != nil {
		log.Printf("ws-game-action: bad body conn=%s: %v", connID, err)
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	log.Printf("ws-game-action: conn=%s req=%s action=%s payload=%q", connID, reqID, msg.SubAction, msg.Payload)

	dbClient, err := db.New(ctx)
	if err != nil {
		log.Printf("ws-game-action: db init conn=%s: %v", connID, err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	conn, err := dbClient.GetConnection(ctx, connID)
	if err != nil {
		log.Printf("ws-game-action: get connection conn=%s: %v", connID, err)
		return events.APIGatewayProxyResponse{StatusCode: 410}, nil
	}
	userID := string(conn.UserID)

	if conn.Streaming {
		ws, _ := wsutil.New(ctx)
		_ = ws.Send(ctx, connID, wsutil.Frame{Type: wsutil.FrameStreamingBlocked})
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	saveState, err := dbClient.GetGame(ctx, conn.GameID)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 404}, nil
	}

	g, err := game.FromSaveState(saveState)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Load D&D characters for this invocation (binds a fresh event bus)
	if saveState.PlayersData != nil {
		if _, loadErr := g.LoadDnDCharacters(ctx, saveState.PlayersData); loadErr != nil {
			log.Printf("ws-game-action: LoadDnDCharacters (non-fatal): %v", loadErr)
		}
	}

	ws, err := wsutil.New(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Execute the action
	var actionErr error
	switch msg.SubAction {
	case "move":
		dest, moveErr := g.MovePlayer(msg.Payload)
		if moveErr == nil && g.DungeonData != nil {
			// Persist fog-of-war: mark the destination room as revealed.
			if g.DungeonData.RevealedRooms == nil {
				g.DungeonData.RevealedRooms = make(map[string]bool)
			}
			g.DungeonData.RevealedRooms[dest.ID] = true
		}
		actionErr = moveErr
	case "pick_up":
		item, findErr := g.GetItemByName(msg.Payload)
		if findErr != nil {
			actionErr = findErr
			break
		}
		player, _ := g.GetPlayerCharacter(userID)
		currentRoom, roomErr := g.GetRoom(player.LocationID)
		if roomErr != nil {
			actionErr = roomErr
			break
		}
		if !currentRoom.HasItem(item.ID) {
			actionErr = fmt.Errorf("item %q is not in this room", msg.Payload)
			break
		}
		_ = currentRoom.RemoveItemID(item.ID)
		g.UpdateRoom(currentRoom)
		actionErr = g.GiveItemToCharacter(item.ID, userID)
	case "drop":
		item, findErr := g.GetItemByName(msg.Payload)
		if findErr != nil {
			actionErr = findErr
			break
		}
		player, _ := g.GetPlayerCharacter(userID)
		if !player.HasItem(item.ID) {
			actionErr = fmt.Errorf("you don't have %q", msg.Payload)
			break
		}
		room, roomErr := g.GetRoom(player.LocationID)
		if roomErr != nil {
			actionErr = roomErr
			break
		}
		actionErr = g.TakeItemFromPlayer(item.ID, room.ID)
	case "equip":
		item, findErr := g.GetItemByName(msg.Payload)
		if findErr != nil {
			actionErr = findErr
			break
		}
		player, _ := g.GetPlayerCharacter(userID)
		if equipErr := player.EquipItem(item); equipErr != nil {
			actionErr = equipErr
			break
		}
		g.SetPlayerCharacter(userID, player)
	case "unequip":
		// Payload is the slot name (e.g. "head", "chest")
		slot := game.EquipmentSlot(msg.Payload)
		player, _ := g.GetPlayerCharacter(userID)
		if _, unequipErr := player.UnequipItem(slot); unequipErr != nil {
			actionErr = unequipErr
			break
		}
		g.SetPlayerCharacter(userID, player)
	case "attack":
		// Payload is the target monster ID. WeaponID is optional.
		actionErr = handleAttack(ctx, g, userID, msg.Payload, msg.WeaponID)
	default:
		actionErr = fmt.Errorf("unknown sub_action: %s", msg.SubAction)
	}

	if actionErr != nil {
		log.Printf("ws-game-action: %s: %v", msg.SubAction, actionErr)
		_ = ws.SendError(ctx, connID, actionErr.Error())
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	// Persist
	g.Version++
	saved := g.ToSaveState(saveState.Narrative, saveState.ChatHistory)
	if err := dbClient.PutGame(ctx, saved); err != nil {
		log.Printf("ws-game-action: put game: %v", err)
		_ = ws.SendError(ctx, connID, "Failed to save game state")
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	// Broadcast per-member state update to all connected party members
	allConns, _ := dbClient.GetConnectionsByGameID(ctx, conn.GameID)
	if len(allConns) == 0 {
		// Fallback: send only to the requesting connection
		stateView := g.BuildGameStateView(userID, saveState.ChatHistory)
		_ = ws.SendFullState(ctx, connID, stateView)
	} else {
		for _, gc := range allConns {
			memberUID := string(gc.UserID)
			memberView := g.BuildGameStateView(memberUID, saveState.ChatHistory)
			if sendErr := ws.SendFullState(ctx, gc.ConnectionID, memberView); sendErr != nil {
				log.Printf("ws-game-action: send state to %s: %v", gc.ConnectionID, sendErr)
				_ = dbClient.DeleteConnection(ctx, gc.ConnectionID)
			}
		}
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

// handleAttack resolves a player's attack against a monster using the rpg-toolkit
// combat engine. It updates the monster's HP in g.RoomMonsters and sets
// g.PendingCombatContext so the next ws-chat call can inject the result into
// the Narrator system prompt.
func handleAttack(ctx context.Context, g *game.Game, userID, targetMonsterID, weaponID string) error {
	if targetMonsterID == "" {
		return fmt.Errorf("attack: target monster ID is required")
	}

	// Get attacker's current room
	player, ok := g.GetPlayerCharacter(userID)
	if !ok {
		return fmt.Errorf("attack: player %s not found in game", userID)
	}
	roomID := player.LocationID
	if roomID == "" {
		return fmt.Errorf("attack: player has no location")
	}

	// Load persisted monster data for the current room
	monsterDataList := g.GetRoomMonsters(roomID)
	if len(monsterDataList) == 0 {
		return fmt.Errorf("attack: no monsters in current room")
	}

	// Rebuild live *monster.Monster objects from persisted data.
	// Use a temporary bus — the Encounter builds its own canonical bus below.
	tempBus := rpgevents.NewEventBus()
	liveMonsters := make([]*monster.Monster, 0, len(monsterDataList))
	for _, data := range monsterDataList {
		if data == nil {
			continue
		}
		m, err := monster.LoadFromData(ctx, data, tempBus)
		if err != nil {
			log.Printf("handleAttack: skip monster %s (load error: %v)", data.ID, err)
			continue
		}
		if err := actions.LoadMonsterActions(m, data.Actions); err != nil {
			log.Printf("handleAttack: skip monster %s (action load error: %v)", data.ID, err)
			continue
		}
		liveMonsters = append(liveMonsters, m)
	}

	// Verify target exists and is alive
	var targetMonster *monster.Monster
	for _, m := range liveMonsters {
		if m.GetID() == targetMonsterID {
			targetMonster = m
			break
		}
	}
	if targetMonster == nil {
		return fmt.Errorf("attack: monster %s not found or already defeated", targetMonsterID)
	}
	if !targetMonster.IsAlive() {
		return fmt.Errorf("attack: %s is already defeated", targetMonster.Name())
	}

	// Get the attacking player's DnD character
	dndChar, hasDnD := g.GetDnDCharacter(userID)
	if !hasDnD || dndChar == nil {
		return fmt.Errorf("attack: player %s has no D&D character", userID)
	}

	// Build encounter player map: all DnD players currently in this room
	encounterPlayers := make(map[string]*dnd5echar.Character)
	for uid, char := range g.Players {
		if char.LocationID == roomID {
			if c, hasDnD := g.GetDnDCharacter(uid); hasDnD && c != nil {
				encounterPlayers[uid] = c
			}
		}
	}
	// Ensure the attacker is always included
	if _, included := encounterPlayers[userID]; !included {
		encounterPlayers[userID] = dndChar
	}

	enc, err := combat.NewEncounter(ctx, encounterPlayers, liveMonsters)
	if err != nil {
		return fmt.Errorf("attack: build encounter: %w", err)
	}
	defer enc.Cleanup(ctx)

	// Roll initiative if this is the first attack in this encounter
	if len(g.InitiativeOrder) == 0 {
		g.InitiativeOrder = combat.RollInitiative(encounterPlayers, liveMonsters)
	}

	// Resolve the attack + monster counter-turns
	out, err := combat.ResolvePlayerAttack(ctx, enc, combat.AttackInput{
		AttackerID: userID,
		TargetID:   targetMonsterID,
		WeaponID:   weaponID,
	})
	if err != nil {
		return fmt.Errorf("attack: resolve: %w", err)
	}

	// Persist updated monster HP back to game state.
	// Rebuild from enc.Monsters (which have current HP after the combat round).
	updatedData := make([]*monster.Data, 0, len(monsterDataList))
	for _, orig := range monsterDataList {
		if orig == nil {
			continue
		}
		if m, inEnc := enc.Monsters[orig.ID]; inEnc {
			updatedData = append(updatedData, m.ToData())
		} else {
			updatedData = append(updatedData, orig)
		}
	}
	g.SetRoomMonsters(roomID, updatedData)

	// Set pending combat context for the Narrator
	g.PendingCombatContext = out.CombatLog

	// Clear initiative if all monsters in room are now defeated
	if !g.HasLiveMonstersInRoom(roomID) {
		g.InitiativeOrder = nil
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
