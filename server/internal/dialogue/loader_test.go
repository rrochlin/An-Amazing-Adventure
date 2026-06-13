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
	result, err := dialogue.RunNode(def.SourceFS, asset.Program, asset.Strings, "RouteChoice", nil)
	if err != nil {
		t.Fatalf("run node: %v", err)
	}
	if !result.Completed {
		t.Fatal("expected dialogue run to complete")
	}
	if result.CurrentNode != "RouteChoice" {
		t.Fatalf("expected current node RouteChoice, got %q", result.CurrentNode)
	}
	if len(result.Lines) != 1 {
		t.Fatalf("expected one dialogue line, got %#v", result.Lines)
	}
	if result.Lines[0] != "Route prompt: send hidden, target, or novice to test label-based branching." {
		t.Fatalf("unexpected dialogue lines %#v", result.Lines)
	}
	if len(result.Commands) != 0 {
		t.Fatalf("expected no commands, got %#v", result.Commands)
	}
}
