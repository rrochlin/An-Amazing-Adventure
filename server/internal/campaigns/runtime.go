package campaigns

import (
	"fmt"
	"sort"
	"strconv"

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
			nextNode, ok := def.StoryNodes[rule.NextNodeID]
			if !ok {
				return TransitionResult{}, fmt.Errorf("next story node %q not found", rule.NextNodeID)
			}
			g.Campaign.PreviousNodeIDs = append(g.Campaign.PreviousNodeIDs, fromNodeID)
			g.Campaign.ActiveNodeID = rule.NextNodeID
			g.Campaign.CurrentObjective = nextNode.ObjectiveText
			if err := applyActions(def, g, nextNode.OnEnter); err != nil {
				return TransitionResult{}, err
			}
			return TransitionResult{
				Transitioned: true,
				FromNodeID:   fromNodeID,
				ToNodeID:     rule.NextNodeID,
			}, nil
		}
	}
	return TransitionResult{}, nil
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
