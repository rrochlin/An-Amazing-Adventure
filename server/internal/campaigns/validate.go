package campaigns

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

var validNodeModes = map[string]bool{
	"ai_scene":      true,
	"yarn_dialogue": true,
	"hybrid":        true,
}

func (d *CampaignDefinition) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("id is required")
	}
	if d.Version == "" {
		return fmt.Errorf("version is required")
	}
	if d.Title == "" {
		return fmt.Errorf("title is required")
	}
	if d.Premise == "" {
		return fmt.Errorf("premise is required")
	}
	switch d.StoryPipeline {
	case "", StoryPipelineCurrent, StoryPipelineGameMaster:
	default:
		return fmt.Errorf("story_pipeline %q must be %q or %q", d.StoryPipeline, StoryPipelineCurrent, StoryPipelineGameMaster)
	}
	if !game.ValidScale(d.PlayerRole.Scale) {
		return fmt.Errorf("player_role.scale %v must be between 0 and %v", d.PlayerRole.Scale, game.MaxScale)
	}
	if len(d.Map.Rooms) == 0 {
		return fmt.Errorf("map.rooms must contain at least one room")
	}
	if d.Party.StartingRoomID == "" {
		return fmt.Errorf("party.starting_room_id is required")
	}
	if _, ok := d.Map.Rooms[d.Party.StartingRoomID]; !ok {
		return fmt.Errorf("party.starting_room_id %q does not exist in map.rooms", d.Party.StartingRoomID)
	}
	if len(d.StoryNodes) == 0 {
		return fmt.Errorf("story_nodes must contain at least one node")
	}
	for id, actor := range d.Actors {
		if actor.StartLocationID == "" {
			continue
		}
		if _, ok := d.Map.Rooms[actor.StartLocationID]; !ok {
			return fmt.Errorf("actor %q start_location_id %q does not exist in map.rooms", id, actor.StartLocationID)
		}
	}
	for roomID, room := range d.Map.Rooms {
		seen := map[string]bool{}
		for _, poi := range room.PointsOfInterest {
			if poi.ID == "" || poi.Name == "" {
				return fmt.Errorf("room %q has a point of interest without an id or name", roomID)
			}
			if seen[poi.ID] {
				return fmt.Errorf("room %q has duplicate point of interest %q", roomID, poi.ID)
			}
			seen[poi.ID] = true
		}
	}
	if d.StartNodeID == "" {
		return fmt.Errorf("start_node_id is required")
	}
	if _, ok := d.StoryNodes[d.StartNodeID]; !ok {
		return fmt.Errorf("start_node_id %q does not exist in story_nodes", d.StartNodeID)
	}

	for _, id := range d.Party.CompanionIDs {
		if _, ok := d.Actors[id]; !ok {
			return fmt.Errorf("party companion %q does not exist in actors", id)
		}
	}

	for _, e := range d.Map.Entrances {
		if e.ID == "" {
			return fmt.Errorf("map entrance id is required")
		}
		if e.StartRoomID == "" {
			return fmt.Errorf("map entrance %q start_room_id is required", e.ID)
		}
		if _, ok := d.Map.Rooms[e.StartRoomID]; !ok {
			return fmt.Errorf("map entrance %q start_room_id %q does not exist", e.ID, e.StartRoomID)
		}
	}

	for roomID, room := range d.Map.Rooms {
		if room.Name == "" {
			return fmt.Errorf("room %q name is required", roomID)
		}
		for direction, target := range room.Connections {
			if direction == "" {
				return fmt.Errorf("room %q has empty connection direction", roomID)
			}
			if _, ok := d.Map.Rooms[target]; !ok {
				return fmt.Errorf("room %q connection %q references unknown room %q", roomID, direction, target)
			}
		}
	}

	for actorID, actor := range d.Actors {
		if actor.Name == "" {
			return fmt.Errorf("actor %q name is required", actorID)
		}
		if err := d.validateSheet(actorID, actor); err != nil {
			return err
		}
		for _, nodeID := range actor.AppearsIn {
			if _, ok := d.StoryNodes[nodeID]; !ok {
				return fmt.Errorf("actor %q appears_in references unknown story node %q", actorID, nodeID)
			}
		}
		for _, step := range actor.Rail.Steps {
			if step.ID == "" {
				return fmt.Errorf("actor %q rail step id is required", actorID)
			}
			if step.LocationID != "" && step.LocationID != PlayerLocation {
				if _, ok := d.Map.Rooms[step.LocationID]; !ok {
					return fmt.Errorf("actor %q rail step %q references unknown room %q", actorID, step.ID, step.LocationID)
				}
			}
			if err := d.validateActions(step.OnAdvance, fmt.Sprintf("actor %q rail step %q on_advance", actorID, step.ID)); err != nil {
				return err
			}
			if err := d.validateConditions(step.AdvanceWhen, fmt.Sprintf("actor %q rail step %q advance_when", actorID, step.ID)); err != nil {
				return err
			}
			if err := d.validateConditions(step.HoldWhen, fmt.Sprintf("actor %q rail step %q hold_when", actorID, step.ID)); err != nil {
				return err
			}
			if err := d.validateConditions(step.RevealWhen, fmt.Sprintf("actor %q rail step %q reveal_when", actorID, step.ID)); err != nil {
				return err
			}
		}
	}

	for objectiveID, obj := range d.Objectives {
		for name, conds := range map[string][]Condition{"success_when": obj.SuccessWhen, "fail_when": obj.FailWhen, "unfinished_when": obj.UnfinishedWhen, "visible_when": obj.VisibleWhen} {
			if err := d.validateConditions(conds, fmt.Sprintf("objective %q %s", objectiveID, name)); err != nil {
				return err
			}
		}
		if err := d.validateActions(obj.OnSuccess, fmt.Sprintf("objective %q on_success", objectiveID)); err != nil {
			return err
		}
	}

	for assetID, asset := range d.DialogueAssets {
		if asset.Format == "" {
			return fmt.Errorf("dialogue_assets.%s format is required", assetID)
		}
		if asset.Program == "" {
			return fmt.Errorf("dialogue_assets.%s program is required", assetID)
		}
		if asset.Strings == "" {
			return fmt.Errorf("dialogue_assets.%s strings is required", assetID)
		}
		if d.SourceDir != "" {
			if err := requireFileFS(d.SourceFS, asset.Program); err != nil {
				return fmt.Errorf("dialogue_assets.%s program: %w", assetID, err)
			}
			if err := requireFileFS(d.SourceFS, asset.Strings); err != nil {
				return fmt.Errorf("dialogue_assets.%s strings: %w", assetID, err)
			}
		}
	}

	for roomID, room := range d.Map.Rooms {
		for i, it := range room.Items {
			if strings.TrimSpace(it.Name) == "" {
				return fmt.Errorf("map.rooms.%s.items[%d] needs a name", roomID, i)
			}
		}
	}
	for id, sm := range d.Map.SceneMaps {
		if sm.Name == "" {
			return fmt.Errorf("map.scene_maps %q needs a name", id)
		}
		if _, ok := d.Map.Rooms[sm.RoomID]; !ok {
			return fmt.Errorf("map.scene_maps %q room_id %q does not exist in map.rooms", id, sm.RoomID)
		}
	}
	for nodeID, node := range d.StoryNodes {
		if node.Title == "" {
			return fmt.Errorf("story node %q title is required", nodeID)
		}
		if !validNodeModes[node.Mode] {
			return fmt.Errorf("story node %q mode %q is invalid", nodeID, node.Mode)
		}
		if node.DialogueAssetID != "" {
			if _, ok := d.DialogueAssets[node.DialogueAssetID]; !ok {
				return fmt.Errorf("story node %q dialogue_asset_id %q does not exist", nodeID, node.DialogueAssetID)
			}
		}
		if node.Mode == "yarn_dialogue" || node.Mode == "hybrid" {
			if node.DialogueAssetID == "" {
				return fmt.Errorf("story node %q mode %q requires dialogue_asset_id", nodeID, node.Mode)
			}
			if node.DialogueStartNode == "" {
				return fmt.Errorf("story node %q mode %q requires dialogue_start_node", nodeID, node.Mode)
			}
		}
		if err := d.validateActions(node.OnEnter, fmt.Sprintf("story node %q on_enter", nodeID)); err != nil {
			return err
		}
		if err := d.validateAllowedConditions(node.AllowedConditions, fmt.Sprintf("story node %q allowed_conditions", nodeID)); err != nil {
			return err
		}
		if err := d.validateTransitionRules(node.CompletionRules, fmt.Sprintf("story node %q completion_rules", nodeID)); err != nil {
			return err
		}
		if err := d.validateTransitionRules(node.FailureRules, fmt.Sprintf("story node %q failure_rules", nodeID)); err != nil {
			return err
		}
		if node.LocationID != "" {
			if _, ok := d.Map.Rooms[node.LocationID]; !ok {
				return fmt.Errorf("story node %q location_id %q does not exist in map.rooms", nodeID, node.LocationID)
			}
		}
		if node.Mood != "" && !slices.Contains(SceneMoods, node.Mood) {
			return fmt.Errorf("story node %q mood %q is not one of %v", nodeID, node.Mood, SceneMoods)
		}
		if node.SceneMap != "" {
			sm, ok := d.Map.SceneMaps[node.SceneMap]
			if !ok {
				return fmt.Errorf("story node %q scene_map %q does not exist in map.scene_maps", nodeID, node.SceneMap)
			}
			if node.LocationID != sm.RoomID {
				return fmt.Errorf("story node %q scene_map %q is inside room %q, but the node's location_id is %q", nodeID, node.SceneMap, sm.RoomID, node.LocationID)
			}
		}
		if err := d.validateScript(nodeID, node); err != nil {
			return err
		}
	}

	if err := d.CharacterCreation.validate(); err != nil {
		return fmt.Errorf("character_creation: %w", err)
	}

	if err := d.validateEndings(); err != nil {
		return err
	}

	if d.IntroSequence.OpeningNodeID != "" {
		if _, ok := d.StoryNodes[d.IntroSequence.OpeningNodeID]; !ok {
			return fmt.Errorf("intro_sequence opening_node_id %q does not exist in story_nodes", d.IntroSequence.OpeningNodeID)
		}
	}

	return d.validateStaging()
}

func (d *CampaignDefinition) validateTransitionRules(rules []TransitionRule, context string) error {
	for i, rule := range rules {
		if rule.NextNodeID == "" {
			return fmt.Errorf("%s[%d] next_node_id is required", context, i)
		}
		if _, ok := d.StoryNodes[rule.NextNodeID]; !ok {
			return fmt.Errorf("%s[%d] next_node_id %q does not exist", context, i, rule.NextNodeID)
		}
		if err := d.validateConditions(rule.Conditions, fmt.Sprintf("%s[%d].conditions", context, i)); err != nil {
			return err
		}
		if err := d.validateActions(rule.Actions, fmt.Sprintf("%s[%d].actions", context, i)); err != nil {
			return err
		}
	}
	return nil
}

func (d *CampaignDefinition) validateConditions(conditions []Condition, context string) error {
	for i, cond := range conditions {
		if cond.Type == "" {
			return fmt.Errorf("%s[%d] type is required", context, i)
		}
		switch cond.Type {
		case "flag_true", "flag_false", "label_equals", "label_in", "counter_gte", "counter_lte":
			if cond.Key == "" {
				return fmt.Errorf("%s[%d] key is required for type %q", context, i, cond.Type)
			}
		case "actor_at_step", "actor_in_room":
			if cond.ActorID == "" {
				return fmt.Errorf("%s[%d] actor_id is required for type %q", context, i, cond.Type)
			}
			if _, ok := d.Actors[cond.ActorID]; !ok {
				return fmt.Errorf("%s[%d] actor_id %q does not exist", context, i, cond.ActorID)
			}
			if cond.Type == "actor_at_step" && cond.StepID == "" {
				return fmt.Errorf("%s[%d] step_id is required for type %q", context, i, cond.Type)
			}
			if cond.Type == "actor_in_room" {
				if cond.RoomID == "" {
					return fmt.Errorf("%s[%d] room_id is required for type %q", context, i, cond.Type)
				}
				if _, ok := d.Map.Rooms[cond.RoomID]; !ok {
					return fmt.Errorf("%s[%d] room_id %q does not exist", context, i, cond.RoomID)
				}
			}
		case "player_in_room":
			if cond.RoomID == "" {
				return fmt.Errorf("%s[%d] room_id is required for type %q", context, i, cond.Type)
			}
			if _, ok := d.Map.Rooms[cond.RoomID]; !ok {
				return fmt.Errorf("%s[%d] room_id %q does not exist", context, i, cond.RoomID)
			}
		case "objective_status":
			if cond.ObjectiveID == "" {
				return fmt.Errorf("%s[%d] objective_id is required for type %q", context, i, cond.Type)
			}
			if _, ok := d.Objectives[cond.ObjectiveID]; !ok {
				return fmt.Errorf("%s[%d] objective_id %q does not exist", context, i, cond.ObjectiveID)
			}
		default:
			return fmt.Errorf("%s[%d] unsupported condition type %q", context, i, cond.Type)
		}
	}
	return nil
}

func (d *CampaignDefinition) validateAllowedConditions(specs []AllowedConditionSpec, context string) error {
	for i, spec := range specs {
		if spec.Type == "" {
			return fmt.Errorf("%s[%d] type is required", context, i)
		}
		if spec.Key == "" {
			return fmt.Errorf("%s[%d] key is required", context, i)
		}
	}
	return nil
}

func (d *CampaignDefinition) validateActions(actions []Action, context string) error {
	for i, action := range actions {
		if action.Type == "" {
			return fmt.Errorf("%s[%d] type is required", context, i)
		}
		switch action.Type {
		case "set_flag", "set_label", "inc_counter", "set_counter":
			if action.Key == "" {
				return fmt.Errorf("%s[%d] key is required for type %q", context, i, action.Type)
			}
		case "give_item", "take_item":
			if action.Item == "" {
				return fmt.Errorf("%s[%d] item is required for type %q", context, i, action.Type)
			}
		case "move_player":
			if _, ok := d.Map.Rooms[action.RoomID]; !ok {
				return fmt.Errorf("%s[%d] room_id %q does not exist", context, i, action.RoomID)
			}
		case "activate_objective", "complete_objective", "fail_objective":
			if action.ObjectiveID == "" {
				return fmt.Errorf("%s[%d] objective_id is required for type %q", context, i, action.Type)
			}
			if _, ok := d.Objectives[action.ObjectiveID]; !ok {
				return fmt.Errorf("%s[%d] objective_id %q does not exist", context, i, action.ObjectiveID)
			}
		case "move_actor_to_step", "move_actor_to_room":
			if action.ActorID == "" {
				return fmt.Errorf("%s[%d] actor_id is required for type %q", context, i, action.Type)
			}
			if _, ok := d.Actors[action.ActorID]; !ok {
				return fmt.Errorf("%s[%d] actor_id %q does not exist", context, i, action.ActorID)
			}
			if action.Type == "move_actor_to_step" && action.StepID == "" {
				return fmt.Errorf("%s[%d] step_id is required for type %q", context, i, action.Type)
			}
			if action.Type == "move_actor_to_room" {
				if action.RoomID == "" {
					return fmt.Errorf("%s[%d] room_id is required for type %q", context, i, action.Type)
				}
				if _, ok := d.Map.Rooms[action.RoomID]; !ok {
					return fmt.Errorf("%s[%d] room_id %q does not exist", context, i, action.RoomID)
				}
			}
		case "start_dialogue":
			if action.AssetID == "" {
				return fmt.Errorf("%s[%d] asset_id is required for type %q", context, i, action.Type)
			}
			if _, ok := d.DialogueAssets[action.AssetID]; !ok {
				return fmt.Errorf("%s[%d] asset_id %q does not exist", context, i, action.AssetID)
			}
			if action.StartNode == "" {
				return fmt.Errorf("%s[%d] start_node is required for type %q", context, i, action.Type)
			}
		case "end_dialogue":
			// no-op validation
		default:
			return fmt.Errorf("%s[%d] unsupported action type %q", context, i, action.Type)
		}
	}
	return nil
}

func requireFileFS(fsys fs.FS, path string) error {
	if fsys == nil {
		return nil
	}
	info, err := fs.Stat(fsys, path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory, expected file", path)
	}
	return nil
}

func (d *CampaignDefinition) validateEndings() error {
	endsRun := false
	for _, node := range d.StoryNodes {
		endsRun = endsRun || node.EndsRun
	}
	if endsRun && len(d.Endings) == 0 {
		return fmt.Errorf("a story node has ends_run but the campaign defines no endings")
	}
	seen := map[string]bool{}
	for i, ending := range d.Endings {
		context := fmt.Sprintf("endings[%d]", i)
		if ending.ID == "" {
			return fmt.Errorf("%s id is required", context)
		}
		if seen[ending.ID] {
			return fmt.Errorf("%s id %q is duplicated", context, ending.ID)
		}
		seen[ending.ID] = true
		if ending.Title == "" {
			return fmt.Errorf("%s (%s) title is required", context, ending.ID)
		}
		if ending.DialogueAssetID != "" {
			if _, ok := d.DialogueAssets[ending.DialogueAssetID]; !ok {
				return fmt.Errorf("%s (%s) dialogue_asset_id %q does not exist", context, ending.ID, ending.DialogueAssetID)
			}
			if ending.DialogueStartNode == "" {
				return fmt.Errorf("%s (%s) dialogue_asset_id requires dialogue_start_node", context, ending.ID)
			}
		}
		if err := d.validateConditions(ending.Conditions, context+".conditions"); err != nil {
			return err
		}
	}
	if n := len(d.Endings); n > 0 && len(d.Endings[n-1].Conditions) > 0 {
		return fmt.Errorf("the last ending (%s) must have no conditions so every run has an ending", d.Endings[n-1].ID)
	}
	return nil
}
