# Phases 2–4: rpg-toolkit Integration

Integrates [KirkDiggler/rpg-toolkit](https://github.com/KirkDiggler/rpg-toolkit) as a proper Go module dependency to replace the current hand-rolled game engine with a real D&D 5e rules implementation.

**License:** rpg-toolkit is GPL-3.0. We use it strictly as a backend library in a hosted service — we do not distribute binaries or source to users. GPL v3's copyleft is triggered by *distribution*, not by running software as a network service (that would require AGPL, which rpg-toolkit does not use). Therefore no open-sourcing obligation applies. We do give attribution in `go.mod` and this documentation.

**Repository:** https://github.com/KirkDiggler/rpg-toolkit
**Author:** KirkDiggler (with contributions from Claude and Copilot per the repo acknowledgments)

---

## What We Are and Are Not Doing

**We ARE:**
- Adding `github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e` (and its toolkit dependencies) to `server-v2/go.mod` as external dependencies
- Using the library's public APIs: `character.NewFromCreationData`, `combat.ResolveAttack`, `dungeon.DungeonData`, `tools/environments`, `tools/spawn`
- Attributing rpg-toolkit in `go.mod` and documentation

**We are NOT:**
- Copying source files into our repository
- Modifying the library
- Hiding or obscuring the dependency

---

## go.mod and go.work Setup

rpg-toolkit is a multi-module monorepo — each sub-package must be added separately to `server-v2/go.mod`. The module versions to import (check https://github.com/KirkDiggler/rpg-toolkit for latest tags):

```
// server-v2/go.mod additions:
require (
    github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e  v0.x.x
    github.com/KirkDiggler/rpg-toolkit/events           v0.6.2
    github.com/KirkDiggler/rpg-toolkit/dice             v0.3.2
    github.com/KirkDiggler/rpg-toolkit/core             v0.10.0
    github.com/KirkDiggler/rpg-toolkit/tools/spatial    v0.4.0
    github.com/KirkDiggler/rpg-toolkit/tools/environments v0.4.0
    github.com/KirkDiggler/rpg-toolkit/tools/spawn      v0.x.x
    github.com/KirkDiggler/rpg-toolkit/tools/selectables v0.x.x
    github.com/KirkDiggler/rpg-toolkit/mechanics/resources v0.3.1
    github.com/KirkDiggler/rpg-toolkit/rpgerr          v0.1.1
)
```

For local development with `go.work` (if working on toolkit and server-v2 side by side):
```
// go.work at repo root (git-ignored — local dev only)
go 1.24

use (
    ./server-v2
)
// If you have a local clone of rpg-toolkit:
// use ../rpg-toolkit/rulebooks/dnd5e
// use ../rpg-toolkit/events
// etc.
```

---

## Three-Phase Structure

| Phase | Branch | What Changes |
|---|---|---|
| 2 | `feat/phase-6-character-model` | Replace `game.Character` with `dnd5e/character.Character`; SchemaVersion 3 |
| 3 | `feat/phase-6-combat-engine` | Move combat resolution from LLM to `combat.ResolveAttack`; remove damage tools |
| 4 | `feat/phase-6-world-gen` | Replace AI blueprint pipeline with `tools/environments` + `tools/spawn`; SchemaVersion 4 |

Phases 3 and 4 can be developed in parallel after Phase 2 merges.

---

## rpg-toolkit Architecture Reference

The toolkit is a multi-module Go monorepo. Each package has its own `go.mod` and is independently versioned. The three-layer architecture:

```
Game Server (server-v2)          — knows it's D&D 5e; handles storage/transport
        ↓
Rulebook (rulebooks/dnd5e)       — knows what a "fighter" is; implements rules
        ↓
RPG Toolkit (core, events, dice) — rule-agnostic: events, actions, effects, dice, spatial
```

### Key packages we import

```
github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e           — character, combat, dungeon
github.com/KirkDiggler/rpg-toolkit/events                    — EventBus (shared per session)
github.com/KirkDiggler/rpg-toolkit/dice                      — Roller interface + CryptoRoller
github.com/KirkDiggler/rpg-toolkit/tools/environments        — dungeon room generation
github.com/KirkDiggler/rpg-toolkit/tools/spawn               — monster placement
github.com/KirkDiggler/rpg-toolkit/tools/selectables         — weighted random selection
```

### Event bus lifecycle in Lambda

rpg-toolkit uses an in-memory `events.EventBus` that character conditions subscribe to. In a persistent server this bus lives for the lifetime of the game. In a stateless Lambda it must be reconstructed on every invocation:

```go
// Pattern for every Lambda handler that touches game state:
func handler(ctx context.Context, req ...) (...) {
    // 1. Load game from DynamoDB
    g, err := loadGame(ctx, sessionID)

    // 2. Reconstruct the event bus and re-subscribe all conditions
    bus := events.NewEventBus()
    gamectx := gamectx.NewGameContext(bus)
    for uid, char := range g.Players {
        char.RebindEvents(ctx, gamectx) // re-subscribes all active conditions
        g.Players[uid] = char
    }

    // 3. Do work (combat, narrative, etc.)
    // ...

    // 4. Cleanup — unsubscribe all listeners before saving
    for _, char := range g.Players {
        char.Cleanup(ctx)
    }

    // 5. Save game to DynamoDB (character.ToData() serializes conditions)
}
```

`character.Cleanup(ctx)` calls `Unsubscribe` on every event the character has subscribed to. This is critical — without it, subsequent Lambda invocations accumulate ghost subscriptions on a new bus (though since the bus is newly created each time, the real risk is a memory leak within a single long invocation).

---

## Phase 2 — Character Model Migration

**Branch:** `feat/phase-6-character-model`
**Depends on:** Phase 1 (Party Sessions) — requires `Players map[string]Character` to exist

### What changes

| Before | After |
|---|---|
| `game.Character{Health int, Equipment{6 slots}}` | `dnd5e/character.Character` — full 5e model |
| `game.Item{Equippable bool, Slot string}` | `dnd5e/weapons.Weapon`, `dnd5e/armor.Armor` |
| `SaveState.Player Character` | `SaveState.Players map[string]*character.CharacterData` (serialized via `ToData()`) |
| `SchemaVersion: 2` | `SchemaVersion: 3` |

### Available classes (constrained to implemented features)

Only three classes have mechanically implemented features in rpg-toolkit as of March 2026:

| Class | Features implemented |
|---|---|
| **Barbarian** | Rage (damage bonus, resistance, 10-round duration, RageCharges resource), Reckless Attack (advantage on attacks) |
| **Fighter** | Second Wind (1d10+level healing, bonus action), Action Surge (extra action, short rest recharge) |
| **Monk** | Flurry of Blows (2 unarmed strikes, Ki cost), Patient Defense (Dodge via Ki), Step of the Wind (Disengage/Dash via Ki) |

All other classes (Bard, Cleric, Druid, Paladin, Ranger, Rogue, Sorcerer, Warlock, Wizard) have their constants and data schema defined but zero mechanical features. Do not offer them in character creation — they would play identically to a Fighter with no features.

### Available races (all 9 PHB races)

All races have complete data (ability score increases, speed, starting languages, proficiency choices). Special traits (Darkvision, Halfling Lucky, Gnome Cunning, etc.) are **data-only** — they show in the character sheet but are not mechanically wired into the event system yet.

| Race | Subraces |
|---|---|
| Human | None (Variant Human not implemented) |
| Dwarf | Hill Dwarf (+1 WIS, 1d4 HP/level), Mountain Dwarf (+2 STR, medium armor prof) |
| Elf | High Elf (+1 INT), Wood Elf (+1 WIS, 35ft speed) |
| Halfling | Lightfoot (+1 CHA), Stout (+1 CON, poison resistance) |
| Dragonborn | None |
| Gnome | Forest Gnome (+1 DEX), Rock Gnome (+1 CON) |
| Half-Elf | None (choice-based +1 to two ability scores) |
| Half-Orc | None |
| Tiefling | None |

### Character creation data flow

`POST /api/games` receives `CharacterCreationData` (new struct replacing `AdventureCreationParams`):

```go
// server-v2/internal/game/creation.go (new file)
type CharacterCreationData struct {
    // Character identity
    Name       string `json:"name"`
    RaceID     string `json:"race_id"`      // e.g. "dwarf"
    SubraceID  string `json:"subrace_id"`   // e.g. "hill_dwarf" (empty if none)
    ClassID    string `json:"class_id"`     // "barbarian" | "fighter" | "monk"

    // Ability scores (standard array or point buy — client resolves before sending)
    AbilityScores map[string]int `json:"ability_scores"`
    // e.g. {"strength": 15, "dexterity": 14, "constitution": 13, ...}

    // Skill selections (pick N from class list — N varies by class)
    SelectedSkills []string `json:"selected_skills"`
    // e.g. ["athletics", "intimidation"]

    // World preferences (used by Phase 4 world gen prompt — optional)
    ThemeHint   string   `json:"theme_hint,omitempty"`
    Preferences []string `json:"preferences,omitempty"`
}
```

In `http-games POST`, build the rpg-toolkit `character.CreationData` from this input:

```go
import (
    dnd5echar "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/races"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/shared"
)

func buildCharacterFromCreationData(input CharacterCreationData) (*dnd5echar.Character, error) {
    raceData, ok := races.Data[input.RaceID]
    if !ok {
        return nil, fmt.Errorf("unknown race: %s", input.RaceID)
    }

    classData, err := loadClassData(input.ClassID) // see Class Data section below
    if err != nil {
        return nil, err
    }

    bgData := defaultBackground() // use "Folk Hero" as default for now

    scores := shared.AbilityScores{
        shared.Strength:     input.AbilityScores["strength"],
        shared.Dexterity:    input.AbilityScores["dexterity"],
        shared.Constitution: input.AbilityScores["constitution"],
        shared.Intelligence: input.AbilityScores["intelligence"],
        shared.Wisdom:       input.AbilityScores["wisdom"],
        shared.Charisma:     input.AbilityScores["charisma"],
    }

    char, err := dnd5echar.NewFromCreationData(dnd5echar.CreationData{
        ID:            uuid.New().String(),
        Name:          input.Name,
        RaceData:      raceData,
        SubraceID:     input.SubraceID,
        ClassData:     classData,
        BackgroundData: bgData,
        AbilityScores: scores,
        Choices: map[string]any{
            "skills": input.SelectedSkills,
        },
    })
    return char, err
}
```

### Class data — we must provide this

rpg-toolkit ships the `class.Data` **schema** but no pre-populated instances. We write these once in `server-v2/internal/game/classdata.go`:

```go
// server-v2/internal/game/classdata.go
package game

import (
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/class"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/classes"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/proficiencies"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/skills"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/abilities"
)

var ClassDataMap = map[string]class.Data{
    "barbarian": {
        ID:          classes.Barbarian,
        HitDie:      12,
        PrimaryAbility: abilities.Strength,
        SavingThrows: []abilities.Ability{abilities.Strength, abilities.Constitution},
        ArmorProficiencies: []proficiencies.ArmorProficiency{
            proficiencies.LightArmor, proficiencies.MediumArmor, proficiencies.Shields,
        },
        WeaponProficiencies: []proficiencies.WeaponProficiency{
            proficiencies.SimpleWeapons, proficiencies.MartialWeapons,
        },
        SkillChoices: class.SkillChoices{
            Count: 2,
            From: []skills.Skill{
                skills.AnimalHandling, skills.Athletics, skills.Intimidation,
                skills.Nature, skills.Perception, skills.Survival,
            },
        },
        // Starting equipment handled by packs — Barbarian gets Explorer's Pack + handaxes
    },
    "fighter": {
        ID:          classes.Fighter,
        HitDie:      10,
        PrimaryAbility: abilities.Strength, // or Dexterity
        SavingThrows: []abilities.Ability{abilities.Strength, abilities.Constitution},
        ArmorProficiencies: []proficiencies.ArmorProficiency{
            proficiencies.LightArmor, proficiencies.MediumArmor, proficiencies.HeavyArmor,
            proficiencies.Shields,
        },
        WeaponProficiencies: []proficiencies.WeaponProficiency{
            proficiencies.SimpleWeapons, proficiencies.MartialWeapons,
        },
        SkillChoices: class.SkillChoices{
            Count: 2,
            From: []skills.Skill{
                skills.Acrobatics, skills.AnimalHandling, skills.Athletics,
                skills.History, skills.Insight, skills.Intimidation,
                skills.Perception, skills.Survival,
            },
        },
    },
    "monk": {
        ID:          classes.Monk,
        HitDie:      8,
        PrimaryAbility: abilities.Dexterity,
        SavingThrows: []abilities.Ability{abilities.Strength, abilities.Dexterity},
        ArmorProficiencies: []proficiencies.ArmorProficiency{}, // none
        WeaponProficiencies: []proficiencies.WeaponProficiency{
            proficiencies.SimpleWeapons, proficiencies.Shortswords,
        },
        SkillChoices: class.SkillChoices{
            Count: 2,
            From: []skills.Skill{
                skills.Acrobatics, skills.Athletics, skills.History,
                skills.Insight, skills.Religion, skills.Stealth,
            },
        },
    },
}

func loadClassData(classID string) (class.Data, error) {
    data, ok := ClassDataMap[classID]
    if !ok {
        return class.Data{}, fmt.Errorf("class %q not available (only barbarian, fighter, monk)", classID)
    }
    return data, nil
}
```

### SaveState serialization — SchemaVersion 3

`character.Character` serializes to/from `character.CharacterData` via `ToData()` / `LoadFromData()`. Store as a map keyed by userID in DynamoDB.

```go
// SaveState v3 — Players stored as serialized CharacterData
type SaveState struct {
    SchemaVersion int                            `dynamodbav:"schema_version"` // 3
    // ... all existing fields ...
    PlayersData   map[string]character.CharacterData `dynamodbav:"players_data"` // NEW
    // Player and Players fields from v2 still present for migration
}
```

Migration in `FromSaveState`:
```go
switch ss.SchemaVersion {
case 1:
    // Migrate: single Player → Players map with old game.Character wrapped
    // Cannot fully migrate — old health int doesn't map to 5e HP cleanly
    // Mark as needs-recreation: set a flag, client shows "Character reset needed"
    g.NeedsCharacterReset = true
case 2:
    // Migrate: game.Character map → attempt to build dnd5e characters
    // Only possible if characters have class/race set; otherwise flag for reset
    for uid, c := range ss.Players {
        if c.ClassID != "" {
            dndChar, err := rebuildCharacterFromLegacy(c)
            if err == nil {
                g.DnDPlayers[uid] = dndChar
            } else {
                g.NeedsCharacterReset = true
            }
        }
    }
case 3:
    // Current — load directly from PlayersData
    for uid, data := range ss.PlayersData {
        char, err := character.LoadFromData(data, bus, gamectx)
        if err != nil { continue }
        g.DnDPlayers[uid] = char
    }
}
```

### Narrator context — character stats

The Narrator's system prompt must be updated to include the character's D&D stats so Claude narrates appropriately:

```go
func buildCharacterContext(char *character.Character) string {
    return fmt.Sprintf(`
CHARACTER STATS:
Name: %s | Race: %s | Class: %s | Level: %d
HP: %d/%d | AC: %d | Speed: 30ft
STR %d (%+d) | DEX %d (%+d) | CON %d (%+d)
INT %d (%+d) | WIS %d (%+d) | CHA %d (%+d)
Proficiency Bonus: +%d
Active Conditions: %s
Resources: %s
`,
        char.GetName(), char.GetRaceID(), char.GetClassID(), char.GetLevel(),
        char.GetHitPoints(), char.GetMaxHitPoints(), /* AC computed below */
        // ability scores and modifiers...
        // conditions list...
        // resources (rage charges remaining, ki points, etc.)...
    )
}
```

---

## Phase 3 — Combat Engine Wiring

**Branch:** `feat/phase-6-combat-engine`
**Depends on:** Phase 2 (character model must be in place)

### What changes

| Before | After |
|---|---|
| Engineer calls `damage_character(id, 15)` | `combat.ResolveAttack()` called in Go; result injected into Narrator prompt |
| Claude invents combat outcomes | Go dice determine outcomes; Claude narrates the result |
| `damage_character`, `heal_character`, `set_character_alive` tools | Removed from Engineer tool set |
| No monster stat blocks | `monster.Monster` with 8 named creatures |
| No initiative | `initiative.InitiativeTracker` for turn ordering |

### The EventBus + CombatantLookup pattern

Every combat operation needs a shared `EventBus` and a `CombatantLookup` that knows about all participants (players + monsters in the encounter):

```go
// server-v2/internal/combat/resolver.go (new package)
package combat

import (
    dnd5ecombat "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/combat"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/gamectx"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster"
    "github.com/KirkDiggler/rpg-toolkit/events"
)

// Encounter holds the runtime state for a combat encounter in the current room.
// Reconstructed from SaveState on each Lambda invocation (stateless).
type Encounter struct {
    Bus      events.EventBus
    GameCtx  *gamectx.GameContext
    Lookup   dnd5ecombat.CombatantLookup
    Players  map[string]*character.Character  // userID → character
    Monsters map[string]*monster.Monster      // monsterID → monster
}

func NewEncounter(players map[string]*character.Character, monsters []*monster.Monster) *Encounter {
    bus := events.NewEventBus()
    gc := gamectx.NewGameContext(bus)

    lookup := gamectx.NewBasicCombatantLookup()
    for _, p := range players {
        lookup.Add(p)
    }
    monsterMap := make(map[string]*monster.Monster)
    for _, m := range monsters {
        lookup.Add(m)
        monsterMap[m.GetID()] = m
    }

    return &Encounter{
        Bus:      bus,
        GameCtx:  gc,
        Lookup:   lookup,
        Players:  players,
        Monsters: monsterMap,
    }
}
```

### Full attack resolution flow

When a player attempts to attack (via `game_action` WebSocket route with `action: "attack"`):

```go
// ws-game-action handler — new "attack" action type
// Request payload:
type attackActionInput struct {
    AttackerID  string `json:"attacker_id"`   // player userID
    TargetID    string `json:"target_id"`     // monster ID
    WeaponRef   string `json:"weapon_ref"`    // e.g. "dnd5e/weapons/longsword"
}
```

Resolution:

```go
// 1. Build encounter from current room's live monsters
encounter := combat.NewEncounter(g.DnDPlayers, g.CurrentRoomMonsters())

// 2. Get attacker character (re-bind events for this invocation)
attacker := encounter.Players[input.AttackerID]
attacker.RebindEvents(ctx, encounter.GameCtx)

// 3. Resolve attack
result, err := dnd5ecombat.ResolveAttack(ctx, dnd5ecombat.AttackInput{
    AttackerID: input.AttackerID,
    TargetID:   input.TargetID,
    Weapon:     resolveWeaponRef(input.WeaponRef),
    EventBus:   encounter.Bus,
    Combatants: encounter.Lookup,
    GameCtx:    encounter.GameCtx,
    Roller:     dice.NewCryptoRoller(), // cryptographically random dice
})
if err != nil {
    return err
}

// 4. Apply damage to target (already done inside ResolveAttack via DamageReceivedEvent)
// result.TotalDamage is the final damage after resistance modifiers

// 5. Build combat result context string for the Narrator
combatCtx := formatCombatResult(result)
// e.g. "You attacked the Skeleton with a Longsword.
//       Attack roll: 17 (d20: 14 + STR mod: 3) vs AC 13 → HIT
//       Damage: 8 slashing (1d8: 5 + STR mod: 3)
//       Skeleton HP: 5/13 → 0. The Skeleton is defeated."

// 6. Inject into next Narrator call
g.PendingCombatContext = combatCtx

// 7. Cleanup
for _, p := range encounter.Players { p.Cleanup(ctx) }
```

### AttackResult formatting

```go
func formatCombatResult(r *dnd5ecombat.AttackResult, targetName string) string {
    hitStr := "MISS"
    damageStr := ""
    if r.Hit {
        hitStr = "HIT"
        if r.Critical {
            hitStr = "CRITICAL HIT"
        }
        damageStr = fmt.Sprintf("\nDamage: %d %s", r.TotalDamage, r.DamageType)
    }
    return fmt.Sprintf(
        "Attack roll: %d (d20: %d + bonus: %d) vs AC %d → %s%s",
        r.TotalAttack, r.AttackRoll, r.AttackBonus, r.TargetAC, hitStr, damageStr,
    )
}
```

### Narrator prompt injection

In `ws-chat`, the Narrator system prompt is extended with a `[COMBAT LOG]` section when combat results are pending:

```go
// bedrock.go NarrateStream — add combat context to system prompt
systemPrompt := baseSystemPrompt
if g.PendingCombatContext != "" {
    systemPrompt += "\n\n[COMBAT LOG - narrate these mechanical results dramatically:]\n" +
        g.PendingCombatContext
    g.PendingCombatContext = "" // consume after injecting
}
```

This keeps Claude as the storyteller — it receives the mechanical truth and narrates it vividly.

### Engineer tool changes

Remove from `tools.go`:
- `damage_character` — replaced by `ResolveAttack`
- `heal_character` — replaced by `SpendHitDice` / `LongRest` / `ShortRest` mechanics
- `set_character_alive` — replaced by HP dropping to 0 and death save system

Keep (still needed for non-combat narrative events):
- All room/item/NPC creation and placement tools
- `give_item_to_player`, `take_item_from_player`

Add (new Engineer capabilities):
- `trigger_short_rest` — calls `char.ShortRest(ctx)` on all player characters; restores Second Wind, Action Surge, Monk ki
- `trigger_long_rest` — calls `char.LongRest(ctx)` on all; restores all resources, full HP

### Monster turns

After each player action, monsters in the current room that are alive take their turn:

```go
// In ws-game-action after player action is resolved:
for _, m := range g.CurrentRoomMonsters() {
    if m.GetHitPoints() <= 0 { continue }
    
    turnResult, err := m.TakeTurn(ctx, monster.TurnInput{
        Room:       currentRoom,
        Combatants: encounter.Lookup,
        EventBus:   encounter.Bus,
        GameCtx:    encounter.GameCtx,
        Roller:     dice.NewCryptoRoller(),
    })
    if err != nil { continue }
    
    // Append monster action to combat log for Narrator
    g.PendingCombatContext += "\n" + formatMonsterTurn(m.GetName(), turnResult)
}
```

### Initiative tracking

Initiative is tracked in `SaveState` as a lightweight list — we don't need the full rpg-toolkit `TurnManager` across Lambda invocations, just the ordered list:

```go
type InitiativeEntry struct {
    CombatantID   string `dynamodbav:"combatant_id"`
    CombatantName string `dynamodbav:"combatant_name"`
    Roll          int    `dynamodbav:"roll"`
    IsPlayer      bool   `dynamodbav:"is_player"`
}

// On encounter start (first attack action when monsters are present):
func rollInitiative(players map[string]*character.Character, monsters []*monster.Monster) []InitiativeEntry {
    roller := dice.NewCryptoRoller()
    var entries []InitiativeEntry
    for uid, char := range players {
        dexMod := char.GetAbilityModifier(abilities.Dexterity)
        roll, _ := roller.Roll(20)
        entries = append(entries, InitiativeEntry{uid, char.GetName(), roll + dexMod, true})
    }
    for _, m := range monsters {
        dexMod := m.GetAbilityModifier(abilities.Dexterity)
        roll, _ := roller.Roll(20)
        entries = append(entries, InitiativeEntry{m.GetID(), m.GetName(), roll + dexMod, false})
    }
    sort.Slice(entries, func(i, j int) bool { return entries[i].Roll > entries[j].Roll })
    return entries
}
```

### Available monsters

The 8 monsters available from rpg-toolkit's `monster/monsters/` package:

| Monster | CR | Key stats |
|---|---|---|
| Giant Rat | 1/8 | HP 7, AC 12, Bite 1d4+2 |
| Skeleton | 1/4 | HP 13, AC 13 (armor), Shortsword 1d6+2 or Shortbow 1d6+2 |
| Zombie | 1/4 | HP 22, AC 8, Slam 1d6+1, Undead Fortitude |
| Wolf | 1/4 | HP 11, AC 13, Bite 2d4+2, Pack Tactics |
| Goblin | 1/4 | HP 7, AC 15, Scimitar 1d6+2 |
| Bandit | 1/8 | HP 11, AC 12, Scimitar 1d6+2 or Hand Crossbow 1d6+2 |
| Ghoul | 1 | HP 22, AC 12, Claws + paralysis, Bite |
| Brown Bear | 1 | HP 34, AC 11, Multiattack (claw + bite) |
| Thug | 1/2 | HP 32, AC 11, Club or Heavy Crossbow |

Reference: `github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster/monsters/`

Instantiation:
```go
import "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster/monsters"

goblin := monsters.NewGoblin(uuid.New().String())
skeleton := monsters.NewSkeleton(uuid.New().String())
// etc.
```

---

## Phase 4 — Procedural World Generation

**Branch:** `feat/phase-6-world-gen`
**Depends on:** Phase 2 (DungeonData replaces game.Area map)

### What changes

| Before | After |
|---|---|
| `GenerateBlueprint` — Claude Sonnet designs the entire world | `tools/environments` generates room layout procedurally |
| `BuildWorldFromBlueprint` — Go builds rooms from blueprint JSON | `tools/spawn` places monsters using constraints |
| Single Claude call writes rooms, items, NPCs, layout | Single Claude call receives dungeon summary → writes narrative framing only |
| `game.Area` / `map[string]Area` | `dungeon.DungeonData` |
| `SchemaVersion: 3` | `SchemaVersion: 4` |

### rpg-toolkit environments + spawn overview

```
tools/environments — generates an EnvironmentData:
    EnvironmentData.Rooms  []ZoneData       — rooms with ID, name, type
    EnvironmentData.Passages []PassageData  — connections between rooms

tools/spawn — populates rooms with entities:
    SpawnEngine.Spawn(SpawnInput) — places entities with constraints
    Constraints: AvoidCenter, MinDistanceFromEntry, NearFeature, MaxDensity, etc.

tools/selectables — weighted random selection:
    Used by spawn to choose which monster type appears where
```

### New world generation pipeline

Replace `world-gen/main.go`'s `GenerateBlueprint` + `BuildWorldFromBlueprint` calls:

```
Old pipeline:
  GenerateBlueprint (Claude Sonnet) → WorldBlueprint JSON
  BuildWorldFromBlueprint (Go) → game rooms + items + NPCs

New pipeline:
  1. GenerateDungeonLayout (tools/environments) → EnvironmentData
  2. PopulateEncounters (tools/spawn) → monsters placed per room
  3. PlaceItems (deterministic Go) → key items placed in boss/treasure rooms
  4. GenerateNarrativeFraming (Claude Sonnet, single call) → Title + Theme + QuestGoal + OpeningScene
  5. BuildDungeonData → dungeon.DungeonData
```

### Step 1: Generate dungeon layout

```go
import (
    "github.com/KirkDiggler/rpg-toolkit/tools/environments"
)

func generateDungeonLayout(seed int64, size string) (*environments.EnvironmentData, error) {
    gen := environments.NewGenerator(environments.GeneratorConfig{
        Seed:      seed,
        RoomCount: roomCountForSize(size), // "small"=6, "medium"=8, "large"=10
        // Room type distribution: entrance(1), chambers(N-3), boss(1), treasure(1)
    })
    return gen.Generate()
}
```

### Step 2: Populate encounters with spawn engine

```go
import (
    "github.com/KirkDiggler/rpg-toolkit/tools/spawn"
    "github.com/KirkDiggler/rpg-toolkit/tools/selectables"
    "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/monster/monsters"
)

func populateEncounters(env *environments.EnvironmentData, seed int64) map[string][]*monster.Monster {
    rng := rand.New(rand.NewSource(seed))
    roomMonsters := make(map[string][]*monster.Monster)

    // Build a weighted selector for low-CR monsters (rooms 1-N-1)
    lowCRSelector := selectables.NewWeighted([]selectables.Entry{
        {Weight: 3, Value: "skeleton"},
        {Weight: 3, Value: "zombie"},
        {Weight: 2, Value: "goblin"},
        {Weight: 2, Value: "wolf"},
        {Weight: 1, Value: "ghoul"},
        {Weight: 2, Value: "giant_rat"},
        {Weight: 2, Value: "bandit"},
    })

    for _, room := range env.Rooms {
        switch room.Type {
        case environments.RoomTypeEntrance:
            // No monsters in entrance room
        case environments.RoomTypeBoss:
            // Boss room: 1 Brown Bear + 2 Ghouls
            roomMonsters[room.ID] = []*monster.Monster{
                monsters.NewBrownBear(uuid.New().String()),
                monsters.NewGhoul(uuid.New().String()),
                monsters.NewGhoul(uuid.New().String()),
            }
        case environments.RoomTypeTreasure:
            // Treasure room: 1 guardian (Thug or Skeleton)
            roomMonsters[room.ID] = []*monster.Monster{
                spawnFromType(pickRandom(rng, "thug", "skeleton")),
            }
        default:
            // Standard chamber: 1-3 low-CR monsters
            count := 1 + rng.Intn(3)
            var ms []*monster.Monster
            for i := 0; i < count; i++ {
                monsterType := lowCRSelector.Select(rng).(string)
                ms = append(ms, spawnFromType(monsterType))
            }
            roomMonsters[room.ID] = ms
        }
    }
    return roomMonsters
}
```

### Step 3: Generate narrative framing (Claude call)

Build a structured dungeon summary and send to Claude Sonnet for narrative framing. This is a single non-streaming `Converse` call — same pattern as the old `GenerateBlueprint` but much narrower scope.

```go
func generateNarrativeFraming(env *environments.EnvironmentData, roomMonsters map[string][]*monster.Monster, creationData CharacterCreationData) (*NarrativeFraming, error) {
    summary := buildDungeonSummary(env, roomMonsters)
    // e.g.:
    // "Dungeon layout: 8 rooms
    //  Entrance: The Stone Gate (entrance chamber, no enemies)
    //  Room 2: The Fungal Cavern (standard chamber, 2 Skeletons)
    //  Room 3: The Drowning Pool (standard chamber, 1 Ghoul, 1 Zombie)
    //  ...
    //  Room 7: The Vault (treasure room, guarded by 1 Thug)
    //  Room 8: The Throne of Bones (boss room, 1 Brown Bear, 2 Ghouls)
    //  
    //  Player: Kira the Barbarian (Half-Orc, level 1)"

    prompt := fmt.Sprintf(`You are creating the narrative framing for a D&D 5e dungeon.
Given this procedurally generated dungeon layout, write:
1. A evocative dungeon TITLE (4-8 words)
2. A dark fantasy THEME (1 sentence)  
3. A QUEST GOAL that fits the dungeon's structure (1-2 sentences)
4. An OPENING SCENE narrative (2-3 paragraphs) that establishes the atmosphere and hooks the player

Dungeon layout:
%s

Player character: %s the %s (race: %s, class: %s)
%s

Respond in JSON: {"title":"...","theme":"...","quest_goal":"...","opening_scene":"..."}`,
        summary,
        creationData.Name, creationData.ClassID, creationData.RaceID, creationData.ClassID,
        themeHintSection(creationData.ThemeHint),
    )

    // Single Converse call — same pattern as old GenerateBlueprint
    result, err := bedrockClient.Converse(ctx, ModelNarrator, prompt, 1024, 0.8)
    // Parse JSON from result...
}
```

### Step 4: Build DungeonData

```go
func buildDungeonData(env *environments.EnvironmentData, roomMonsters map[string][]*monster.Monster, framing *NarrativeFraming) *dungeon.DungeonData {
    // Convert EnvironmentData rooms → dungeon.RoomData
    rooms := make(map[string]*dungeon.RoomData)
    for _, zone := range env.Rooms {
        rooms[zone.ID] = &dungeon.RoomData{
            ID:          zone.ID,
            Name:        generateRoomName(zone.Type, framing.Theme), // deterministic name gen
            Description: generateRoomDesc(zone.Type, framing.Theme),
            Type:        mapRoomType(zone.Type),
            Monsters:    serializeMonsters(roomMonsters[zone.ID]),
        }
    }

    return &dungeon.DungeonData{
        ID:           uuid.New().String(),
        StartRoomID:  env.EntranceRoomID(),
        BossRoomID:   env.BossRoomID(),
        CurrentRoomID: env.EntranceRoomID(),
        Rooms:        rooms,
        RevealedRooms: map[string]bool{env.EntranceRoomID(): true},
        Seed:         env.Seed,
        State:        dungeon.DungeonStateActive,
        CreatedAt:    time.Now(),
        // Title and QuestGoal stored at Game level, not DungeonData level
    }
}
```

### DungeonData in SaveState (SchemaVersion 4)

```go
type SaveState struct {
    SchemaVersion int                      `dynamodbav:"schema_version"` // 4
    // ... all existing fields ...
    DungeonData   *dungeon.DungeonData     `dynamodbav:"dungeon_data,omitempty"` // NEW
    // Rooms []Area still present for v2/v3 migration reading
}
```

Migration: v1/v2/v3 records with `Rooms []Area` are converted to a basic `DungeonData` on load (rooms without encounter data, no boss room designation). Players keep their characters but encounter the world as a "cleared" dungeon (no monsters). This is acceptable — old games become legacy story sessions.

### Room name generation

Without an AI blueprint, room names must come from somewhere. Two options:
1. **Themed name tables** — deterministic lookup by `(roomType, themeKeyword)`. Simple, fast, no tokens.
2. **Claude names rooms** — include room names in the narrative framing prompt above (cheapest to bundle in).

Recommended: option 2 — have the `generateNarrativeFraming` call also return `room_names: {"room_id": "The Fungal Cavern", ...}`. Adds ~100 tokens but keeps all narrative in one call.

### world-gen WS progress log updates

The `world_gen_log` frames shown in the `WorldGenTerminal` update to reflect the new pipeline:

```
"Generating dungeon layout..."         ← environments.Generate()
"Placing encounters..."                ← spawn engine
"Rolling initiative for room bosses..."
"Generating narrative..."              ← Claude call
"Building world..."                    ← BuildDungeonData()
"World ready."                         ← world_gen_ready frame
```

---

## Phase 6 — Cleanup

**Branch:** `feat/phase-6-cleanup` (last branch, after all rpg-toolkit work merged)

No GPL compliance obligations apply — see license note at the top of this document. This phase is purely removing obsolete code.

### Required changes

1. **Remove obsolete AI tools and prompts:**
   - Remove `damage_character`, `heal_character`, `set_character_alive` from `tools.go`
   - Remove `GenerateBlueprint`, `BuildWorldFromBlueprint` from `bedrock.go`
   - Clean up old system prompt sections that describe the Engineer's combat inference role

2. **Add rpg-toolkit attribution comment to `server-v2/go.mod`** — a single comment line is sufficient:
   ```
   // rpg-toolkit by KirkDiggler (https://github.com/KirkDiggler/rpg-toolkit), GPL-3.0
   ```

3. **Update `CLAUDE.md`** to reflect the completed Phase 6 state.

---

## Testing Notes

### Phase 2 — character model

```go
// Test buildCharacterFromCreationData for each class
// Test SchemaVersion migration: v1 → NeedsCharacterReset flag
// Test SchemaVersion migration: v3 load → character.LoadFromData round-trip
// Test RebindEvents + Cleanup don't panic on fresh bus
t.Setenv("SESSIONS_TABLE", "test-sessions")
```

### Phase 3 — combat

```go
// Test ResolveAttack with mock dice roller (use dice.MockRoller from rpg-toolkit)
// Test Barbarian Rage: verify damage bonus applied in chain
// Test Fighter Second Wind: verify HP restored, resource consumed
// Test monster TakeTurn: verify it targets a player character
// Test formatCombatResult produces expected string for Narrator
```

### Phase 4 — world gen

```go
// Test generateDungeonLayout produces correct room count
// Test populateEncounters: entrance has no monsters, boss room has boss monsters
// Test generateNarrativeFraming: mock Bedrock call, verify JSON parsing
// Test buildDungeonData: all rooms have names, connections are bidirectional
// Test world_gen_log frames broadcast to all party connections
```
