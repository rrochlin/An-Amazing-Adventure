package game

import (
	"context"
	"fmt"

	"github.com/KirkDiggler/rpg-toolkit/events"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/abilities"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/backgrounds"
	dnd5echar "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character/choices"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/classes"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/fightingstyles"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/languages"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/races"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/shared"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/skills"
	"github.com/google/uuid"
)

// CharacterCreationData replaces AdventureCreationParams for D&D 5e character
// creation. All fields except the world-preference hints are required.
type CharacterCreationData struct {
	// Character identity
	Name      string `json:"name"`
	Backstory string `json:"backstory,omitempty"` // optional player-written backstory
	RaceID    string `json:"race_id"`             // e.g. "dwarf" (kebab-case, matches toolkit)
	SubraceID string `json:"subrace_id"`          // e.g. "hill-dwarf" (empty if none)
	ClassID   string `json:"class_id"`            // "barbarian" | "fighter" | "monk"

	// Ability scores (standard array or point buy — client resolves before sending).
	// Keys must be the short form: "str", "dex", "con", "int", "wis", "cha"
	AbilityScores map[string]int `json:"ability_scores"`

	// Skill selections (pick N from class list — N varies by class)
	SelectedSkills []string `json:"selected_skills"` // e.g. ["athletics", "intimidation"]

	// World preferences (used by world-gen prompt — optional)
	ThemeHint   string   `json:"theme_hint,omitempty"`
	Preferences []string `json:"preferences,omitempty"`
}

// SupportedClasses lists the only classes with mechanically implemented
// features in rpg-toolkit as of the current version.
var SupportedClasses = []string{classes.Barbarian, classes.Fighter, classes.Monk}

// SupportedRaces lists races that can be fully constructed by BuildDnDCharacter.
// Half-Elf is excluded because rpg-toolkit v0.51.0 records its racial skill choice
// without a ChoiceID, so ValidateChoices() always fails for Half-Elf.
// TODO(toolkit): re-enable "half-elf" once rpg-toolkit sets ChoiceID=HalfElfSkills
// in draft.SetRace for racial skill choices.
var SupportedRaces = []string{
	string(races.Human),
	string(races.Dwarf),
	string(races.Elf),
	string(races.Halfling),
	string(races.Dragonborn),
	string(races.Gnome),
	// races.HalfElf — excluded: toolkit bug, see TODO above
	string(races.HalfOrc),
	string(races.Tiefling),
}

// BuildDnDCharacter builds a full D&D 5e character from creation data.
// The returned character has an event bus bound and is ready for use within
// a single Lambda invocation. Call char.Cleanup(ctx) before discarding.
//
// Background is always FolkHero — the only fully-mechanical background that
// does not require additional language or tool choices.
//
// ID format: race_id, subrace_id, and selected_skills must use kebab-case
// (e.g. "half-orc", "hill-dwarf", "animal-handling") matching the toolkit
// constants. The frontend is expected to send the correct format; no
// normalization is performed here so errors surface immediately.
func BuildDnDCharacter(ctx context.Context, input CharacterCreationData) (*dnd5echar.Character, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("character name is required")
	}

	// Validate class
	validClass := false
	for _, c := range SupportedClasses {
		if c == input.ClassID {
			validClass = true
			break
		}
	}
	if !validClass {
		return nil, fmt.Errorf("class %q is not available; supported classes: barbarian, fighter, monk", input.ClassID)
	}

	// Validate race
	validRace := false
	for _, r := range SupportedRaces {
		if r == input.RaceID {
			validRace = true
			break
		}
	}
	if !validRace {
		return nil, fmt.Errorf("race %q is not available; supported races: human, dwarf, elf, halfling, dragonborn, gnome, half-orc, tiefling", input.RaceID)
	}

	// Map ability score keys (client may send full names or short names)
	scores := make(shared.AbilityScores)
	for k, v := range input.AbilityScores {
		ab, ok := abilities.All[k]
		if !ok {
			return nil, fmt.Errorf("unknown ability score key: %q", k)
		}
		scores[ab] = v
	}
	if len(scores) != 6 {
		return nil, fmt.Errorf("all six ability scores are required (got %d)", len(scores))
	}

	// Validate selected skills
	resolvedSkills := make([]skills.Skill, 0, len(input.SelectedSkills))
	for _, s := range input.SelectedSkills {
		sk, ok := skills.All[s]
		if !ok {
			return nil, fmt.Errorf("unknown skill: %q", s)
		}
		resolvedSkills = append(resolvedSkills, sk)
	}

	// Create a fresh event bus for character construction
	bus := events.NewEventBus()
	characterID := uuid.NewString()

	draft, err := dnd5echar.NewDraft(&dnd5echar.DraftConfig{
		ID:       uuid.NewString(),
		PlayerID: characterID,
	})
	if err != nil {
		return nil, fmt.Errorf("new draft: %w", err)
	}

	if err := draft.SetName(&dnd5echar.SetNameInput{Name: input.Name}); err != nil {
		return nil, fmt.Errorf("set name: %w", err)
	}

	if err := draft.SetRace(&dnd5echar.SetRaceInput{
		RaceID:    races.Race(input.RaceID),
		SubraceID: races.Subrace(input.SubraceID),
		Choices:   defaultRaceChoices(input.RaceID),
	}); err != nil {
		return nil, fmt.Errorf("set race: %w", err)
	}

	if err := draft.SetClass(&dnd5echar.SetClassInput{
		ClassID: classes.Class(input.ClassID),
		Choices: dnd5echar.ClassChoices{
			Skills:        resolvedSkills,
			Equipment:     defaultEquipmentChoices(input.ClassID),
			FightingStyle: defaultFightingStyle(input.ClassID),
			Tools:         defaultToolChoices(input.ClassID),
		},
	}); err != nil {
		return nil, fmt.Errorf("set class: %w", err)
	}

	// Folk Hero: skills are fixed (Animal Handling + Survival) so no choices needed.
	if err := draft.SetBackground(&dnd5echar.SetBackgroundInput{
		BackgroundID: backgrounds.FolkHero,
	}); err != nil {
		return nil, fmt.Errorf("set background: %w", err)
	}

	if err := draft.SetAbilityScores(&dnd5echar.SetAbilityScoresInput{
		Scores: scores,
		Method: "standard",
	}); err != nil {
		return nil, fmt.Errorf("set ability scores: %w", err)
	}

	char, err := draft.ToCharacter(ctx, characterID, bus)
	if err != nil {
		return nil, fmt.Errorf("finalize character: %w", err)
	}

	return char, nil
}

// RebindCharacters reconstructs the event bus and re-subscribes all active
// conditions for all DnD characters in the game. Must be called on every
// Lambda invocation that touches game state. Returns the new bus.
//
// Call CleanupCharacters when done to unsubscribe all listeners.
func RebindCharacters(ctx context.Context, g *Game) events.EventBus {
	bus := events.NewEventBus()
	// Characters loaded via LoadFromData already subscribe during load;
	// for characters already in DnDPlayers (loaded in-memory), the bus is
	// embedded in the character. Since we always load from Data fresh each
	// invocation, this is a no-op placeholder for the event-bus lifecycle.
	_ = ctx
	_ = g
	return bus
}

// LoadDnDPlayers loads all DnD characters from their persisted Data forms
// and binds them to the provided event bus. Returns a map keyed by userID.
func LoadDnDPlayers(ctx context.Context, bus events.EventBus, playersData map[string]*dnd5echar.Data) (map[string]*dnd5echar.Character, error) {
	result := make(map[string]*dnd5echar.Character, len(playersData))
	for uid, data := range playersData {
		if data == nil {
			continue
		}
		char, err := dnd5echar.LoadFromData(ctx, data, bus)
		if err != nil {
			return nil, fmt.Errorf("load character for user %s: %w", uid, err)
		}
		result[uid] = char
	}
	return result, nil
}

// CleanupDnDPlayers is a no-op in Lambda. The event bus is created per
// invocation and goes out of scope when the handler returns, so there is
// no need to explicitly unsubscribe. This function is kept as a named hook
// for future use (e.g. long-running server mode with persistent buses).
func CleanupDnDPlayers(_ context.Context, _ map[string]*dnd5echar.Character) {
	// In Lambda: bus lifetime = handler lifetime — GC cleans up automatically.
}

// defaultRaceChoices returns required choices for races that need them.
// We always pick the simplest/first valid option to keep creation non-interactive.
//
// Known toolkit gaps (do not remove these TODOs):
// TODO(toolkit): compileProficiencies() in rpg-toolkit draft.go does not read
// ChoiceToolProficiency entries from d.choices — tool proficiency choices pass
// validation but are silently absent from char.toolProficiencies. Fix requires
// updating rpg-toolkit to iterate d.choices in compileProficiencies().
//
// TODO(toolkit): Half-Elf racial skill choices are recorded by SetRace without a
// ChoiceID, so ValidateChoices() always fails (validator matches by ChoiceID ==
// "half-elf-skills"). Half-Elf is excluded from SupportedRaces until this is fixed
// upstream in rpg-toolkit draft.go SetRace().
//
// TODO(toolkit): Background grants (skills, tools) are not applied to the character.
// FolkHero should grant Animal Handling + Survival skills and ToolVehicleLand.
// Fix requires rpg-toolkit to implement compileSkills() and compileProficiencies()
// background grant sections (currently marked TODO in draft.go).
func defaultRaceChoices(raceID string) dnd5echar.RaceChoices {
	switch raceID {
	case "human":
		// Human gets 1 free language — default to Elvish
		return dnd5echar.RaceChoices{
			Languages: []languages.Language{languages.Elvish},
		}
	case "dwarf":
		// Dwarf gets proficiency with 1 artisan's tool — default to smith's tools.
		// Note: this choice passes validation but due to a toolkit bug the proficiency
		// is not compiled into char.toolProficiencies (see TODO above).
		return dnd5echar.RaceChoices{
			Tools: []shared.SelectionID{"smiths-tools"},
		}
	}
	return dnd5echar.RaceChoices{}
}

// defaultEquipmentChoices returns the simplest valid equipment set for a class.
// We always pick the first option for each required equipment choice so that
// character creation succeeds without exposing equipment selection to the user
// in Phase 2. Phase 5 (Frontend Wizard) can add full equipment selection later.
func defaultEquipmentChoices(classID string) []dnd5echar.EquipmentChoiceSelection {
	switch classID {
	case classes.Barbarian:
		return []dnd5echar.EquipmentChoiceSelection{
			{ChoiceID: choices.BarbarianWeaponsPrimary, OptionID: choices.BarbarianWeaponGreataxe},
			{ChoiceID: choices.BarbarianWeaponsSecondary, OptionID: choices.BarbarianSecondaryHandaxes},
			{ChoiceID: choices.BarbarianPack, OptionID: choices.BarbarianPackExplorer},
		}
	case classes.Fighter:
		// FighterWeaponMartialShield (option a) requires a category selection (martial weapon).
		// Provide a longsword as the category selection.
		return []dnd5echar.EquipmentChoiceSelection{
			{ChoiceID: choices.FighterArmor, OptionID: choices.FighterArmorChainMail},
			{ChoiceID: choices.FighterWeaponsPrimary, OptionID: choices.FighterWeaponMartialShield,
				CategorySelections: []shared.EquipmentID{"longsword"}},
			{ChoiceID: choices.FighterWeaponsSecondary, OptionID: choices.FighterRangedCrossbow},
			{ChoiceID: choices.FighterPack, OptionID: choices.FighterPackDungeoneer},
		}
	case classes.Monk:
		return []dnd5echar.EquipmentChoiceSelection{
			{ChoiceID: choices.MonkWeaponsPrimary, OptionID: choices.MonkWeaponShortsword},
			{ChoiceID: choices.MonkPack, OptionID: choices.MonkPackDungeoneer},
		}
	}
	return nil
}

// defaultFightingStyle returns the default fighting style for classes that require one.
func defaultFightingStyle(classID string) fightingstyles.FightingStyle {
	if classID == classes.Fighter {
		return fightingstyles.Defense
	}
	return ""
}

// defaultToolChoices returns a default tool selection for classes that require one.
// Monk requires 1 artisan's tool or musical instrument — we default to "flute".
func defaultToolChoices(classID string) []shared.SelectionID {
	if classID == classes.Monk {
		return []shared.SelectionID{"flute"}
	}
	return nil
}

// BuildCharacterContext generates the D&D character stats section for the
// narrator system prompt.
func BuildCharacterContext(name string, data *dnd5echar.Data) string {
	if data == nil {
		return ""
	}

	// Compute ability modifiers
	mod := func(score int) int { return (score - 10) / 2 }
	modStr := func(score int) string {
		m := mod(score)
		if m >= 0 {
			return fmt.Sprintf("+%d", m)
		}
		return fmt.Sprintf("%d", m)
	}

	str := data.AbilityScores[abilities.STR]
	dex := data.AbilityScores[abilities.DEX]
	con := data.AbilityScores[abilities.CON]
	intel := data.AbilityScores[abilities.INT]
	wis := data.AbilityScores[abilities.WIS]
	cha := data.AbilityScores[abilities.CHA]

	return fmt.Sprintf(`CHARACTER STATS:
Name: %s | Race: %s | Class: %s | Level: %d
HP: %d/%d | AC: %d
STR %d (%s) | DEX %d (%s) | CON %d (%s)
INT %d (%s) | WIS %d (%s) | CHA %d (%s)
Proficiency Bonus: +%d`,
		data.Name, data.RaceID, data.ClassID, data.Level,
		data.HitPoints, data.MaxHitPoints, data.ArmorClass,
		str, modStr(str),
		dex, modStr(dex),
		con, modStr(con),
		intel, modStr(intel),
		wis, modStr(wis),
		cha, modStr(cha),
		data.ProficiencyBonus,
	)
}
