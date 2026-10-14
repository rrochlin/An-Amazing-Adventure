# Phase 5 — Frontend D&D Character Creation Wizard

Replaces the current 4-question adventure creation form with a full D&D 5e character sheet wizard. Also adds the party join flow for players entering an existing session via invite.

**Branch:** `feat/phase-6-frontend-wizard`
**Depends on:** Phase 2 backend (character model migration, `CharacterCreationData` endpoint)

---

## Current State (what we're replacing)

`client/src/routes/create.tsx` currently presents:
1. Character name (text)
2. Character description (text)
3. Adventure theme hint (text)
4. Preferences (multi-select chips)

This maps to `AdventureCreationParams` in `POST /api/games`. After Phase 2, the backend expects `CharacterCreationData` with race, class, ability scores, and skill selections.

---

## Two Wizard Modes

| Mode | URL | When |
|---|---|---|
| **New game** | `/create` | Owner creating a new session + character |
| **Join game** | `/create?session={sessionId}` | Party member joining an existing session |

In join mode:
- Steps 1–4 (character sheet) are identical
- The "Adventure Theme" / world preferences step is **skipped** (world already exists)
- Submit calls `POST /api/games/{sessionId}/join-character` instead of `POST /api/games`
- No world-gen loading screen — navigates directly to the game on success

---

## Wizard Steps

```
Step 1: Name + Backstory
Step 2: Race Selection
Step 3: Class Selection  
Step 4: Ability Scores
Step 5: Skill Selection
Step 6: Adventure Preferences  ← new game only; skipped in join mode
Step 7: Review + Confirm
```

---

## Step 1 — Name + Backstory

Simple text inputs, same as today. Backstory is optional (2–3 sentence textarea).

```tsx
// Fields:
// name: string (required, 2–40 chars)
// backstory: string (optional, max 300 chars)
```

---

## Step 2 — Race Selection

Nine PHB race cards in a grid (3×3). Each card shows:
- Race name
- Ability score increases (e.g. "+2 STR, +1 CON")
- Speed and size
- One notable trait name (e.g. "Darkvision", "Lucky", "Relentless Endurance")

On selection, if the race has subraces, a secondary row of subrace chips appears below the grid.

```tsx
interface RaceCardData {
  id: string           // "dwarf", "elf", etc.
  name: string
  asi: string          // "+2 STR, +1 CON"
  speed: string        // "25 ft"
  traits: string[]     // ["Darkvision", "Dwarven Resilience", "Stonecunning"]
  subraces: {
    id: string
    name: string
    asiBonusLabel: string  // "+1 WIS" for Hill Dwarf
  }[]
}

const RACES: RaceCardData[] = [
  {
    id: "human", name: "Human",
    asi: "+1 to all ability scores", speed: "30 ft",
    traits: ["Extra Language", "Versatile"],
    subraces: []
  },
  {
    id: "dwarf", name: "Dwarf",
    asi: "+2 CON", speed: "25 ft",
    traits: ["Darkvision", "Dwarven Resilience", "Stonecunning"],
    subraces: [
      { id: "hill_dwarf", name: "Hill Dwarf", asiBonusLabel: "+1 WIS" },
      { id: "mountain_dwarf", name: "Mountain Dwarf", asiBonusLabel: "+2 STR" },
    ]
  },
  {
    id: "elf", name: "Elf",
    asi: "+2 DEX", speed: "30 ft",
    traits: ["Darkvision", "Fey Ancestry", "Trance"],
    subraces: [
      { id: "high_elf", name: "High Elf", asiBonusLabel: "+1 INT" },
      { id: "wood_elf", name: "Wood Elf", asiBonusLabel: "+1 WIS, 35 ft speed" },
    ]
  },
  {
    id: "halfling", name: "Halfling",
    asi: "+2 DEX", speed: "25 ft",
    traits: ["Lucky", "Brave", "Halfling Nimbleness"],
    subraces: [
      { id: "lightfoot", name: "Lightfoot", asiBonusLabel: "+1 CHA" },
      { id: "stout", name: "Stout", asiBonusLabel: "+1 CON, poison resistance" },
    ]
  },
  {
    id: "dragonborn", name: "Dragonborn",
    asi: "+2 STR, +1 CHA", speed: "30 ft",
    traits: ["Draconic Ancestry", "Breath Weapon", "Damage Resistance"],
    subraces: []
  },
  {
    id: "gnome", name: "Gnome",
    asi: "+2 INT", speed: "25 ft",
    traits: ["Darkvision", "Gnome Cunning"],
    subraces: [
      { id: "forest_gnome", name: "Forest Gnome", asiBonusLabel: "+1 DEX" },
      { id: "rock_gnome", name: "Rock Gnome", asiBonusLabel: "+1 CON" },
    ]
  },
  {
    id: "half_elf", name: "Half-Elf",
    asi: "+2 CHA, +1 to two others", speed: "30 ft",
    traits: ["Darkvision", "Fey Ancestry", "Skill Versatility"],
    subraces: []
  },
  {
    id: "half_orc", name: "Half-Orc",
    asi: "+2 STR, +1 CON", speed: "30 ft",
    traits: ["Darkvision", "Menacing", "Relentless Endurance", "Savage Attacks"],
    subraces: []
  },
  {
    id: "tiefling", name: "Tiefling",
    asi: "+2 CHA, +1 INT", speed: "30 ft",
    traits: ["Darkvision", "Hellish Resistance", "Infernal Legacy"],
    subraces: []
  },
]
```

---

## Step 3 — Class Selection

Three class cards (Barbarian, Fighter, Monk). Each card is larger than a race card and shows:
- Class name + icon (use MUI icons or custom SVG)
- Hit Die (d12 / d10 / d8)
- Primary ability
- Armor: what armor they can wear
- Key features (bullet list of implemented features)
- Brief flavor description

```tsx
const CLASSES: ClassCardData[] = [
  {
    id: "barbarian",
    name: "Barbarian",
    hitDie: "d12",
    primaryAbility: "Strength",
    armorNote: "Light & Medium armor, Shields",
    flavor: "A fierce warrior driven by primal rage, the Barbarian shrugs off wounds that would fell lesser fighters.",
    features: [
      "Rage — Bonus damage, resistance to physical damage, 2/day",
      "Reckless Attack — Advantage on attacks (enemies also get advantage on you)",
      "Extra Attack at level 5",
    ]
  },
  {
    id: "fighter",
    name: "Fighter",
    hitDie: "d10",
    primaryAbility: "Strength or Dexterity",
    armorNote: "All armor, Shields",
    flavor: "Master of weapons and armor, the Fighter is a versatile warrior capable of multiple attacks and tactical recovery.",
    features: [
      "Second Wind — Heal 1d10+level HP as a bonus action, 1/short rest",
      "Action Surge — Take an extra action on your turn, 1/short rest",
      "Extra Attack at level 5 (2 additional at levels 11 and 20)",
    ]
  },
  {
    id: "monk",
    name: "Monk",
    hitDie: "d8",
    primaryAbility: "Dexterity & Wisdom",
    armorNote: "No armor (Unarmored Defense)",
    flavor: "A master of martial arts, the Monk harnesses ki energy to perform extraordinary feats of speed and power.",
    features: [
      "Flurry of Blows — 2 extra unarmed strikes, costs 1 Ki point",
      "Patient Defense — Take Dodge action as bonus action, costs 1 Ki point",
      "Step of the Wind — Dash or Disengage as bonus action, costs 1 Ki point",
      "Extra Attack at level 5",
    ]
  },
]
```

---

## Step 4 — Ability Scores

**Standard Array assignment** (default): 15, 14, 13, 12, 10, 8 assigned to the six abilities. The player drags (or selects from a dropdown) each score to each ability.

Show the racial ASI bonuses applied below each score in a lighter color:
```
STR:  [15 ▼]  +2 (racial)  = 17  (+3)
DEX:  [14 ▼]              = 14  (+2)
CON:  [13 ▼]  +1 (subrace) = 14  (+2)
INT:  [10 ▼]              = 10  (+0)
WIS:  [12 ▼]              = 12  (+1)
CHA:  [ 8 ▼]              =  8  (-1)
```

The six score values (15/14/13/12/10/8) are shown as a pool. Each score can only be assigned once (remove from pool on assignment, return to pool on reassignment).

**Point Buy** (optional toggle): 27 points to spend. Point costs:
```
Score  8  9  10  11  12  13  14  15
Cost   0  1   2   3   4   5   7   9
```

Implementation: a `useAbilityScores` hook manages the assignment state.

```typescript
interface AbilityScoreState {
  assignments: Record<string, number | null>  // ability → score value | null
  pool: number[]                               // remaining unassigned scores
  mode: 'standard' | 'pointbuy'
  racialBonuses: Record<string, number>        // from selected race/subrace
}

function computedScores(state: AbilityScoreState): Record<string, number> {
  // assignment + racial bonus for each ability
}

function abilityModifier(score: number): number {
  return Math.floor((score - 10) / 2)
}
```

---

## Step 5 — Skill Selection

Show only the skills available to the selected class. Checkboxes with the ability in parentheses. The count limit is shown: "Choose 2".

```tsx
// Barbarian skills (choose 2 of 6):
const BARBARIAN_SKILLS = [
  { id: "animal_handling", label: "Animal Handling (WIS)" },
  { id: "athletics",       label: "Athletics (STR)" },
  { id: "intimidation",    label: "Intimidation (CHA)" },
  { id: "nature",          label: "Nature (INT)" },
  { id: "perception",      label: "Perception (WIS)" },
  { id: "survival",        label: "Survival (WIS)" },
]

// Fighter skills (choose 2 of 8):
const FIGHTER_SKILLS = [
  { id: "acrobatics",      label: "Acrobatics (DEX)" },
  { id: "animal_handling", label: "Animal Handling (WIS)" },
  { id: "athletics",       label: "Athletics (STR)" },
  { id: "history",         label: "History (INT)" },
  { id: "insight",         label: "Insight (WIS)" },
  { id: "intimidation",    label: "Intimidation (CHA)" },
  { id: "perception",      label: "Perception (WIS)" },
  { id: "survival",        label: "Survival (WIS)" },
]

// Monk skills (choose 2 of 6):
const MONK_SKILLS = [
  { id: "acrobatics",  label: "Acrobatics (DEX)" },
  { id: "athletics",   label: "Athletics (STR)" },
  { id: "history",     label: "History (INT)" },
  { id: "insight",     label: "Insight (WIS)" },
  { id: "religion",    label: "Religion (INT)" },
  { id: "stealth",     label: "Stealth (DEX)" },
]
```

Checking beyond the limit disables all other unchecked options until one is deselected.

---

## Step 6 — Adventure Preferences (new game only)

This step is skipped in join mode (`?session=` URL param present).

```tsx
// Identical to current create.tsx adventure fields:
// - Theme hint (text input, optional)
// - Preferences (multi-select chips: "Dark and gritty", "High magic", "Political intrigue", etc.)
```

---

## Step 7 — Review + Confirm

Full character sheet preview before submitting. Shows:
- Name, Race, Class
- Final ability scores (with racial bonuses applied) + modifiers
- HP (hit die max + CON mod at level 1)
- AC (from starting equipment + DEX mod)
- Proficiency Bonus (+2 at level 1)
- Proficient skills (from class selection + racial grants)
- Saving throw proficiencies (from class)
- Class features list
- Starting equipment (described in text)
- Adventure theme/preferences (new game mode only)

"Create Character" submit button.

---

## API Payload

```typescript
// POST /api/games (new game) or POST /api/games/{sessionId}/join-character (join)
interface CharacterCreationData {
  // Character identity
  name: string
  race_id: string        // "dwarf"
  subrace_id?: string    // "hill_dwarf"
  class_id: string       // "barbarian" | "fighter" | "monk"

  // Ability scores — raw values before racial bonus (backend applies racial ASI)
  ability_scores: {
    strength: number
    dexterity: number
    constitution: number
    intelligence: number
    wisdom: number
    charisma: number
  }

  // Selected skill proficiencies (from class list)
  selected_skills: string[]  // ["athletics", "intimidation"]

  // World preferences — only sent for new games
  theme_hint?: string
  preferences?: string[]
}
```

---

## Wizard State Management

Use a single `useWizardState` hook managing all steps:

```typescript
interface WizardState {
  step: number                         // 1–7 (or 1–6 in join mode)
  mode: 'new_game' | 'join_game'
  sessionId?: string                   // set in join mode

  // Step data
  name: string
  backstory: string
  raceId: string
  subraceId: string
  classId: string
  abilityAssignments: Record<string, number | null>
  abilityMode: 'standard' | 'pointbuy'
  selectedSkills: string[]
  themeHint: string
  preferences: string[]
}

function canAdvance(state: WizardState): boolean {
  switch (state.step) {
    case 1: return state.name.trim().length >= 2
    case 2: return !!state.raceId && (!raceHasSubraces(state.raceId) || !!state.subraceId)
    case 3: return !!state.classId
    case 4: return Object.values(state.abilityAssignments).every(v => v !== null)
    case 5: return state.selectedSkills.length === skillCountForClass(state.classId)
    default: return true
  }
}
```

---

## Route Changes

```typescript
// client/src/routes/create.tsx
export const Route = createFileRoute('/create')({
  validateSearch: (search) => ({
    session: search.session as string | undefined,  // join mode
  }),
  beforeLoad: () => {
    if (!isAuthenticated()) throw redirect({ to: '/login' })
  },
  component: CreateCharacterWizard,
})
```

The existing route file is rewritten entirely — the old `AdventureCreationParams` form is replaced by the multi-step wizard component.

---

## Join Flow Integration

When `?session={sessionId}` is present:

```typescript
// On mount: verify the session exists and user is a member
const { session } = Route.useSearch()

useEffect(() => {
  if (session) {
    api.getGame(session).then(game => {
      if (!game) navigate({ to: '/' })
      setWizardMode('join_game')
      setSessionId(session)
    })
  }
}, [session])

// On submit in join mode:
async function submitJoin(data: CharacterCreationData) {
  const result = await api.joinGameWithCharacter(sessionId!, data)
  navigate({ to: `/game-${result.session_id}` })
}
```

New API method in `client/src/services/api.game.ts`:
```typescript
joinGameWithCharacter: (sessionId: string, data: CharacterCreationData) =>
  api.post<{ session_id: string }>(`/api/games/${sessionId}/join-character`, data)
```

---

## Corresponding Backend Change

New route on `http-games`: `POST /api/games/{sessionId}/join-character`

```go
// Validates:
// 1. Session exists
// 2. Caller is a member (has membership record) but has no character yet
// 3. Party is not full (len(g.Players) < g.PartySize)
// Then:
// 4. Build dnd5e character from CharacterCreationData
// 5. Add to g.Players[callerUserID]
// 6. Save game
// Returns: { session_id }
```

---

## Testing Notes

```typescript
// Wizard navigation: can advance only when step is valid
// Standard array: each score can only be used once
// Point buy: cannot exceed 27 points
// Skill selection: cannot select more than class allows
// Race with subraces: cannot advance step 2 without subrace selected
// Review step: shows correct final ability scores (base + racial ASI)
// Join mode: adventure preferences step is not shown
// Join mode: submit calls join-character endpoint, not create-game

// Mock POST /api/games → assert payload matches CharacterCreationData shape
// Mock POST /api/games/{id}/join-character → assert redirect to game route
```

---

## Styling Notes

The wizard follows the existing MUI dark fantasy theme (Cinzel/Crimson Text, gold/purple palette). Reference the existing `create.tsx` for the card container and stepper styling.

- Race cards: 3-column grid, `Card` with `onClick` border highlight, gold border on selection
- Class cards: larger single-row layout, MUI `Card` with expanded description panel
- Ability scores: custom drag-and-drop or select-based assignment UI — MUI `Select` dropdowns per ability is the simpler option
- Review step: styled like a D&D character sheet (bordered sections, fantasy typography)
- Step indicator: MUI `Stepper` with step labels, horizontal on desktop, vertical collapse on mobile
