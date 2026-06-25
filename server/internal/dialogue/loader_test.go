package dialogue_test

import (
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/dialogue"
)

func TestOpeningLine_ReturnsFirstDialogueString(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	asset := def.DialogueAssets["intro_dialogue"]
	line, err := dialogue.OpeningLine(def.SourceFS, asset.Strings)
	if err != nil {
		t.Fatalf("opening line: %v", err)
	}
	if line != "Intro prompt: send a message containing proceed to advance the route-selection test." {
		t.Fatalf("unexpected opening line %q", line)
	}
}

func TestRunNode_ExecutesCompiledDialogueProgram(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	asset := def.DialogueAssets["route_dialogue"]
	result, err := dialogue.RunNode(def.SourceFS, asset.Program, asset.Strings, "RouteChoice", nil, nil)
	if err != nil {
		t.Fatalf("run node: %v", err)
	}
	if result.Completed {
		t.Fatal("expected route dialogue to pause for player choice")
	}
	if result.CurrentNode != "RouteChoice" {
		t.Fatalf("expected current node RouteChoice, got %q", result.CurrentNode)
	}
	if len(result.Lines) != 1 {
		t.Fatalf("expected one dialogue line, got %#v", result.Lines)
	}
	if result.Lines[0] != "Route prompt: choose Hidden Route, Target Route, or Novice Route." {
		t.Fatalf("unexpected dialogue lines %#v", result.Lines)
	}
	if len(result.PendingChoices) != 3 {
		t.Fatalf("expected three pending choices, got %#v", result.PendingChoices)
	}
	if result.PendingChoices[0].Text != "Take the Hidden Route." {
		t.Fatalf("unexpected first choice text %#v", result.PendingChoices)
	}
	if len(result.Commands) != 0 {
		t.Fatalf("expected no commands before choice selection, got %#v", result.Commands)
	}
}

func TestRunNode_ResumesDialogueWithSelectedChoice(t *testing.T) {
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, ok := reg.Get("test")
	if !ok {
		t.Fatal("expected test campaign")
	}
	asset := def.DialogueAssets["route_dialogue"]
	paused, err := dialogue.RunNode(def.SourceFS, asset.Program, asset.Strings, "RouteChoice", nil, nil)
	if err != nil {
		t.Fatalf("run node (pause): %v", err)
	}
	if len(paused.PendingChoices) != 3 {
		t.Fatalf("expected pending choices before resume, got %#v", paused.PendingChoices)
	}
	targetChoiceID := paused.PendingChoices[1].ID
	resumed, err := dialogue.RunNode(def.SourceFS, asset.Program, asset.Strings, "RouteChoice", paused.Variables, &targetChoiceID)
	if err != nil {
		t.Fatalf("run node (resume): %v", err)
	}
	if !resumed.Completed {
		t.Fatal("expected resumed dialogue to complete")
	}
	if len(resumed.PendingChoices) != 0 {
		t.Fatalf("expected no pending choices after resume, got %#v", resumed.PendingChoices)
	}
	if len(resumed.Commands) != 1 || resumed.Commands[0] != "set_label entry_route target" {
		t.Fatalf("expected target route command, got %#v", resumed.Commands)
	}
	if len(resumed.Lines) != 1 || resumed.Lines[0] != "You commit to the target approach." {
		t.Fatalf("expected post-choice acknowledgement line, got %#v", resumed.Lines)
	}
}
