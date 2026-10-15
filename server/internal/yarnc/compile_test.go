package yarnc_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rrochlin/an-amazing-adventure/internal/dialogue"
	"github.com/rrochlin/an-amazing-adventure/internal/yarnc"
)

const sample = `title: Start
---
// comments are ignored
Hello there. #line:hello
-> Ask about the road
    <<set_flag asked_road>>
    "It's long," she says.
    -> Go anyway
        <<set_label route long>>
    -> Turn back
        <<jump Home>>
-> Leave
    Goodbye.
After the choice.
===
title: Home
---
You head home.
===
`

// run compiles src and executes startNode through the real dialogue
// runtime, optionally choosing an option by index at each pause.
func run(t *testing.T, src, startNode string, choices ...int) dialogue.RunResult {
	t.Helper()
	prog, err := yarnc.Compile("sample", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	fsys := fstest.MapFS{
		"p.yarnc": {Data: prog.ProgramText()},
		"p.csv":   {Data: prog.StringsCSV()},
	}
	res, err := dialogue.RunNode(fsys, "p.yarnc", "p.csv", startNode, nil, nil)
	if err != nil {
		t.Fatalf("run %s: %v", startNode, err)
	}
	for _, idx := range choices {
		if len(res.PendingChoices) == 0 {
			t.Fatalf("expected pending choices before choosing %d, got %#v", idx, res)
		}
		id := res.PendingChoices[idx].ID
		res, err = dialogue.RunNode(fsys, "p.yarnc", "p.csv", res.CurrentNode, res.Variables, append(res.ChoiceHistory, id))
		if err != nil {
			t.Fatalf("resume with choice %d: %v", idx, err)
		}
	}
	return res
}

func TestCompile_LinesAndOptionsRunOnVM(t *testing.T) {
	res := run(t, sample, "Start")
	if got := strings.Join(res.Lines, "|"); got != "Hello there." {
		t.Fatalf("expected opening line before options, got %q", got)
	}
	if len(res.PendingChoices) != 2 || res.PendingChoices[0].Text != "Ask about the road" {
		t.Fatalf("expected two top-level options, got %#v", res.PendingChoices)
	}

	res = run(t, sample, "Start", 1)
	if got := strings.Join(res.Lines, "|"); got != "Goodbye.|After the choice." {
		t.Fatalf("expected option body then shared tail, got %q", got)
	}
}

func TestCompile_NestedOptionsAndCommands(t *testing.T) {
	res := run(t, sample, "Start", 0)
	if !contains(res.Commands, "set_flag asked_road") {
		t.Fatalf("expected set_flag command, got %#v", res.Commands)
	}
	if !contains(res.Lines, `"It's long," she says.`) {
		t.Fatalf("expected quoted line intact, got %#v", res.Lines)
	}
	if len(res.PendingChoices) != 2 || res.PendingChoices[1].Text != "Turn back" {
		t.Fatalf("expected nested options, got %#v", res.PendingChoices)
	}
}

func TestCompile_JumpRunsOtherNode(t *testing.T) {
	res := run(t, sample, "Start", 0, 1)
	if !contains(res.Lines, "You head home.") {
		t.Fatalf("expected jump to Home, got %#v", res.Lines)
	}
	if contains(res.Lines, "After the choice.") {
		t.Fatalf("jump should leave Start for good, got %#v", res.Lines)
	}
}

func TestCompile_IsDeterministic(t *testing.T) {
	a, err := yarnc.Compile("sample", sample)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := yarnc.Compile("sample", sample)
	if !bytes.Equal(a.ProgramText(), b.ProgramText()) || !bytes.Equal(a.StringsCSV(), b.StringsCSV()) {
		t.Fatal("compiling the same source twice produced different output")
	}
}

func TestCompile_RejectsUnsupportedSyntax(t *testing.T) {
	cases := map[string]string{
		"set":            "title: A\n---\n<<set $x to 1>>\n===\n",
		"if":             "title: A\n---\n<<if true>>\nHi\n<<endif>>\n===\n",
		"interpolation":  "title: A\n---\nHello {$name}\n===\n",
		"option cond":    "title: A\n---\n-> Pick me <<if $ok>>\n===\n",
		"missing title":  "foo: bar\n---\nHi\n===\n",
		"unknown jump":   "title: A\n---\n<<jump Nowhere>>\n===\n",
		"duplicate id":   "title: A\n---\nOne #line:x\nTwo #line:x\n===\n",
		"duplicate node": "title: A\n---\nHi\n===\ntitle: A\n---\nHi\n===\n",
		"unterminated":   "title: A\n---\nHi\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := yarnc.Compile("bad", src); err == nil {
				t.Fatalf("expected an error for %s", name)
			}
		})
	}
}

// TestCampaignDialogueIsCompiled fails when a .yarn source under campaigns/
// has been edited without recompiling.
func TestCampaignDialogueIsCompiled(t *testing.T) {
	root := filepath.Join("..", "..", "campaigns")
	sources, err := yarnc.FindSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) == 0 {
		t.Fatal("expected .yarn sources under campaigns/")
	}
	for _, src := range sources {
		out, err := yarnc.CompileFile(src)
		if err != nil {
			t.Fatal(err)
		}
		for path, want := range map[string][]byte{out.ProgramPath: out.Program, out.StringsPath: out.Strings} {
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("%s is stale: run `go run ./tools/yarnc` from server/", path)
			}
		}
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
