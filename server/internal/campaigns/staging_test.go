package campaigns_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"google.golang.org/protobuf/proto"

	yarnpb "drjosh.dev/yarn/bytecode"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
)

// cueLine finds the line of text a cue plays on, as the stage splits it.
func cueLine(text string, line int) string {
	n := 0
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if n == line {
			return l
		}
		n++
	}
	return ""
}

func TestAuthoredStagingPlaysOnItsLine(t *testing.T) {
	def, g, _ := bootstrapBriarwell(t)
	g.Campaign.ActiveNodeID = "birthday"
	text, err := campaigns.EnterActiveNode(def, g)
	if err != nil {
		t.Fatal(err)
	}
	cues := campaigns.DrainStaging(g, text)
	// The scene's intro text comes first; Ros hops on the dialogue's first line.
	if len(cues) != 1 || cues[0].Character != "sister" || cues[0].Emote != "hop" {
		t.Fatalf("opening cues %+v", cues)
	}
	if got := cueLine(text, cues[0].Line); !strings.Contains(got, "The cottage glows") {
		t.Errorf("Ros should hop on the cottage line, got %q", got)
	}
	if again := campaigns.DrainStaging(g, text); len(again) != 0 {
		t.Errorf("drained once, got %+v again", again)
	}

	var give int
	for _, c := range g.Campaign.ActiveDialogue.PendingChoices {
		if strings.Contains(c.Text, "Give Ros") {
			give = c.ID
		}
	}
	text, err = campaigns.ResumeDialogueWithChoice(def, g, give)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"sister:laugh":      "Ros squeals",
		"father:front_door": "Father goes to the window",
		"father:startle":    "There is smoke",
	}
	cues = campaigns.DrainStaging(g, text)
	if len(cues) != len(want) {
		t.Fatalf("cues %+v", cues)
	}
	for _, c := range cues {
		key := c.Character + ":" + c.Emote + c.Mark
		if got := cueLine(text, c.Line); !strings.Contains(got, want[key]) {
			t.Errorf("%s plays on %q, want the line with %q", key, got, want[key])
		}
	}
	if m := g.NPCs["father"].MarkHere(); m != "front_door" {
		t.Errorf("Father should stand by the door, got %q", m)
	}
}

func TestStagingCommandsAreCheckedOnLoad(t *testing.T) {
	program := func(cmd string) []byte {
		prog := &yarnpb.Program{Name: "t", Nodes: map[string]*yarnpb.Node{
			"Start": {Name: "Start", Instructions: []*yarnpb.Instruction{{
				Opcode:   yarnpb.Instruction_RUN_COMMAND,
				Operands: []*yarnpb.Operand{{Value: &yarnpb.Operand_StringValue{StringValue: cmd}}},
			}}},
		}}
		b, err := proto.Marshal(prog)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	reg, err := campaigns.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatal(err)
	}
	base, _ := reg.Get("briarwell")
	for cmd, wantErr := range map[string]string{
		"move father front_door": "",
		"emote player bow":       "",
		"move father the_moon":   "no point of interest",
		"emote nobody hop":       "no actor",
		"emote sister pirouette": "emote must be one of",
		"move father":            "needs a character and a mark",
	} {
		def := *base
		def.SourceFS = overlay{extra: fstest.MapFS{"t.yarnc": {Data: program(cmd)}, "t.csv": {Data: []byte("id,text\n")}}, base: base.SourceFS}
		def.DialogueAssets = map[string]campaigns.DialogueAsset{}
		for id, a := range base.DialogueAssets {
			def.DialogueAssets[id] = a
		}
		def.DialogueAssets["t"] = campaigns.DialogueAsset{Format: "yarn", Program: "t.yarnc", Strings: "t.csv"}
		err := def.Validate()
		if wantErr == "" && err != nil {
			t.Errorf("<<%s>>: %v", cmd, err)
		}
		if wantErr != "" && (err == nil || !strings.Contains(err.Error(), wantErr)) {
			t.Errorf("<<%s>>: want %q, got %v", cmd, wantErr, err)
		}
	}
}

// overlay serves extra files over a campaign's own.
type overlay struct {
	extra fstest.MapFS
	base  fs.FS
}

func (o overlay) Open(name string) (fs.File, error) {
	if _, ok := o.extra[name]; ok {
		return o.extra.Open(name)
	}
	return o.base.Open(name)
}
