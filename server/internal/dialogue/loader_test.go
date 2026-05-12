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
	if line != "Welcome to the test campaign." {
		t.Fatalf("unexpected opening line %q", line)
	}
}
