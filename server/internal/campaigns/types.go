package campaigns

import (
	"io/fs"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// CampaignDefinition is the top-level authored campaign file loaded from JSON.
// It is intentionally broader than dialogue because it owns the overall
// campaign shape: map, actors, objectives, and story graph.
type CampaignDefinition struct {
	ID                string                       `json:"id"`
	Version           string                       `json:"version"`
	Title             string                       `json:"title"`
	Premise           string                       `json:"premise"`
	Description       string                       `json:"description,omitempty"`
	Tone              string                       `json:"tone,omitempty"`
	CharacterCreation CharacterCreationRules       `json:"character_creation"`
	PlayerRole        PlayerRoleTemplate           `json:"player_role,omitempty"`
	Party             PartyTemplate                `json:"party"`
	Map               CampaignMap                  `json:"map"`
	Actors            map[string]ActorTemplate     `json:"actors"`
	Objectives        map[string]ObjectiveTemplate `json:"objectives,omitempty"`
	DialogueAssets    map[string]DialogueAsset     `json:"dialogue_assets,omitempty"`
	StoryNodes        map[string]StoryNode         `json:"story_nodes"`
	// StoryPipeline picks how turns are run: "" or "current" (narrator,
	// engineer, resolver) or "game_master" (one agent, see
	// docs/story-engine/game-master.md).
	StoryPipeline string        `json:"story_pipeline,omitempty"`
	StartNodeID   string        `json:"start_node_id"`
	IntroSequence IntroSequence `json:"intro_sequence,omitempty"`
	// Endings are checked in order when a run ends; the first whose
	// conditions all match is chosen. The last must have no conditions, so it
	// is the catch-all.
	Endings []Ending `json:"endings,omitempty"`

	SourceDir string `json:"-"`
	SourceFS  fs.FS  `json:"-"`
}

type CharacterCreationRules struct {
	UseStandardCreator   bool     `json:"use_standard_creator"`
	AllowedClasses       []string `json:"allowed_classes,omitempty"`
	AllowedRaces         []string `json:"allowed_races,omitempty"`
	AllowedSubraces      []string `json:"allowed_subraces,omitempty"`
	LockedBackgroundText string   `json:"locked_background_prompt,omitempty"`
	ThemeHintMode        string   `json:"theme_hint_mode,omitempty"`
	PreferenceMode       string   `json:"preference_mode,omitempty"`
	// Preset railroads character creation: the player only chooses a name,
	// and race, class, scores, and skills come from here.
	Preset *PresetCharacter `json:"preset,omitempty"`
}

// PresetCharacter is the fixed character a railed campaign hands the player.
type PresetCharacter struct {
	DefaultName    string         `json:"default_name,omitempty"`
	RaceID         string         `json:"race_id"`
	SubraceID      string         `json:"subrace_id,omitempty"`
	ClassID        string         `json:"class_id"`
	AbilityScores  map[string]int `json:"ability_scores"`
	SelectedSkills []string       `json:"selected_skills"`
}

type PlayerRoleTemplate struct {
	Name        string `json:"name,omitempty"`
	Premise     string `json:"premise,omitempty"`
	RoleSummary string `json:"role_summary,omitempty"`
	// Scale is the player character's height relative to an adult (see
	// game.Character.Scale); 0 means 1.
	Scale float64 `json:"scale,omitempty"`
}

type PartyTemplate struct {
	MaxHumanPlayers int      `json:"max_human_players"`
	StartingRoomID  string   `json:"starting_room_id"`
	CompanionIDs    []string `json:"companion_ids,omitempty"`
}

type CampaignMap struct {
	Rooms     map[string]CampaignRoom `json:"rooms"`
	Entrances []EntranceOption        `json:"entrances,omitempty"`
	// SceneMaps are maps of their own for scenes inside a room (an
	// interior), keyed by ID; the client draws them in place of the room.
	SceneMaps map[string]SceneMap `json:"scene_maps,omitempty"`
}

// SceneMap is an interior: a place inside one room with its own map.
type SceneMap struct {
	RoomID      string `json:"room_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CampaignRoom struct {
	Name        string            `json:"name"`
	Floor       int               `json:"floor,omitempty"`
	Type        string            `json:"type,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Connections map[string]string `json:"connections,omitempty"`
	// PointsOfInterest are notable spots inside the room, shown on the map.
	PointsOfInterest []game.PointOfInterest `json:"points_of_interest,omitempty"`
}

type EntranceOption struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	StartRoomID  string          `json:"start_room_id"`
	InitialFlags map[string]bool `json:"initial_flags,omitempty"`
}

type ActorTemplate struct {
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	Goal       string    `json:"goal,omitempty"`
	Importance string    `json:"importance,omitempty"`
	Rail       ActorRail `json:"rail,omitempty"`
	// AppearsIn limits the story nodes in which the narrator is told this
	// actor exists, so later characters aren't foreshadowed early. Empty
	// means every node.
	AppearsIn []string `json:"appears_in,omitempty"`
	// StartLocationID places the character in a room when the run begins, as a
	// real character the map and agents can see. Empty means the character
	// isn't in the world at the start (for example, bandits who arrive later).
	StartLocationID string `json:"start_location_id,omitempty"`
	// Description says who the character is (role, relationships); it is
	// what an agent playing them needs to know.
	Description string `json:"description,omitempty"`

	// Character sheet (docs/story-engine/script-format.md).
	Backstory   string   `json:"backstory,omitempty"`
	Motivations []string `json:"motivations,omitempty"`
	// Relationships maps an actor ID, or "player", to how this character
	// sees them.
	Relationships map[string]string `json:"relationships,omitempty"`
	// Knows are things the character knows (including secrets).
	Knows []string `json:"knows,omitempty"`
	// DefaultTier is the agent tier when a scene doesn't say: "actor"
	// (default) or "player".
	DefaultTier string `json:"default_tier,omitempty"`
	// Scale is height relative to an adult (see game.Character.Scale); 0
	// means 1. Older child ~0.8, small child ~0.6, giant 2+.
	Scale float64 `json:"scale,omitempty"`
}

type ActorRail struct {
	Steps []ActorRailStep `json:"steps,omitempty"`
}

type ActorRailStep struct {
	ID          string      `json:"id"`
	LocationID  string      `json:"location_id,omitempty"`
	AdvanceWhen []Condition `json:"advance_when,omitempty"`
	HoldWhen    []Condition `json:"hold_when,omitempty"`
	RevealWhen  []Condition `json:"reveal_when,omitempty"`
	OnAdvance   []Action    `json:"on_advance,omitempty"`
}

type ObjectiveTemplate struct {
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
	VisibleWhen []Condition `json:"visible_when,omitempty"`
	SuccessWhen []Condition `json:"success_when,omitempty"`
	FailWhen    []Condition `json:"fail_when,omitempty"`
	Optional    bool        `json:"optional,omitempty"`
	// OnSuccess runs when the objective completes: the deterministic side
	// of a deed (the coin it pays, the doll handed back), so it doesn't
	// depend on the narration being read correctly.
	OnSuccess []Action `json:"on_success,omitempty"`
}

type DialogueAsset struct {
	Format  string `json:"format"`
	Program string `json:"program"`
	Strings string `json:"strings"`
}

type StoryNode struct {
	Mode              string `json:"mode"`
	DialogueAssetID   string `json:"dialogue_asset_id,omitempty"`
	DialogueStartNode string `json:"dialogue_start_node,omitempty"`
	Title             string `json:"title"`
	Summary           string `json:"summary,omitempty"`
	// IntroText is authored narration shown to the player when the node is
	// entered, before any dialogue. It orients them in the new scene.
	IntroText       string           `json:"intro_text,omitempty"`
	ObjectiveText   string           `json:"objective_text,omitempty"`
	OnEnter         []Action         `json:"on_enter,omitempty"`
	CompletionRules []TransitionRule `json:"completion_rules,omitempty"`
	FailureRules    []TransitionRule `json:"failure_rules,omitempty"`
	// NarratorContract is the older name for StageDirections; the loader
	// copies it over when stage_directions is absent.
	NarratorContract  NodeNarratorContract   `json:"narrator_contract,omitempty"`
	AllowedConditions []AllowedConditionSpec `json:"allowed_conditions,omitempty"`

	// Scene script (docs/story-engine/script-format.md).
	// Control is how tightly the scene follows its script: "controlled"
	// scenes unfold as written; "bounded" scenes improvise inside their
	// stage directions' boundaries. Defaults: controlled for yarn_dialogue,
	// bounded otherwise.
	Control         string          `json:"control,omitempty"`
	StageDirections StageDirections `json:"stage_directions,omitempty"`
	// Cast lists the characters with permission to act in this scene, and
	// how. A character not in the cast doesn't act here.
	Cast []SceneCastMember `json:"cast,omitempty"`
	// Beats are the story conditions this scene can settle, each outcome
	// with its meaning written out, so the resolver (or game master) judges
	// against a definition instead of guessing from a value's name.
	Beats []Beat `json:"beats,omitempty"`
	// EndsRun ends the run when this node is entered.
	EndsRun bool `json:"ends_run,omitempty"`
	// LocationID is where the scene takes place: entering the node moves
	// the player there (and discovers it).
	LocationID string `json:"location_id,omitempty"`
	// SceneMap names the map in map.scene_maps the scene is played on (an
	// interior of its location), instead of the room's own.
	SceneMap string `json:"scene_map,omitempty"`
}

// Ending is an authored conclusion to a run (Phase 7 decision 15).
type Ending struct {
	ID                string      `json:"id"`
	Title             string      `json:"title"`
	Summary           string      `json:"summary,omitempty"`
	Conditions        []Condition `json:"conditions,omitempty"`
	DialogueAssetID   string      `json:"dialogue_asset_id,omitempty"`
	DialogueStartNode string      `json:"dialogue_start_node,omitempty"`
}

type TransitionRule struct {
	Conditions []Condition `json:"conditions"`
	NextNodeID string      `json:"next_node_id"`
	Actions    []Action    `json:"actions,omitempty"`
}

// Story pipelines a campaign can ask for.
const (
	StoryPipelineCurrent    = "current"
	StoryPipelineGameMaster = "game_master"
)

// Scene control levels.
const (
	ControlControlled = "controlled"
	ControlBounded    = "bounded"
)

// Agent tiers: how much freedom the agent playing a character has.
const (
	// TierActor enacts the character's part with little autonomy.
	TierActor = "actor"
	// TierPlayer pursues the character's motivations with autonomy within
	// reason, and can be a real obstacle.
	TierPlayer = "player"
)

// WorldEventTypes are the world changes a turn can make, by the name of the
// engineer tool that applies each (internal/ai/tools.go).
var WorldEventTypes = []string{
	"move_player", "move_character",
	"give_item_to_player", "take_item_from_player", "create_item", "place_item_in_room",
	"create_character", "set_character_scale", "set_character_alive",
	"damage_character", "heal_character",
	"update_room", "create_room",
	"trigger_short_rest", "trigger_long_rest",
}

// StageDirections tell whoever runs a scene what it is and its limits.
type StageDirections struct {
	// Focus is what the scene is about.
	Focus string `json:"focus,omitempty"`
	// Setting is where and when, and how it looks and feels.
	Setting string `json:"setting,omitempty"`
	// MustMention are details to work into the scene.
	MustMention []string `json:"must_mention,omitempty"`
	// MustHappen are events the scene has to contain, in order. A controlled
	// scene plays these as written.
	MustHappen []string `json:"must_happen,omitempty"`
	// MustNotReveal are things the scene must not reveal or hint at.
	MustNotReveal []string `json:"must_not_reveal,omitempty"`
	// Boundaries are the limits a bounded scene must not stray outside of.
	Boundaries []string `json:"boundaries,omitempty"`
	// AllowedWorldEvents names the world changes the game master may make
	// in this scene (see WorldEventTypes), on top of the control level's
	// defaults. Only here can a scene allow create_character or
	// set_character_alive: who exists and who dies is authored.
	AllowedWorldEvents []string `json:"allowed_world_events,omitempty"`
	// Tone notes.
	Tone []string `json:"tone,omitempty"`
}

// SceneCastMember gives a character permission to act in a scene.
type SceneCastMember struct {
	ActorID string `json:"actor_id"`
	// Tier overrides the character's default tier for this scene.
	Tier string `json:"tier,omitempty"`
	// Part is what the character is doing in this scene.
	Part string `json:"part,omitempty"`
	// Wants is what the character wants out of this scene.
	Wants string `json:"wants,omitempty"`
	// Knows is what the character knows here, beyond their sheet.
	Knows []string `json:"knows,omitempty"`
}

// Beat is a story condition a scene can settle, with written meanings.
type Beat struct {
	// Key is the campaign state key: a label, or a flag.
	Key string `json:"key"`
	// Kind is "label" (one of Outcomes) or "flag" (set true when Meaning
	// happens).
	Kind string `json:"kind"`
	// Description says what the beat is about.
	Description string `json:"description"`
	// Meaning is when a flag beat is set.
	Meaning string `json:"meaning,omitempty"`
	// Outcomes are a label beat's values and what each one means.
	Outcomes []BeatOutcome `json:"outcomes,omitempty"`
}

// BeatOutcome is one value of a label beat and what it means.
type BeatOutcome struct {
	Value   string `json:"value"`
	Meaning string `json:"meaning"`
}

type NodeNarratorContract struct {
	SceneFocus    string   `json:"scene_focus,omitempty"`
	MustMention   []string `json:"must_mention,omitempty"`
	MustNotReveal []string `json:"must_not_reveal,omitempty"`
	ToneNotes     []string `json:"tone_notes,omitempty"`
}

type IntroSequence struct {
	Mode           string              `json:"mode,omitempty"`
	OpeningNodeID  string              `json:"opening_node_id,omitempty"`
	OpeningText    string              `json:"opening_text,omitempty"`
	OpeningChoices map[string][]string `json:"opening_choices,omitempty"`
}

type Condition struct {
	Type        string   `json:"type"`
	Key         string   `json:"key,omitempty"`
	Value       any      `json:"value,omitempty"`
	Values      []string `json:"values,omitempty"`
	AllowedVals []string `json:"allowed_values,omitempty"`
	ActorID     string   `json:"actor_id,omitempty"`
	StepID      string   `json:"step_id,omitempty"`
	RoomID      string   `json:"room_id,omitempty"`
	ObjectiveID string   `json:"objective_id,omitempty"`
	Status      string   `json:"status,omitempty"`
}

type AllowedConditionSpec struct {
	Type          string   `json:"type"`
	Key           string   `json:"key,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	// Description and Meanings come from the scene's beats: what the
	// condition is about, and what each value (or, for a flag, "true") means.
	Description string            `json:"description,omitempty"`
	Meanings    map[string]string `json:"meanings,omitempty"`
}

type Action struct {
	Type        string `json:"type"`
	Key         string `json:"key,omitempty"`
	Value       any    `json:"value,omitempty"`
	ActorID     string `json:"actor_id,omitempty"`
	StepID      string `json:"step_id,omitempty"`
	RoomID      string `json:"room_id,omitempty"`
	ObjectiveID string `json:"objective_id,omitempty"`
	AssetID     string `json:"asset_id,omitempty"`
	StartNode   string `json:"start_node,omitempty"`
	// Item names an item for give_item (created and handed to the player)
	// and take_item (the first match taken from the player; "a|b" tries each).
	Item        string `json:"item,omitempty"`
	Description string `json:"description,omitempty"`
}

type Manifest struct {
	ID                string                    `json:"id"`
	Version           string                    `json:"version"`
	Title             string                    `json:"title"`
	Premise           string                    `json:"premise"`
	Description       string                    `json:"description,omitempty"`
	Tone              string                    `json:"tone,omitempty"`
	CharacterCreation CharacterCreationManifest `json:"character_creation"`
}

// CharacterCreationManifest tells the create wizard which steps to show.
type CharacterCreationManifest struct {
	Mode             string           `json:"mode"` // "standard" | "preset"
	AllowedRaces     []string         `json:"allowed_races,omitempty"`
	AllowedSubraces  []string         `json:"allowed_subraces,omitempty"`
	AllowedClasses   []string         `json:"allowed_classes,omitempty"`
	LockedBackground string           `json:"locked_background,omitempty"`
	Preset           *PresetCharacter `json:"preset,omitempty"`
}
