package campaigns

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/dialogue"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

type ConditionResolution struct {
	BoolFlags map[string]bool   `json:"bool_flags,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Counters  map[string]int    `json:"counters,omitempty"`
}

type TransitionResult struct {
	Transitioned bool
	FromNodeID   string
	ToNodeID     string
	DialogueText string
}

type TurnOrchestrationResult struct {
	Transition TransitionResult
	Events     []game.WorldEvent
}

func AllowedConditionSpecsForNode(node StoryNode) []AllowedConditionSpec {
	merged := make(map[string]AllowedConditionSpec)
	for _, spec := range node.AllowedConditions {
		mergeAllowedSpec(merged, spec)
	}
	for _, rule := range append(node.FailureRules, node.CompletionRules...) {
		for _, cond := range rule.Conditions {
			if spec, ok := allowedSpecFromCondition(cond); ok {
				mergeAllowedSpec(merged, spec)
			}
		}
	}
	out := make([]AllowedConditionSpec, 0, len(merged))
	for _, spec := range merged {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key == out[j].Key {
			return out[i].Type < out[j].Type
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func ApplyConditionResolution(state *game.CampaignRuntimeState, resolution ConditionResolution) {
	if state == nil {
		return
	}
	if state.BoolFlags == nil {
		state.BoolFlags = make(map[string]bool)
	}
	if state.Labels == nil {
		state.Labels = make(map[string]string)
	}
	if state.Counters == nil {
		state.Counters = make(map[string]int)
	}
	for key, value := range resolution.BoolFlags {
		state.BoolFlags[key] = value
	}
	for key, value := range resolution.Labels {
		if value != "" {
			state.Labels[key] = value
		}
	}
	for key, value := range resolution.Counters {
		state.Counters[key] = value
	}
}

func FilterConditionResolution(specs []AllowedConditionSpec, resolution ConditionResolution) ConditionResolution {
	if len(specs) == 0 {
		return ConditionResolution{}
	}
	allowedFlags := map[string]bool{}
	allowedLabels := map[string]map[string]bool{}
	allowedCounters := map[string]bool{}
	for _, spec := range specs {
		switch spec.Type {
		case "flag_true", "flag_false":
			allowedFlags[spec.Key] = true
		case "label_equals":
			if allowedLabels[spec.Key] == nil {
				allowedLabels[spec.Key] = map[string]bool{}
			}
			for _, value := range spec.AllowedValues {
				allowedLabels[spec.Key][value] = true
			}
		case "counter_gte", "counter_lte":
			allowedCounters[spec.Key] = true
		}
	}
	filtered := ConditionResolution{}
	for key, value := range resolution.BoolFlags {
		if !allowedFlags[key] {
			continue
		}
		if filtered.BoolFlags == nil {
			filtered.BoolFlags = make(map[string]bool)
		}
		filtered.BoolFlags[key] = value
	}
	for key, value := range resolution.Labels {
		allowedValues, ok := allowedLabels[key]
		if !ok {
			continue
		}
		if len(allowedValues) > 0 && !allowedValues[value] {
			continue
		}
		if filtered.Labels == nil {
			filtered.Labels = make(map[string]string)
		}
		filtered.Labels[key] = value
	}
	for key, value := range resolution.Counters {
		if !allowedCounters[key] {
			continue
		}
		if filtered.Counters == nil {
			filtered.Counters = make(map[string]int)
		}
		filtered.Counters[key] = value
	}
	return filtered
}

func EnterActiveNode(def *CampaignDefinition, g *game.Game) (string, error) {
	if def == nil || g == nil || g.Campaign == nil {
		return "", nil
	}
	node, ok := def.StoryNodes[g.Campaign.ActiveNodeID]
	if !ok {
		return "", fmt.Errorf("active story node %q not found", g.Campaign.ActiveNodeID)
	}
	g.Campaign.CurrentObjective = node.ObjectiveText
	if err := applyActions(def, g, node.OnEnter); err != nil {
		return "", err
	}
	switch node.Mode {
	case "ai_scene":
		g.Campaign.ActiveDialogue = nil
	case "hybrid", "yarn_dialogue":
		if g.Campaign.ActiveDialogue == nil || g.Campaign.ActiveDialogue.AssetID != node.DialogueAssetID {
			g.Campaign.ActiveDialogue = &game.DialogueRuntimeState{
				AssetID:     node.DialogueAssetID,
				CurrentNode: node.DialogueStartNode,
			}
		} else if g.Campaign.ActiveDialogue.CurrentNode == "" {
			g.Campaign.ActiveDialogue.CurrentNode = node.DialogueStartNode
		}
	}
	return currentDialogueOpening(def, g), nil
}

func AdvanceCampaign(def *CampaignDefinition, g *game.Game) (TransitionResult, error) {
	if def == nil || g == nil || g.Campaign == nil {
		return TransitionResult{}, nil
	}
	currentNode, ok := def.StoryNodes[g.Campaign.ActiveNodeID]
	if !ok {
		return TransitionResult{}, fmt.Errorf("active story node %q not found", g.Campaign.ActiveNodeID)
	}
	for _, rules := range [][]TransitionRule{currentNode.FailureRules, currentNode.CompletionRules} {
		for _, rule := range rules {
			matched, err := conditionsMatch(def, g, rule.Conditions)
			if err != nil {
				return TransitionResult{}, err
			}
			if !matched {
				continue
			}
			fromNodeID := g.Campaign.ActiveNodeID
			if err := applyActions(def, g, rule.Actions); err != nil {
				return TransitionResult{}, err
			}
			if rule.NextNodeID == "" {
				return TransitionResult{Transitioned: true, FromNodeID: fromNodeID}, nil
			}
			g.Campaign.PreviousNodeIDs = append(g.Campaign.PreviousNodeIDs, fromNodeID)
			g.Campaign.ActiveNodeID = rule.NextNodeID
			dialogueText, err := EnterActiveNode(def, g)
			if err != nil {
				return TransitionResult{}, err
			}
			return TransitionResult{
				Transitioned: true,
				FromNodeID:   fromNodeID,
				ToNodeID:     rule.NextNodeID,
				DialogueText: dialogueText,
			}, nil
		}
	}
	return TransitionResult{}, nil
}

func OrchestrateTurn(def *CampaignDefinition, g *game.Game, resolution ConditionResolution) (TurnOrchestrationResult, error) {
	if def == nil || g == nil || g.Campaign == nil {
		return TurnOrchestrationResult{}, nil
	}

	node, ok := def.StoryNodes[g.Campaign.ActiveNodeID]
	if !ok {
		return TurnOrchestrationResult{}, fmt.Errorf("active story node %q not found", g.Campaign.ActiveNodeID)
	}

	ApplyConditionResolution(g.Campaign, FilterConditionResolution(AllowedConditionSpecsForNode(node), resolution))

	transition, err := AdvanceCampaign(def, g)
	if err != nil {
		return TurnOrchestrationResult{}, err
	}

	result := TurnOrchestrationResult{Transition: transition}
	if !transition.Transitioned || transition.ToNodeID == "" {
		return result, nil
	}

	nextNode, ok := def.StoryNodes[transition.ToNodeID]
	if ok && nextNode.ObjectiveText != "" {
		result.Events = append(result.Events, game.WorldEvent{
			Type:    "campaign_progress",
			Message: "Objective updated: " + nextNode.ObjectiveText,
		})
	}

	return result, nil
}

func currentDialogueOpening(def *CampaignDefinition, g *game.Game) string {
	if def == nil || g == nil || g.Campaign == nil || g.Campaign.ActiveDialogue == nil || g.Campaign.ActiveDialogue.AssetID == "" {
		return ""
	}
	state := g.Campaign
	asset, ok := def.DialogueAssets[state.ActiveDialogue.AssetID]
	if !ok {
		return ""
	}
	result, err := dialogue.RunNode(def.SourceFS, asset.Program, asset.Strings, state.ActiveDialogue.CurrentNode, state.ActiveDialogue.Variables, nil)
	if err != nil {
		return ""
	}
	if len(result.Commands) > 0 {
		if applyErr := ApplyDialogueCommands(def, g, result.Commands); applyErr != nil {
			return ""
		}
	}
	state.ActiveDialogue.CurrentNode = result.CurrentNode
	state.ActiveDialogue.Variables = result.Variables
	if len(result.PendingChoices) > 0 {
		// Dialogue paused waiting for a player choice; surface the options.
		choices := make([]game.DialogueChoice, len(result.PendingChoices))
		for i, c := range result.PendingChoices {
			choices[i] = game.DialogueChoice{ID: c.ID, Text: c.Text}
		}
		state.ActiveDialogue.AwaitingChoice = true
		state.ActiveDialogue.PendingChoices = choices
	} else {
		state.ActiveDialogue.AwaitingChoice = false
		state.ActiveDialogue.PendingChoices = nil
	}
	if len(result.Lines) == 0 {
		return ""
	}
	return strings.Join(result.Lines, "\n")
}

// ResumeDialogueWithChoice resumes a paused Yarn dialogue by re-running the
// active node with the player's selected choice ID. It returns the narrative
// text emitted after the choice and any Yarn commands encountered during
// post-choice execution.
//
// The caller is responsible for persisting the updated state and processing
// any returned commands.
func ResumeDialogueWithChoice(def *CampaignDefinition, g *game.Game, choiceID int) (string, error) {
	if def == nil || g == nil || g.Campaign == nil || g.Campaign.ActiveDialogue == nil {
		return "", fmt.Errorf("no active dialogue to resume")
	}
	state := g.Campaign
	if !state.ActiveDialogue.AwaitingChoice {
		return "", fmt.Errorf("dialogue is not awaiting a choice")
	}
	asset, ok := def.DialogueAssets[state.ActiveDialogue.AssetID]
	if !ok {
		return "", fmt.Errorf("dialogue asset %q not found in campaign", state.ActiveDialogue.AssetID)
	}
	result, err := dialogue.RunNode(
		def.SourceFS,
		asset.Program,
		asset.Strings,
		state.ActiveDialogue.CurrentNode,
		state.ActiveDialogue.Variables,
		&choiceID,
	)
	if err != nil {
		return "", fmt.Errorf("resume dialogue with choice %d: %w", choiceID, err)
	}
	if len(result.Commands) > 0 {
		if applyErr := ApplyDialogueCommands(def, g, result.Commands); applyErr != nil {
			return "", fmt.Errorf("apply dialogue commands: %w", applyErr)
		}
	}

	// Update runtime state.
	state.ActiveDialogue.CurrentNode = result.CurrentNode
	state.ActiveDialogue.Variables = result.Variables
	if len(result.PendingChoices) > 0 {
		// Another choice follows immediately; keep awaiting state.
		choices := make([]game.DialogueChoice, len(result.PendingChoices))
		for i, c := range result.PendingChoices {
			choices[i] = game.DialogueChoice{ID: c.ID, Text: c.Text}
		}
		state.ActiveDialogue.AwaitingChoice = true
		state.ActiveDialogue.PendingChoices = choices
	} else {
		state.ActiveDialogue.AwaitingChoice = false
		state.ActiveDialogue.PendingChoices = nil
	}

	text := strings.Join(result.Lines, "\n")
	return text, nil
}

// ApplyDialogueCommands parses and executes a list of Yarn command strings
// against campaign runtime state via the existing action executor.
func ApplyDialogueCommands(def *CampaignDefinition, g *game.Game, commands []string) error {
	if g == nil || g.Campaign == nil || len(commands) == 0 {
		return nil
	}
	actions := make([]Action, 0, len(commands))
	for _, cmd := range commands {
		action, err := parseDialogueCommand(cmd)
		if err != nil {
			return err
		}
		actions = append(actions, action)
	}
	return applyActions(def, g, actions)
}

func parseDialogueCommand(command string) (Action, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return Action{}, fmt.Errorf("dialogue command is empty")
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return Action{}, fmt.Errorf("dialogue command is empty")
	}
	switch parts[0] {
	case "set_flag":
		if len(parts) < 2 {
			return Action{}, fmt.Errorf("set_flag requires key")
		}
		value := true
		if len(parts) >= 3 {
			if parts[2] == "false" {
				value = false
			} else if parts[2] != "true" {
				return Action{}, fmt.Errorf("set_flag value must be true or false")
			}
		}
		return Action{Type: "set_flag", Key: parts[1], Value: value}, nil
	case "set_label":
		if len(parts) < 3 {
			return Action{}, fmt.Errorf("set_label requires key and value")
		}
		return Action{Type: "set_label", Key: parts[1], Value: strings.Join(parts[2:], " ")}, nil
	case "inc_counter":
		if len(parts) < 2 {
			return Action{}, fmt.Errorf("inc_counter requires key")
		}
		if len(parts) >= 3 {
			return Action{Type: "inc_counter", Key: parts[1], Value: parts[2]}, nil
		}
		return Action{Type: "inc_counter", Key: parts[1], Value: 1}, nil
	case "set_counter":
		if len(parts) < 3 {
			return Action{}, fmt.Errorf("set_counter requires key and value")
		}
		return Action{Type: "set_counter", Key: parts[1], Value: parts[2]}, nil
	case "activate_objective":
		if len(parts) < 2 {
			return Action{}, fmt.Errorf("activate_objective requires objective ID")
		}
		return Action{Type: "activate_objective", ObjectiveID: parts[1]}, nil
	case "complete_objective":
		if len(parts) < 2 {
			return Action{}, fmt.Errorf("complete_objective requires objective ID")
		}
		return Action{Type: "complete_objective", ObjectiveID: parts[1]}, nil
	case "fail_objective":
		if len(parts) < 2 {
			return Action{}, fmt.Errorf("fail_objective requires objective ID")
		}
		return Action{Type: "fail_objective", ObjectiveID: parts[1]}, nil
	case "start_dialogue":
		if len(parts) < 3 {
			return Action{}, fmt.Errorf("start_dialogue requires asset_id and start_node")
		}
		return Action{Type: "start_dialogue", AssetID: parts[1], StartNode: parts[2]}, nil
	case "end_dialogue":
		return Action{Type: "end_dialogue"}, nil
	default:
		return Action{}, fmt.Errorf("unsupported dialogue command %q", parts[0])
	}
}

func conditionsMatch(def *CampaignDefinition, g *game.Game, conditions []Condition) (bool, error) {
	for _, cond := range conditions {
		matched, err := conditionMatches(def, g, cond)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func conditionMatches(def *CampaignDefinition, g *game.Game, cond Condition) (bool, error) {
	state := g.Campaign
	if state == nil {
		return false, nil
	}
	switch cond.Type {
	case "flag_true":
		return state.BoolFlags[cond.Key], nil
	case "flag_false":
		return !state.BoolFlags[cond.Key], nil
	case "label_equals":
		return state.Labels[cond.Key] == stringifyValue(cond.Value), nil
	case "label_in":
		value := state.Labels[cond.Key]
		for _, candidate := range append([]string{}, cond.Values...) {
			if value == candidate {
				return true, nil
			}
		}
		for _, candidate := range cond.AllowedVals {
			if value == candidate {
				return true, nil
			}
		}
		return false, nil
	case "counter_gte":
		threshold, err := intValue(cond.Value)
		if err != nil {
			return false, err
		}
		return state.Counters[cond.Key] >= threshold, nil
	case "counter_lte":
		threshold, err := intValue(cond.Value)
		if err != nil {
			return false, err
		}
		return state.Counters[cond.Key] <= threshold, nil
	case "actor_at_step":
		actorState, ok := state.ActorStates[cond.ActorID]
		return ok && actorState.CurrentRailStepID == cond.StepID, nil
	case "actor_in_room":
		actorState, ok := state.ActorStates[cond.ActorID]
		return ok && actorState.CurrentLocationID == cond.RoomID, nil
	case "objective_status":
		for _, objective := range state.ActiveObjectives {
			if objective.ID == cond.ObjectiveID {
				return objective.Status == cond.Status, nil
			}
		}
		if def != nil {
			_, exists := def.Objectives[cond.ObjectiveID]
			return !exists && cond.Status == "", nil
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported condition type %q", cond.Type)
	}
}

func applyActions(def *CampaignDefinition, g *game.Game, actions []Action) error {
	if g.Campaign == nil {
		return nil
	}
	state := g.Campaign
	if state.BoolFlags == nil {
		state.BoolFlags = make(map[string]bool)
	}
	if state.Labels == nil {
		state.Labels = make(map[string]string)
	}
	if state.Counters == nil {
		state.Counters = make(map[string]int)
	}
	if state.ActorStates == nil {
		state.ActorStates = make(map[string]game.ActorRuntimeState)
	}
	for _, action := range actions {
		switch action.Type {
		case "set_flag":
			if action.Key == "" {
				continue
			}
			value := true
			if boolValue, ok := action.Value.(bool); ok {
				value = boolValue
			}
			state.BoolFlags[action.Key] = value
		case "set_label":
			if action.Key == "" {
				continue
			}
			state.Labels[action.Key] = stringifyValue(action.Value)
		case "inc_counter":
			if action.Key == "" {
				continue
			}
			delta, err := intValueDefault(action.Value, 1)
			if err != nil {
				return err
			}
			state.Counters[action.Key] += delta
		case "set_counter":
			if action.Key == "" {
				continue
			}
			value, err := intValue(action.Value)
			if err != nil {
				return err
			}
			state.Counters[action.Key] = value
		case "activate_objective":
			setObjectiveStatus(def, state, action.ObjectiveID, "active")
		case "complete_objective":
			setObjectiveStatus(def, state, action.ObjectiveID, "completed")
		case "fail_objective":
			setObjectiveStatus(def, state, action.ObjectiveID, "failed")
		case "move_actor_to_step":
			actorState := state.ActorStates[action.ActorID]
			actorState.ActorID = action.ActorID
			actorState.CurrentRailStepID = action.StepID
			if def != nil {
				for _, step := range def.Actors[action.ActorID].Rail.Steps {
					if step.ID == action.StepID {
						actorState.CurrentLocationID = step.LocationID
						break
					}
				}
			}
			state.ActorStates[action.ActorID] = actorState
		case "move_actor_to_room":
			actorState := state.ActorStates[action.ActorID]
			actorState.ActorID = action.ActorID
			actorState.CurrentLocationID = action.RoomID
			state.ActorStates[action.ActorID] = actorState
		case "start_dialogue":
			state.ActiveDialogue = &game.DialogueRuntimeState{
				AssetID:     action.AssetID,
				CurrentNode: action.StartNode,
			}
		case "end_dialogue":
			state.ActiveDialogue = nil
		default:
			return fmt.Errorf("unsupported action type %q", action.Type)
		}
	}
	return nil
}

func setObjectiveStatus(def *CampaignDefinition, state *game.CampaignRuntimeState, objectiveID, status string) {
	if objectiveID == "" {
		return
	}
	visibleText := objectiveID
	if def != nil {
		if obj, ok := def.Objectives[objectiveID]; ok && obj.Title != "" {
			visibleText = obj.Title
		}
	}
	for i := range state.ActiveObjectives {
		if state.ActiveObjectives[i].ID == objectiveID {
			state.ActiveObjectives[i].Status = status
			if state.ActiveObjectives[i].VisibleText == "" {
				state.ActiveObjectives[i].VisibleText = visibleText
			}
			return
		}
	}
	state.ActiveObjectives = append(state.ActiveObjectives, game.ObjectiveState{
		ID:          objectiveID,
		Status:      status,
		VisibleText: visibleText,
	})
}

func allowedSpecFromCondition(cond Condition) (AllowedConditionSpec, bool) {
	switch cond.Type {
	case "flag_true", "flag_false", "counter_gte", "counter_lte":
		spec := AllowedConditionSpec{Type: cond.Type, Key: cond.Key}
		if rendered := stringifyValue(cond.Value); rendered != "" {
			spec.AllowedValues = []string{rendered}
		}
		return spec, cond.Key != ""
	case "label_equals", "label_in":
		values := append([]string{}, cond.Values...)
		values = append(values, cond.AllowedVals...)
		if rendered := stringifyValue(cond.Value); rendered != "" {
			values = append(values, rendered)
		}
		return AllowedConditionSpec{Type: "label_equals", Key: cond.Key, AllowedValues: dedupeStrings(values)}, cond.Key != ""
	default:
		return AllowedConditionSpec{}, false
	}
}

func mergeAllowedSpec(dst map[string]AllowedConditionSpec, spec AllowedConditionSpec) {
	if spec.Key == "" || spec.Type == "" {
		return
	}
	key := spec.Type + "::" + spec.Key
	if existing, ok := dst[key]; ok {
		existing.AllowedValues = dedupeStrings(append(existing.AllowedValues, spec.AllowedValues...))
		dst[key] = existing
		return
	}
	spec.AllowedValues = dedupeStrings(spec.AllowedValues)
	dst[key] = spec
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringifyValue(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case float64:
		return strconv.Itoa(int(typed))
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.Itoa(int(typed))
	case int64:
		return strconv.Itoa(int(typed))
	default:
		return ""
	}
}

func intValueDefault(v any, fallback int) (int, error) {
	if v == nil {
		return fallback, nil
	}
	return intValue(v)
}

func intValue(v any) (int, error) {
	switch typed := v.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case string:
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0, fmt.Errorf("parse int %q: %w", typed, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported numeric value type %T", v)
	}
}
