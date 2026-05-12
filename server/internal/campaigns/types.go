package campaigns

import "io/fs"

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
	StartNodeID       string                       `json:"start_node_id"`
	IntroSequence     IntroSequence                `json:"intro_sequence,omitempty"`

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
}

type PlayerRoleTemplate struct {
	Name        string `json:"name,omitempty"`
	Premise     string `json:"premise,omitempty"`
	RoleSummary string `json:"role_summary,omitempty"`
}

type PartyTemplate struct {
	MaxHumanPlayers int      `json:"max_human_players"`
	StartingRoomID  string   `json:"starting_room_id"`
	CompanionIDs    []string `json:"companion_ids,omitempty"`
}

type CampaignMap struct {
	Rooms     map[string]CampaignRoom `json:"rooms"`
	Entrances []EntranceOption        `json:"entrances,omitempty"`
}

type CampaignRoom struct {
	Name        string            `json:"name"`
	Floor       int               `json:"floor,omitempty"`
	Type        string            `json:"type,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Connections map[string]string `json:"connections,omitempty"`
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
}

type DialogueAsset struct {
	Format  string `json:"format"`
	Program string `json:"program"`
	Strings string `json:"strings"`
}

type StoryNode struct {
	Mode              string                 `json:"mode"`
	DialogueAssetID   string                 `json:"dialogue_asset_id,omitempty"`
	DialogueStartNode string                 `json:"dialogue_start_node,omitempty"`
	Title             string                 `json:"title"`
	Summary           string                 `json:"summary,omitempty"`
	ObjectiveText     string                 `json:"objective_text,omitempty"`
	OnEnter           []Action               `json:"on_enter,omitempty"`
	CompletionRules   []TransitionRule       `json:"completion_rules,omitempty"`
	FailureRules      []TransitionRule       `json:"failure_rules,omitempty"`
	NarratorContract  NodeNarratorContract   `json:"narrator_contract,omitempty"`
	AllowedConditions []AllowedConditionSpec `json:"allowed_conditions,omitempty"`
}

type TransitionRule struct {
	Conditions []Condition `json:"conditions"`
	NextNodeID string      `json:"next_node_id"`
	Actions    []Action    `json:"actions,omitempty"`
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
}

type Manifest struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Title       string `json:"title"`
	Premise     string `json:"premise"`
	Description string `json:"description,omitempty"`
	Tone        string `json:"tone,omitempty"`
}
