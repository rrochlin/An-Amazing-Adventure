package campaigns_test

import (
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func TestAllowedConditionSpecsForNode_DerivesRuleConditions(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("lichs-labyrinth")
	if !ok {
		t.Fatal("expected lichs-labyrinth campaign")
	}
	node := def.StoryNodes["choose_entry_plan"]
	specs := campaigns.AllowedConditionSpecsForNode(node)
	if len(specs) < 2 {
		t.Fatalf("expected derived condition specs, got %#v", specs)
	}
	byKey := map[string]campaigns.AllowedConditionSpec{}
	for _, spec := range specs {
		byKey[spec.Key] = spec
	}
	entryRoute, ok := byKey["entry_route"]
	if !ok {
		t.Fatalf("expected entry_route spec, got %#v", byKey)
	}
	if len(entryRoute.AllowedValues) != 3 {
		t.Fatalf("expected entry_route allowed values to be preserved, got %#v", entryRoute.AllowedValues)
	}
}

func TestAdvanceCampaign_TransitionsAndUpdatesObjectives(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	g, _, err := campaigns.BootstrapGame(
		def,
		"session-1",
		"user-1",
		game.NewCharacter("Hero", ""),
		game.CharacterCreationData{Name: "Hero", RaceID: "human", ClassID: "fighter"},
	)
	if err != nil {
		t.Fatalf("bootstrap game: %v", err)
	}

	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{
		BoolFlags: map[string]bool{"intro_complete": true},
	})

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign: %v", err)
	}
	if !result.Transitioned {
		t.Fatal("expected campaign to transition")
	}
	if g.Campaign.ActiveNodeID != "route_choice" {
		t.Fatalf("expected active node route_choice, got %q", g.Campaign.ActiveNodeID)
	}
	if g.Campaign.CurrentObjective != "Send a message containing hidden, target, or novice to test label-based branching." {
		t.Fatalf("unexpected current objective %q", g.Campaign.CurrentObjective)
	}
	statuses := map[string]string{}
	for _, objective := range g.Campaign.ActiveObjectives {
		statuses[objective.ID] = objective.Status
	}
	if statuses["finish_intro"] != "completed" {
		t.Fatalf("expected finish_intro to be completed, got %#v", statuses)
	}
	if statuses["choose_route"] != "active" {
		t.Fatalf("expected choose_route to be active, got %#v", statuses)
	}
	if g.Campaign.ActiveDialogue == nil || g.Campaign.ActiveDialogue.AssetID != "route_dialogue" {
		t.Fatalf("expected active dialogue to move with node transition, got %#v", g.Campaign.ActiveDialogue)
	}
}

func TestBootstrapGame_SeedsOpeningDialogueLine(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	_, history, err := campaigns.BootstrapGame(
		def,
		"session-bootstrap",
		"user-1",
		game.NewCharacter("Hero", ""),
		game.CharacterCreationData{Name: "Hero", RaceID: "human", ClassID: "fighter"},
	)
	if err != nil {
		t.Fatalf("bootstrap game: %v", err)
	}
	if len(history) < 2 {
		t.Fatalf("expected opening text and dialogue line, got %#v", history)
	}
	if history[1].Content != "Intro prompt: send a message containing proceed to advance the route-selection test." {
		t.Fatalf("expected seeded dialogue line, got %#v", history)
	}
}

func TestAdvanceCampaign_AppliesCounterAndLabelConditions(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("lichs-labyrinth")
	if !ok {
		t.Fatal("expected lichs-labyrinth campaign")
	}
	g := game.NewGame("session-2", "user-1")
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      def.ID,
		CampaignVersion: def.Version,
		ActiveNodeID:    "choose_entry_plan",
		BoolFlags:       map[string]bool{},
		Labels: map[string]string{
			"entry_route":     "hidden_entrance",
			"timing_strategy": "after_target",
		},
		Counters:         map[string]int{},
		ActiveObjectives: []game.ObjectiveState{{ID: "choose_entry_plan", Status: "active", VisibleText: "Choose Entry Plan"}},
		ActorStates:      map[string]game.ActorRuntimeState{},
		CompanionStates:  map[string]game.CompanionState{},
	}

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "floor1_shadow_game" {
		t.Fatalf("expected transition to floor1_shadow_game, got %#v", result)
	}
	if g.Campaign.ActiveNodeID != "floor1_shadow_game" {
		t.Fatalf("expected active node floor1_shadow_game, got %q", g.Campaign.ActiveNodeID)
	}
	statuses := map[string]string{}
	for _, objective := range g.Campaign.ActiveObjectives {
		statuses[objective.ID] = objective.Status
	}
	if statuses["choose_entry_plan"] != "completed" {
		t.Fatalf("expected choose_entry_plan completed, got %#v", statuses)
	}
	if statuses["track_target_party"] != "active" {
		t.Fatalf("expected track_target_party active, got %#v", statuses)
	}
}

func TestAdvanceCampaign_ReturnsDialogueTextForEnteredHybridNode(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("lichs-labyrinth")
	if !ok {
		t.Fatal("expected lichs-labyrinth campaign")
	}
	g := game.NewGame("session-3", "user-1")
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      def.ID,
		CampaignVersion: def.Version,
		ActiveNodeID:    "camp_briefing",
		BoolFlags:       map[string]bool{"briefing_complete": true},
		Labels:          map[string]string{},
		Counters:        map[string]int{},
		ActorStates:     map[string]game.ActorRuntimeState{},
		CompanionStates: map[string]game.CompanionState{},
	}

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "choose_entry_plan" {
		t.Fatalf("expected transition to choose_entry_plan, got %#v", result)
	}
	if result.DialogueText != "Choose your route into the labyrinth." {
		t.Fatalf("expected dialogue text from entered node, got %q", result.DialogueText)
	}
	if g.Campaign.ActiveDialogue == nil || g.Campaign.ActiveDialogue.AssetID != "choose_entry_plan" {
		t.Fatalf("expected active dialogue to be started, got %#v", g.Campaign.ActiveDialogue)
	}
}

func TestAdvanceCampaign_ClearsDialogueWhenEnteringAIScene(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	g := game.NewGame("session-4", "user-1")
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      def.ID,
		CampaignVersion: def.Version,
		ActiveNodeID:    "target_probe",
		BoolFlags:       map[string]bool{"room_verified": true},
		Labels:          map[string]string{"selected_branch": "target"},
		Counters:        map[string]int{},
		ActorStates: map[string]game.ActorRuntimeState{
			"caretaker": {ActorID: "caretaker", CurrentLocationID: "relic_room"},
		},
		CompanionStates: map[string]game.CompanionState{},
		ActiveDialogue:  &game.DialogueRuntimeState{AssetID: "room_dialogue", CurrentNode: "RoomProbe"},
	}

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "complete" {
		t.Fatalf("expected transition to complete, got %#v", result)
	}
	if g.Campaign.ActiveDialogue != nil {
		t.Fatalf("expected active dialogue to be cleared on ai_scene entry, got %#v", g.Campaign.ActiveDialogue)
	}
}

func TestAdvanceCampaign_TestCampaignHiddenBranchAppliesLabelCounterAndObjectiveState(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	g := game.NewGame("session-hidden-1", "user-1")
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      def.ID,
		CampaignVersion: def.Version,
		ActiveNodeID:    "route_choice",
		BoolFlags:       map[string]bool{},
		Labels:          map[string]string{"entry_route": "hidden"},
		Counters:        map[string]int{},
		ActiveObjectives: []game.ObjectiveState{{ID: "choose_route", Status: "active", VisibleText: "Choose Route"}},
		ActorStates: map[string]game.ActorRuntimeState{
			"caretaker": {ActorID: "caretaker", CurrentRailStepID: "camp_idle", CurrentLocationID: "camp"},
		},
		CompanionStates: map[string]game.CompanionState{},
		ActiveDialogue:  &game.DialogueRuntimeState{AssetID: "route_dialogue", CurrentNode: "RouteChoice"},
	}

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign to hidden_probe: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "hidden_probe" {
		t.Fatalf("expected transition to hidden_probe, got %#v", result)
	}
	if g.Campaign.ActiveDialogue != nil {
		t.Fatalf("expected ai_scene hidden_probe to clear dialogue, got %#v", g.Campaign.ActiveDialogue)
	}
	if g.Campaign.Labels["selected_branch"] != "hidden" {
		t.Fatalf("expected selected_branch label to be hidden, got %#v", g.Campaign.Labels)
	}
	if !g.Campaign.BoolFlags["hidden_branch_seen"] {
		t.Fatalf("expected hidden_branch_seen flag to be set, got %#v", g.Campaign.BoolFlags)
	}
	caretaker := g.Campaign.ActorStates["caretaker"]
	if caretaker.CurrentRailStepID != "shadowed" || caretaker.CurrentLocationID != "crossroads" {
		t.Fatalf("expected caretaker to move to shadowed step, got %#v", caretaker)
	}

	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{Counters: map[string]int{"scan_count": 2}})
	result, err = campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign to hidden_verification: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "hidden_verification" {
		t.Fatalf("expected transition to hidden_verification, got %#v", result)
	}
	if result.DialogueText != "Verification prompt: send a message containing verify to confirm objective-status gating." {
		t.Fatalf("expected verification dialogue text, got %q", result.DialogueText)
	}
	if g.Campaign.ActiveDialogue == nil || g.Campaign.ActiveDialogue.AssetID != "verification_dialogue" {
		t.Fatalf("expected verification dialogue to be active, got %#v", g.Campaign.ActiveDialogue)
	}
	statuses := map[string]string{}
	for _, objective := range g.Campaign.ActiveObjectives {
		statuses[objective.ID] = objective.Status
	}
	if statuses["advance_hidden_probe"] != "completed" {
		t.Fatalf("expected advance_hidden_probe completed, got %#v", statuses)
	}
	if statuses["verify_hidden_probe"] != "active" {
		t.Fatalf("expected verify_hidden_probe active, got %#v", statuses)
	}
	if !g.Campaign.BoolFlags["hidden_gate_passed"] {
		t.Fatalf("expected hidden_gate_passed flag to be set, got %#v", g.Campaign.BoolFlags)
	}
	if g.Campaign.Counters["hidden_score"] != 2 {
		t.Fatalf("expected hidden_score counter to be 2, got %#v", g.Campaign.Counters)
	}

	campaigns.ApplyConditionResolution(g.Campaign, campaigns.ConditionResolution{BoolFlags: map[string]bool{"verification_complete": true}})
	result, err = campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign to complete: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "complete" {
		t.Fatalf("expected transition to complete, got %#v", result)
	}
	statuses = map[string]string{}
	for _, objective := range g.Campaign.ActiveObjectives {
		statuses[objective.ID] = objective.Status
	}
	if statuses["verify_hidden_probe"] != "completed" {
		t.Fatalf("expected verify_hidden_probe completed, got %#v", statuses)
	}
	if statuses["complete_test"] != "completed" {
		t.Fatalf("expected complete_test completed, got %#v", statuses)
	}
	if !g.Campaign.BoolFlags["verification_passed"] {
		t.Fatalf("expected verification_passed flag to be set, got %#v", g.Campaign.BoolFlags)
	}
	if g.Campaign.Labels["outcome"] != "hidden_complete" {
		t.Fatalf("expected hidden outcome label, got %#v", g.Campaign.Labels)
	}
}

func TestAdvanceCampaign_TestCampaignFailureRuleFailsObjective(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	g := game.NewGame("session-failure-1", "user-1")
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      def.ID,
		CampaignVersion: def.Version,
		ActiveNodeID:    "route_choice",
		BoolFlags:       map[string]bool{},
		Labels:          map[string]string{"entry_route": "novice"},
		Counters:        map[string]int{},
		ActiveObjectives: []game.ObjectiveState{{ID: "choose_route", Status: "active", VisibleText: "Choose Route"}},
		ActorStates:     map[string]game.ActorRuntimeState{},
		CompanionStates: map[string]game.CompanionState{},
		ActiveDialogue:  &game.DialogueRuntimeState{AssetID: "route_dialogue", CurrentNode: "RouteChoice"},
	}

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign failure rule: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "route_failure" {
		t.Fatalf("expected transition to route_failure, got %#v", result)
	}
	if g.Campaign.ActiveDialogue != nil {
		t.Fatalf("expected route_failure ai_scene to clear dialogue, got %#v", g.Campaign.ActiveDialogue)
	}
	if !g.Campaign.BoolFlags["route_failed"] {
		t.Fatalf("expected route_failed flag to be set, got %#v", g.Campaign.BoolFlags)
	}
	if g.Campaign.Labels["selected_branch"] != "novice" {
		t.Fatalf("expected selected_branch novice, got %#v", g.Campaign.Labels)
	}
	if len(g.Campaign.ActiveObjectives) == 0 || g.Campaign.ActiveObjectives[0].Status != "failed" {
		t.Fatalf("expected choose_route objective to be failed, got %#v", g.Campaign.ActiveObjectives)
	}
}

func TestAdvanceCampaign_TestCampaignTargetBranchRequiresActorRoom(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	g := game.NewGame("session-target-1", "user-1")
	g.Campaign = &game.CampaignRuntimeState{
		CampaignID:      def.ID,
		CampaignVersion: def.Version,
		ActiveNodeID:    "target_probe",
		BoolFlags:       map[string]bool{"room_verified": true},
		Labels:          map[string]string{"selected_branch": "target"},
		Counters:        map[string]int{},
		ActiveObjectives: []game.ObjectiveState{{ID: "inspect_target_probe", Status: "active", VisibleText: "Inspect Target Probe"}},
		ActorStates: map[string]game.ActorRuntimeState{
			"caretaker": {ActorID: "caretaker", CurrentLocationID: "relic_room"},
		},
		CompanionStates: map[string]game.CompanionState{},
		ActiveDialogue:  &game.DialogueRuntimeState{AssetID: "room_dialogue", CurrentNode: "RoomProbe"},
	}

	result, err := campaigns.AdvanceCampaign(def, g)
	if err != nil {
		t.Fatalf("advance campaign target branch: %v", err)
	}
	if !result.Transitioned || result.ToNodeID != "complete" {
		t.Fatalf("expected transition to complete, got %#v", result)
	}
	if g.Campaign.ActiveDialogue != nil {
		t.Fatalf("expected active dialogue cleared on complete, got %#v", g.Campaign.ActiveDialogue)
	}
	statuses := map[string]string{}
	for _, objective := range g.Campaign.ActiveObjectives {
		statuses[objective.ID] = objective.Status
	}
	if statuses["inspect_target_probe"] != "completed" {
		t.Fatalf("expected inspect_target_probe completed, got %#v", statuses)
	}
	if statuses["complete_test"] != "completed" {
		t.Fatalf("expected complete_test completed, got %#v", statuses)
	}
	if !g.Campaign.BoolFlags["target_gate_passed"] {
		t.Fatalf("expected target_gate_passed flag to be set, got %#v", g.Campaign.BoolFlags)
	}
	if g.Campaign.Labels["outcome"] != "target_complete" {
		t.Fatalf("expected target outcome label, got %#v", g.Campaign.Labels)
	}
}

func TestFilterConditionResolution_DropsUnexpectedKeys(t *testing.T) {
	node := campaigns.StoryNode{
		AllowedConditions: []campaigns.AllowedConditionSpec{
			{Type: "flag_true", Key: "briefing_complete"},
			{Type: "label_equals", Key: "entry_route", AllowedValues: []string{"target_entrance"}},
		},
	}
	filtered := campaigns.FilterConditionResolution(
		campaigns.AllowedConditionSpecsForNode(node),
		campaigns.ConditionResolution{
			BoolFlags: map[string]bool{"briefing_complete": true, "invented": true},
			Labels:    map[string]string{"entry_route": "target_entrance", "timing_strategy": "after_target"},
			Counters:  map[string]int{"target_party_progress": 4},
		},
	)
	if len(filtered.BoolFlags) != 1 || !filtered.BoolFlags["briefing_complete"] {
		t.Fatalf("expected only allowed boolean flag, got %#v", filtered.BoolFlags)
	}
	if filtered.Labels["entry_route"] != "target_entrance" || len(filtered.Labels) != 1 {
		t.Fatalf("expected only allowed label, got %#v", filtered.Labels)
	}
	if len(filtered.Counters) != 0 {
		t.Fatalf("expected no counters, got %#v", filtered.Counters)
	}
}
