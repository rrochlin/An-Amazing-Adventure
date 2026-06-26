package dialogue_test

import (
	"testing"
	"testing/fstest"

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

func TestRunNode_ResumeDoesNotRepeatPreChoiceCommands(t *testing.T) {
	fsys := fstest.MapFS{
		"resume-bug.yarnc": &fstest.MapFile{Data: []byte(`name: "resume-bug"
nodes: {
  key: "ChoiceNode"
  value: {
    name: "ChoiceNode"
    instructions: {
      opcode: RUN_COMMAND
      operands: { string_value: "set_flag prompt_seen true" }
    }
    instructions: {
      opcode: RUN_LINE
      operands: { string_value: "choice_prompt" }
    }
    instructions: {
      opcode: ADD_OPTION
      operands: { string_value: "choice_hidden" }
      operands: { string_value: "HiddenSelected" }
    }
    instructions: { opcode: SHOW_OPTIONS }
    instructions: { opcode: JUMP }
    instructions: { opcode: STOP }
    instructions: {
      opcode: RUN_COMMAND
      operands: { string_value: "set_label entry_route hidden" }
    }
    instructions: {
      opcode: RUN_LINE
      operands: { string_value: "choice_ack" }
    }
    instructions: { opcode: STOP }
    labels: {
      key: "HiddenSelected"
      value: 6
    }
  }
}`)},
		"resume-bug.csv": &fstest.MapFile{Data: []byte("id,text\nchoice_prompt,\"Choose a route.\"\nchoice_hidden,\"Take the hidden route.\"\nchoice_ack,\"You slip into the shadows.\"\n")},
	}

	paused, err := dialogue.RunNode(fsys, "resume-bug.yarnc", "resume-bug.csv", "ChoiceNode", nil, nil)
	if err != nil {
		t.Fatalf("run node (pause): %v", err)
	}
	if len(paused.Commands) != 1 || paused.Commands[0] != "set_flag prompt_seen true" {
		t.Fatalf("expected pre-choice command before pause, got %#v", paused.Commands)
	}
	if len(paused.PendingChoices) != 1 {
		t.Fatalf("expected one pending choice, got %#v", paused.PendingChoices)
	}

	choiceID := paused.PendingChoices[0].ID
	resumed, err := dialogue.RunNode(fsys, "resume-bug.yarnc", "resume-bug.csv", "ChoiceNode", paused.Variables, &choiceID)
	if err != nil {
		t.Fatalf("run node (resume): %v", err)
	}
	if len(resumed.Commands) != 1 || resumed.Commands[0] != "set_label entry_route hidden" {
		t.Fatalf("expected only post-choice command after resume, got %#v", resumed.Commands)
	}
	if len(resumed.Lines) != 1 || resumed.Lines[0] != "You slip into the shadows." {
		t.Fatalf("expected post-choice line after resume, got %#v", resumed.Lines)
	}
}
