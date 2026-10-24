package campaigns

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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
	// RunEnded is set when this transition ended the run.
	RunEnded bool
	// ObjectiveEvents report objectives closed by their success_when or
	// fail_when conditions, whether or not the node changed.
	ObjectiveEvents []game.WorldEvent
}

type TurnOrchestrationResult struct {
	Transition TransitionResult
	Events     []game.WorldEvent
}

func AllowedConditionSpecsForNode(node StoryNode) []AllowedConditionSpec {
	merged := make(map[string]AllowedConditionSpec)
	for _, spec := range beatSpecs(node.Beats) {
		mergeAllowedSpec(merged, spec)
	}
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
	g.Campaign.ActiveNodeTitle = node.Title
	g.Campaign.ActiveMood = node.Mood
	g.Campaign.ActiveSceneMap = nil
	if sm, ok := def.Map.SceneMaps[node.SceneMap]; ok && node.SceneMap != "" {
		g.Campaign.ActiveSceneMap = &game.SceneMapView{ID: node.SceneMap, RoomID: sm.RoomID, Name: sm.Name}
	}
	if node.LocationID != "" {
		// The scene happens somewhere: the player is there.
		if err := g.PlaceCharacter(g.OwnerID, node.LocationID); err != nil {
			return "", fmt.Errorf("move player to scene location %q: %w", node.LocationID, err)
		}
	}
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
	opening := currentDialogueOpening(def, g)
	if node.IntroText == "" {
		return opening, nil
	}
	if opening == "" {
		return node.IntroText, nil
	}
	return node.IntroText + "\n\n" + opening, nil
}

func AdvanceCampaign(def *CampaignDefinition, g *game.Game) (TransitionResult, error) {
	if def == nil || g == nil || g.Campaign == nil || g.Campaign.RunEnded() {
		return TransitionResult{}, nil
	}
	currentNode, ok := def.StoryNodes[g.Campaign.ActiveNodeID]
	if !ok {
		return TransitionResult{}, fmt.Errorf("active story node %q not found", g.Campaign.ActiveNodeID)
	}
	objectiveEvents, err := EvaluateObjectives(def, g)
	if err != nil {
		return TransitionResult{}, err
	}
	// A scene ends on the player's last choice, not mid-conversation: while
	// the dialogue is still waiting on a choice, stay in this node.
	if d := g.Campaign.ActiveDialogue; d != nil && d.AwaitingChoice {
		return TransitionResult{ObjectiveEvents: objectiveEvents}, nil
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
				more, err := EvaluateObjectives(def, g)
				if err != nil {
					return TransitionResult{}, err
				}
				return TransitionResult{Transitioned: true, FromNodeID: fromNodeID, ObjectiveEvents: append(objectiveEvents, more...)}, nil
			}
			g.Campaign.PreviousNodeIDs = append(g.Campaign.PreviousNodeIDs, fromNodeID)
			g.Campaign.ActiveNodeID = rule.NextNodeID
			dialogueText, err := EnterActiveNode(def, g)
			if err != nil {
				return TransitionResult{}, err
			}
			more, err := EvaluateObjectives(def, g)
			if err != nil {
				return TransitionResult{}, err
			}
			result := TransitionResult{
				Transitioned:    true,
				FromNodeID:      fromNodeID,
				ToNodeID:        rule.NextNodeID,
				DialogueText:    dialogueText,
				ObjectiveEvents: append(objectiveEvents, more...),
			}
			if def.StoryNodes[rule.NextNodeID].EndsRun {
				endingText, err := EndRun(def, g)
				if err != nil {
					return TransitionResult{}, err
				}
				result.RunEnded = true
				result.DialogueText = joinText(result.DialogueText, endingText)
			}
			return result, nil
		}
	}
	return TransitionResult{ObjectiveEvents: objectiveEvents}, nil
}

// EvaluateObjectives closes active objectives whose authored success_when
// (checked first), fail_when or unfinished_when conditions now hold, and
// returns an event for each one so the player sees the change. An objective
// closed without success takes its quest items with it.
func EvaluateObjectives(def *CampaignDefinition, g *game.Game) ([]game.WorldEvent, error) {
	var events []game.WorldEvent
	for i := range g.Campaign.ActiveObjectives {
		o := &g.Campaign.ActiveObjectives[i]
		tmpl, ok := def.Objectives[o.ID]
		if o.Status != "active" || !ok {
			continue
		}
		title := tmpl.Title
		if title == "" {
			title = o.ID
		}
		if len(tmpl.SuccessWhen) > 0 {
			matched, err := conditionsMatch(def, g, tmpl.SuccessWhen)
			if err != nil {
				return nil, fmt.Errorf("objective %q success_when: %w", o.ID, err)
			}
			if matched {
				o.Status = "completed"
				events = append(events, game.WorldEvent{Type: "objective_completed", Message: "Objective complete: " + title})
				if err := applyActions(def, g, tmpl.OnSuccess); err != nil {
					return nil, fmt.Errorf("objective %q on_success: %w", o.ID, err)
				}
				continue
			}
		}
		if len(tmpl.FailWhen) > 0 {
			matched, err := conditionsMatch(def, g, tmpl.FailWhen)
			if err != nil {
				return nil, fmt.Errorf("objective %q fail_when: %w", o.ID, err)
			}
			if matched {
				o.Status = "failed"
				events = append(events, game.WorldEvent{Type: "objective_failed", Message: "Objective missed: " + title})
				events = append(events, removeQuestItems(g, tmpl.QuestItems)...)
				continue
			}
		}
		if len(tmpl.UnfinishedWhen) > 0 {
			matched, err := conditionsMatch(def, g, tmpl.UnfinishedWhen)
			if err != nil {
				return nil, fmt.Errorf("objective %q unfinished_when: %w", o.ID, err)
			}
			if matched {
				o.Status = ObjectiveUnfinished
				events = append(events, game.WorldEvent{Type: "objective_unfinished", Message: "Left unfinished: " + title})
				events = append(events, removeQuestItems(g, tmpl.QuestItems)...)
			}
		}
	}
	return events, nil
}

// ObjectiveUnfinished is the status of an objective the story moved past.
const ObjectiveUnfinished = "unfinished"

// removeQuestItems takes every item matching the patterns out of the
// player's pack and the rooms, with an event for what the player loses.
func removeQuestItems(g *game.Game, patterns []string) []game.WorldEvent {
	var events []game.WorldEvent
	matches := func(name string) bool {
		name = strings.ToLower(name)
		for _, p := range patterns {
			for _, want := range strings.Split(p, "|") {
				want = strings.ToLower(strings.TrimSpace(want))
				if want != "" && strings.Contains(name, want) {
					return true
				}
			}
		}
		return false
	}
	if len(patterns) == 0 {
		return nil
	}
	if owner, ok := g.OwnerCharacter(); ok {
		for _, id := range append([]string(nil), owner.Inventory...) {
			if it, ok := g.Items[id]; ok && matches(it.Name) {
				_ = owner.RemoveItemID(id)
				delete(g.Items, id)
				events = append(events, game.WorldEvent{Type: "item_lost", Message: it.Name + " is gone from your pack."})
			}
		}
		g.Players[g.OwnerID] = owner
	}
	for roomID, room := range g.Rooms {
		kept := room.Items[:0:0]
		for _, id := range room.Items {
			if it, ok := g.Items[id]; ok && matches(it.Name) {
				delete(g.Items, id)
				continue
			}
			kept = append(kept, id)
		}
		if len(kept) != len(room.Items) {
			room.Items = kept
			g.Rooms[roomID] = room
		}
	}
	return events
}

func OrchestrateTurn(def *CampaignDefinition, g *game.Game, resolution ConditionResolution) (TurnOrchestrationResult, error) {
	if def == nil || g == nil || g.Campaign == nil || g.Campaign.RunEnded() {
		return TurnOrchestrationResult{}, nil
	}

	// A party wipe ends the run before any story transition is considered.
	if wiped, err := EndRunIfPartyWiped(def, g); err != nil || wiped.Transition.RunEnded {
		return wiped, err
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
	result.Events = append(result.Events, transition.ObjectiveEvents...)
	if transition.RunEnded {
		result.Events = append(result.Events, runEndedEvent(g))
		return result, nil
	}
	if !transition.Transitioned || transition.ToNodeID == "" {
		return result, nil
	}

	nextNode, ok := def.StoryNodes[transition.ToNodeID]
	if ok {
		result.Events = append(result.Events, SceneStartedEvent(nextNode))
	}
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
	if commands := takeStaging(g, result); len(commands) > 0 {
		if applyErr := ApplyDialogueCommands(def, g, commands); applyErr != nil {
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
		state.ActiveDialogue.ChoiceHistory = result.ChoiceHistory
	} else {
		state.ActiveDialogue.AwaitingChoice = false
		state.ActiveDialogue.PendingChoices = nil
		state.ActiveDialogue.ChoiceHistory = nil
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
		append(append([]int(nil), state.ActiveDialogue.ChoiceHistory...), choiceID),
	)
	if err != nil {
		return "", fmt.Errorf("resume dialogue with choice %d: %w", choiceID, err)
	}
	if commands := takeStaging(g, result); len(commands) > 0 {
		if applyErr := ApplyDialogueCommands(def, g, commands); applyErr != nil {
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
		state.ActiveDialogue.ChoiceHistory = result.ChoiceHistory
	} else {
		state.ActiveDialogue.AwaitingChoice = false
		state.ActiveDialogue.PendingChoices = nil
		state.ActiveDialogue.ChoiceHistory = nil
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
	case "give_item", "take_item":
		if len(parts) < 2 {
			return Action{}, fmt.Errorf("%s requires an item name", parts[0])
		}
		return Action{Type: parts[0], Item: strings.Join(parts[1:], " ")}, nil
	case "move_player":
		if len(parts) != 2 {
			return Action{}, fmt.Errorf("move_player requires a room ID")
		}
		return Action{Type: "move_player", RoomID: parts[1]}, nil
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
	case "player_in_room":
		owner, ok := g.OwnerCharacter()
		return ok && owner.LocationID == cond.RoomID, nil
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
			if def != nil {
				removeQuestItems(g, def.Objectives[action.ObjectiveID].QuestItems)
			}
		case "move_actor_to_step":
			actorState := state.ActorStates[action.ActorID]
			actorState.ActorID = action.ActorID
			actorState.CurrentRailStepID = action.StepID
			if def != nil {
				for _, step := range def.Actors[action.ActorID].Rail.Steps {
					if step.ID == action.StepID {
						actorState.CurrentLocationID = resolveLocation(g, step.LocationID)
						break
					}
				}
			}
			state.ActorStates[action.ActorID] = actorState
			if err := placeActor(def, g, action.ActorID, actorState.CurrentLocationID); err != nil {
				return err
			}
		case "move_actor_to_room":
			actorState := state.ActorStates[action.ActorID]
			actorState.ActorID = action.ActorID
			actorState.CurrentLocationID = resolveLocation(g, action.RoomID)
			state.ActorStates[action.ActorID] = actorState
			if err := placeActor(def, g, action.ActorID, actorState.CurrentLocationID); err != nil {
				return err
			}
		case "start_dialogue":
			state.ActiveDialogue = &game.DialogueRuntimeState{
				AssetID:     action.AssetID,
				CurrentNode: action.StartNode,
			}
		case "end_dialogue":
			state.ActiveDialogue = nil
		case "give_item":
			item := game.NewItem(action.Item, action.Description)
			item.Weight = 0.1
			if err := g.AddItem(item); err != nil {
				return err
			}
			if err := g.GiveItemToPlayer(item.ID); err != nil {
				return err
			}
		case "take_item":
			takeItemFromPlayer(g, action.Item)
		case "move_player":
			if err := g.PlaceCharacter(g.OwnerID, action.RoomID); err != nil {
				return err
			}
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
		if existing.Description == "" {
			existing.Description = spec.Description
		}
		for value, meaning := range spec.Meanings {
			if existing.Meanings == nil {
				existing.Meanings = map[string]string{}
			}
			if _, ok := existing.Meanings[value]; !ok {
				existing.Meanings[value] = meaning
			}
		}
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

// now is the clock used to stamp ended runs; tests may replace it.
var now = time.Now

// SelectEnding returns the first ending whose conditions all match the
// current campaign state, or nil if the campaign defines no endings.
func SelectEnding(def *CampaignDefinition, g *game.Game) (*Ending, error) {
	for i := range def.Endings {
		matched, err := conditionsMatch(def, g, def.Endings[i].Conditions)
		if err != nil {
			return nil, fmt.Errorf("ending %q: %w", def.Endings[i].ID, err)
		}
		if matched {
			return &def.Endings[i], nil
		}
	}
	return nil, nil
}

// EndRun marks the run as ended, records the selected ending, and starts
// the ending's dialogue if it has one. It returns the ending dialogue text.
// Ending a run that has already ended does nothing.
func EndRun(def *CampaignDefinition, g *game.Game) (string, error) {
	if def == nil || g == nil || g.Campaign == nil || g.Campaign.RunEnded() {
		return "", nil
	}
	ending, err := SelectEnding(def, g)
	if err != nil {
		return "", err
	}
	state := g.Campaign
	state.RunStatus = game.RunStatusEnded
	state.EndedAt = now().UnixMilli()
	state.ActiveDialogue = nil
	if ending == nil {
		return "", nil
	}
	state.EndingID = ending.ID
	state.EndingTitle = ending.Title
	state.EndingSummary = ending.Summary
	state.CurrentObjective = ""
	if ending.DialogueAssetID == "" {
		return "", nil
	}
	state.ActiveDialogue = &game.DialogueRuntimeState{
		AssetID:     ending.DialogueAssetID,
		CurrentNode: ending.DialogueStartNode,
	}
	return currentDialogueOpening(def, g), nil
}

// EndRunIfPartyWiped ends the run when every player character is dead.
func EndRunIfPartyWiped(def *CampaignDefinition, g *game.Game) (TurnOrchestrationResult, error) {
	if def == nil || g == nil || g.Campaign == nil || g.Campaign.RunEnded() || !g.PartyWiped() {
		return TurnOrchestrationResult{}, nil
	}
	text, err := EndRun(def, g)
	if err != nil {
		return TurnOrchestrationResult{}, err
	}
	return TurnOrchestrationResult{
		Transition: TransitionResult{
			Transitioned: true,
			FromNodeID:   g.Campaign.ActiveNodeID,
			DialogueText: text,
			RunEnded:     true,
		},
		Events: []game.WorldEvent{runEndedEvent(g)},
	}, nil
}

func runEndedEvent(g *game.Game) game.WorldEvent {
	msg := "The adventure has ended."
	if g.Campaign.EndingTitle != "" {
		msg = "The adventure has ended: " + g.Campaign.EndingTitle
	}
	return game.WorldEvent{Type: "campaign_ended", Message: msg}
}

func joinText(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n")
}

// SceneStartedEvent marks the start of a story node so the client can show
// a title card before the scene's text.
func SceneStartedEvent(node StoryNode) game.WorldEvent {
	return game.WorldEvent{Type: "scene_started", Message: node.Title}
}

// takeItemFromPlayer removes one item from the player's inventory: the first
// of the "|"-separated names that matches exactly (ignoring case), else the
// first item whose name contains it. Nothing happens when none match; the
// story doesn't fail because the player lost something.
func takeItemFromPlayer(g *game.Game, names string) {
	owner, ok := g.OwnerCharacter()
	if !ok {
		return
	}
	for _, want := range strings.Split(names, "|") {
		want = strings.ToLower(strings.TrimSpace(want))
		if want == "" {
			continue
		}
		for _, exact := range []bool{true, false} {
			for _, id := range owner.Inventory {
				name := strings.ToLower(g.Items[id].Name)
				if (exact && name == want) || (!exact && strings.Contains(name, want)) {
					_ = owner.RemoveItemID(id)
					g.Players[g.OwnerID] = owner
					delete(g.Items, id)
					return
				}
			}
		}
	}
}

// PlayerLocation is the location an actor's rail step or move can name to
// mean wherever the player is: someone arriving to find them.
const PlayerLocation = "@player"

func resolveLocation(g *game.Game, roomID string) string {
	if roomID != PlayerLocation {
		return roomID
	}
	if owner, ok := g.OwnerCharacter(); ok {
		return owner.LocationID
	}
	return ""
}

// placeActor puts an actor's character where the story says they are, so
// the world (the map, the game master's view of the room) agrees with the
// actor's state. An actor who isn't in the world yet arrives; an empty room
// takes them off-stage.
func placeActor(def *CampaignDefinition, g *game.Game, actorID, roomID string) error {
	if g.NPCs == nil {
		g.NPCs = map[string]game.Character{}
	}
	npc, exists := g.NPCs[actorID]
	if !exists {
		if roomID == "" || def == nil {
			return nil
		}
		actor, ok := def.Actors[actorID]
		if !ok || actor.Kind == "companion" {
			return nil
		}
		npc = newActorCharacter(actorID, actor)
	}
	if npc.LocationID != "" {
		if old, ok := g.Rooms[npc.LocationID]; ok {
			_ = old.RemoveOccupant(actorID)
			g.Rooms[npc.LocationID] = old
		}
	}
	if room, ok := g.Rooms[roomID]; ok {
		_ = room.AddOccupant(actorID)
		g.Rooms[roomID] = room
		npc.LocationID = roomID
	} else {
		// Off-stage, or a room this world doesn't have: the actor's state
		// still says where they are.
		npc.LocationID = ""
	}
	g.NPCs[actorID] = npc
	return nil
}
