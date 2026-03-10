// Package combat provides server-side D&D 5e combat resolution.
// All mechanical outcomes (hit/miss, damage rolls, monster turns) are determined
// here using rpg-toolkit. Claude narrates the results — it no longer invents them.
package combat

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/KirkDiggler/rpg-toolkit/dice"
	"github.com/KirkDiggler/rpg-toolkit/events"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/abilities"
	dnd5echar "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
	dnd5ecombat "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/combat"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/gamectx"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster/actions"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster/monsters"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/weapons"
	"github.com/google/uuid"
)

// Encounter holds all runtime state for one combat resolution pass in a single
// Lambda invocation. It is reconstructed from SaveState on every invocation —
// the event bus does not survive across Lambda calls.
type Encounter struct {
	Bus      events.EventBus
	Registry *gamectx.CombatantRegistry
	Players  map[string]*dnd5echar.Character // userID → character
	Monsters map[string]*monster.Monster     // monsterID → monster
}

// NewEncounter builds an Encounter from loaded characters and monsters.
// It registers all combatants in the context registry and subscribes each
// monster to the new event bus.
func NewEncounter(
	ctx context.Context,
	players map[string]*dnd5echar.Character,
	monsterList []*monster.Monster,
) (*Encounter, error) {
	bus := events.NewEventBus()
	registry := gamectx.NewCombatantRegistry()

	for _, p := range players {
		registry.Add(p)
	}

	monsterMap := make(map[string]*monster.Monster, len(monsterList))
	for _, m := range monsterList {
		// Re-subscribe monster to the new bus by reloading from its data
		data := m.ToData()
		reloaded, err := monster.LoadFromData(ctx, data, bus)
		if err != nil {
			return nil, fmt.Errorf("failed to reload monster %s: %w", m.Name(), err)
		}
		// Reload actions (needed after LoadFromData — see monster package docs)
		if err := actions.LoadMonsterActions(reloaded, data.Actions); err != nil {
			return nil, fmt.Errorf("failed to reload monster actions for %s: %w", m.Name(), err)
		}
		registry.Add(reloaded)
		monsterMap[reloaded.GetID()] = reloaded
	}

	return &Encounter{
		Bus:      bus,
		Registry: registry,
		Players:  players,
		Monsters: monsterMap,
	}, nil
}

// Cleanup unsubscribes all character and monster event listeners.
// Must be called at the end of every Lambda handler that creates an Encounter.
func (e *Encounter) Cleanup(ctx context.Context) {
	for _, p := range e.Players {
		_ = p.Cleanup(ctx)
	}
	for _, m := range e.Monsters {
		_ = m.Cleanup(ctx)
	}
}

// AttackInput is the server-side input for a player attack action.
type AttackInput struct {
	AttackerID string // userID of the attacking player
	TargetID   string // monsterID of the target
	// WeaponID is optional; if empty the character's equipped main-hand weapon is used.
	WeaponID string
}

// AttackOutput is returned from ResolvePlayerAttack.
type AttackOutput struct {
	Result      *dnd5ecombat.AttackResult
	TargetName  string
	TargetHP    int
	TargetMaxHP int
	TargetDied  bool
	// CombatLog is the full formatted string for injection into the Narrator prompt.
	// It includes both the player's attack and all subsequent monster turns.
	CombatLog string
}

// ResolvePlayerAttack resolves a player's attack against a monster, then runs
// monster turns for all live monsters in the encounter.
// Returns a structured result with a formatted combat log for the Narrator.
func ResolvePlayerAttack(ctx context.Context, enc *Encounter, input AttackInput) (*AttackOutput, error) {
	attacker, ok := enc.Players[input.AttackerID]
	if !ok {
		return nil, fmt.Errorf("attacker %s not found in encounter", input.AttackerID)
	}

	target, ok := enc.Monsters[input.TargetID]
	if !ok {
		return nil, fmt.Errorf("target monster %s not found in encounter", input.TargetID)
	}

	// Resolve weapon — use provided WeaponID or fall back to main-hand equipped weapon
	weapon, err := resolveWeapon(attacker, input.WeaponID)
	if err != nil {
		return nil, fmt.Errorf("cannot determine weapon: %w", err)
	}

	// Build context with combatant registry so ResolveAttack can look up combatants
	combatCtx := dnd5ecombat.WithCombatantLookup(ctx, enc.Registry)

	result, err := dnd5ecombat.ResolveAttack(combatCtx, &dnd5ecombat.AttackInput{
		AttackerID: attacker.GetID(),
		TargetID:   target.GetID(),
		Weapon:     weapon,
		EventBus:   enc.Bus,
		Roller:     dice.NewRoller(),
	})
	if err != nil {
		return nil, fmt.Errorf("ResolveAttack failed: %w", err)
	}

	playerLog := FormatAttackResult(attacker.GetName(), target.Name(), result, target.GetHitPoints(), target.GetMaxHitPoints())

	// Run monster turns for all live monsters
	var monsterLines []string
	for _, m := range enc.Monsters {
		if !m.IsAlive() {
			continue
		}
		turnLog, err := runMonsterTurn(ctx, enc, m)
		if err != nil {
			monsterLines = append(monsterLines, fmt.Sprintf("%s's turn: (error: %v)", m.Name(), err))
			continue
		}
		if turnLog != "" {
			monsterLines = append(monsterLines, turnLog)
		}
	}

	var fullLog string
	if len(monsterLines) > 0 {
		fullLog = playerLog + "\n" + strings.Join(monsterLines, "\n")
	} else {
		fullLog = playerLog
	}

	return &AttackOutput{
		Result:      result,
		TargetName:  target.Name(),
		TargetHP:    target.GetHitPoints(),
		TargetMaxHP: target.GetMaxHitPoints(),
		TargetDied:  target.GetHitPoints() == 0,
		CombatLog:   fullLog,
	}, nil
}

// runMonsterTurn executes a single monster's turn and returns a formatted log line.
func runMonsterTurn(ctx context.Context, enc *Encounter, m *monster.Monster) (string, error) {
	// Build PerceptionData: enemies are all live players, treated as adjacent
	// (simplified — no spatial grid in current architecture)
	enemies := make([]monster.PerceivedEntity, 0, len(enc.Players))
	for _, p := range enc.Players {
		if p.GetHitPoints() > 0 {
			enemies = append(enemies, monster.PerceivedEntity{
				Entity:   p,
				Distance: 1,
				Adjacent: true,
				HP:       p.GetHitPoints(),
				AC:       p.AC(),
			})
		}
	}

	if len(enemies) == 0 {
		return "", nil // No targets — monster does nothing
	}

	economy := dnd5ecombat.NewActionEconomy()
	turnResult, err := m.TakeTurn(ctx, &monster.TurnInput{
		Bus:           enc.Bus,
		ActionEconomy: economy,
		Perception: &monster.PerceptionData{
			Enemies: enemies,
		},
		Roller: dice.NewRoller(),
		Speed:  6,
	})
	if err != nil {
		return "", err
	}

	return FormatMonsterTurn(m.Name(), turnResult, enc.Players), nil
}

// resolveWeapon determines the weapon to use for an attack.
// Priority: explicit WeaponID → main-hand equipped weapon → unarmed (1 bludgeoning).
func resolveWeapon(char *dnd5echar.Character, weaponID string) (*weapons.Weapon, error) {
	if weaponID != "" {
		w, err := weapons.GetByID(weapons.WeaponID(weaponID))
		if err != nil {
			return nil, fmt.Errorf("unknown weapon %q: %w", weaponID, err)
		}
		return &w, nil
	}

	// Try main-hand equipped slot
	equipped := char.GetEquippedSlot(dnd5echar.SlotMainHand)
	if equipped != nil {
		w := equipped.AsWeapon()
		if w != nil {
			return w, nil
		}
	}

	// Fall back to unarmed strike (improvised 1 bludgeoning)
	return &weapons.Weapon{
		ID:         "unarmed",
		Name:       "Unarmed Strike",
		Damage:     "1",
		DamageType: "bludgeoning",
	}, nil
}

// -------------------------------------------------------------------
// Formatting helpers — produce human-readable combat log for Narrator
// -------------------------------------------------------------------

// FormatAttackResult formats a player's attack result for the Narrator system prompt.
func FormatAttackResult(attackerName, targetName string, r *dnd5ecombat.AttackResult, targetHP, targetMaxHP int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s attacks %s.\n", attackerName, targetName))

	advStr := ""
	if r.HasAdvantage {
		advStr = fmt.Sprintf(" (advantage, rolls: %v)", r.AllRolls)
	} else if r.HasDisadvantage {
		advStr = fmt.Sprintf(" (disadvantage, rolls: %v)", r.AllRolls)
	}

	sb.WriteString(fmt.Sprintf("Attack roll: %d%s + %d bonus = %d vs AC %d → ",
		r.AttackRoll, advStr, r.AttackBonus, r.TotalAttack, r.TargetAC))

	switch {
	case r.Critical:
		sb.WriteString("CRITICAL HIT!\n")
	case r.Hit:
		sb.WriteString("HIT\n")
	default:
		sb.WriteString("MISS\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Damage: %d %s (dice: %v + %d bonus)\n",
		r.TotalDamage, r.DamageType, r.DamageRolls, r.DamageBonus))
	sb.WriteString(fmt.Sprintf("%s HP: %d/%d", targetName, targetHP, targetMaxHP))
	if targetHP == 0 {
		sb.WriteString(" — DEFEATED")
	}
	sb.WriteString("\n")
	return sb.String()
}

// FormatMonsterTurn formats a monster's turn result for the Narrator system prompt.
func FormatMonsterTurn(monsterName string, result *monster.TurnResult, players map[string]*dnd5echar.Character) string {
	if result == nil || len(result.Actions) == 0 {
		return fmt.Sprintf("%s takes no action.", monsterName)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s's turn:\n", monsterName))
	for _, act := range result.Actions {
		status := "failed"
		if act.Success {
			status = "succeeded"
		}
		if act.TargetID != "" {
			targetName := act.TargetID
			for uid, p := range players {
				if uid == act.TargetID || p.GetID() == act.TargetID {
					targetName = p.GetName()
					break
				}
			}
			sb.WriteString(fmt.Sprintf("  %s %s vs %s\n", act.ActionType, status, targetName))
		} else {
			sb.WriteString(fmt.Sprintf("  %s %s\n", act.ActionType, status))
		}
	}
	return sb.String()
}

// -------------------------------------------------------------------
// Initiative helpers
// -------------------------------------------------------------------

// InitiativeEntry records one combatant's initiative roll for persistence.
type InitiativeEntry struct {
	CombatantID   string `json:"combatant_id" dynamodbav:"combatant_id"`
	CombatantName string `json:"combatant_name" dynamodbav:"combatant_name"`
	Roll          int    `json:"roll" dynamodbav:"roll"`
	IsPlayer      bool   `json:"is_player" dynamodbav:"is_player"`
}

// RollInitiative rolls initiative for all players and monsters.
// Returns entries ordered highest-first; ties resolved player-first.
func RollInitiative(players map[string]*dnd5echar.Character, monsterList []*monster.Monster) []InitiativeEntry {
	roller := dice.NewRoller()
	ctx := context.Background()
	var entries []InitiativeEntry

	for uid, char := range players {
		dexMod := char.GetAbilityModifier(abilities.DEX)
		roll, _ := roller.Roll(ctx, 20)
		entries = append(entries, InitiativeEntry{
			CombatantID:   uid,
			CombatantName: char.GetName(),
			Roll:          roll + dexMod,
			IsPlayer:      true,
		})
	}
	for _, m := range monsterList {
		scores := m.AbilityScores()
		dexMod := scores.Modifier(abilities.DEX)
		roll, _ := roller.Roll(ctx, 20)
		entries = append(entries, InitiativeEntry{
			CombatantID:   m.GetID(),
			CombatantName: m.Name(),
			Roll:          roll + dexMod,
			IsPlayer:      false,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Roll != entries[j].Roll {
			return entries[i].Roll > entries[j].Roll
		}
		return entries[i].IsPlayer && !entries[j].IsPlayer
	})
	return entries
}

// -------------------------------------------------------------------
// Monster factory helpers
// -------------------------------------------------------------------

// NewMonsterByType instantiates a fresh monster of the given type with a new UUID.
// Returns nil for unknown types.
func NewMonsterByType(monsterType string) *monster.Monster {
	id := uuid.NewString()
	switch monsterType {
	case "goblin":
		return monster.NewGoblin(id)
	case "skeleton":
		return monsters.NewSkeleton(id)
	case "zombie":
		return monsters.NewZombie(id)
	case "wolf":
		return monsters.NewWolf(id)
	case "giant_rat":
		return monsters.NewGiantRat(id)
	case "bandit":
		return monsters.NewBanditMelee(id)
	case "ghoul":
		return monsters.NewGhoul(id)
	case "brown_bear":
		return monsters.NewBrownBear(id)
	case "thug":
		return monsters.NewThug(id)
	default:
		return nil
	}
}
