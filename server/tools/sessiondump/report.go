package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// render writes the Markdown report for one session.
func render(w io.Writer, save game.SaveState, traces []*trace.Turn, mutations []game.MutationEntry, full bool) {
	p := func(format string, args ...any) { fmt.Fprintf(w, format+"\n", args...) }

	title := save.Title
	if title == "" {
		title = "(untitled)"
	}
	p("# Session %s — %s", save.SessionID, title)
	p("")
	p("- turns: %d, tokens: %d, version: %d, schema: %d", save.ConversationCount, save.TotalTokens, save.Version, save.SchemaVersion)
	if c := save.Campaign; c != nil {
		p("- campaign: %s v%s, node: %s (%s), run: %s", c.CampaignID, c.CampaignVersion, c.ActiveNodeID, c.ActiveNodeTitle, orDash(c.RunStatus))
		p("- objective: %s", orDash(c.CurrentObjective))
	}
	p("- chat entries: %d, narrator context messages: %d, traces: %d, mutations: %d",
		len(save.ChatHistory), len(save.Narrative), len(traces), len(mutations))

	p("")
	p("## Transcript")
	p("")
	for i, m := range save.ChatHistory {
		who := "NARRATIVE"
		if m.Type == "player" {
			who = "PLAYER"
		}
		p("%d. **%s** %s", i, who, oneLine(m.Content))
		for _, e := range m.Events {
			p("    - _event %s_: %s", e.Type, e.Message)
		}
	}

	p("")
	p("## Turns")
	if len(traces) == 0 {
		p("")
		p("_No traces (TRACES_TABLE unset, or the session predates tracing)._")
	}
	for i, t := range traces {
		p("")
		p("### %d. %s `%s` — %s (%dms)", i+1, t.Kind, oneLine(t.Input),
			time.UnixMilli(t.StartedAt).UTC().Format("2006-01-02 15:04:05Z"), t.DurationMs)
		p("")
		p("- lambda: %s, request: %s, turn: %d", t.Lambda, orDash(t.RequestID), t.Turn)
		renderStateChange(p, t.Before, t.After)
		for _, c := range t.Calls {
			status := "ok"
			if c.Error != "" {
				status = "ERROR " + c.Error
			}
			p("- %s call (%s): %dms, tokens in %d / out %d, stop %s, %s",
				c.Role, c.Model, c.DurationMs, c.InputTok, c.OutputTok, orDash(c.StopReason), status)
			if c.Role == "resolver" {
				p("    - verdict: `%s`", oneLine(c.Output))
			}
		}
		if t.Resolution != nil {
			p("- resolution: `%s`", compactJSON(t.Resolution))
		}
		if t.Transition != nil {
			p("- transition: `%s`", compactJSON(t.Transition))
		}
		for _, e := range t.Events {
			p("- event %s: %s", e.Type, e.Message)
		}
		p("- frames: %s", frameSummary(t.Frames))
		for _, e := range t.Errors {
			p("- **error**: %s", e)
		}
		if full {
			renderCalls(w, t.Calls)
		}
	}

	if len(mutations) > 0 {
		p("")
		p("## Mutations")
		p("")
		for _, m := range mutations {
			p("- turn %d `%s` %s → %s", m.Turn, m.Tool, compactJSON(m.Input), oneLine(m.Result))
		}
	}
}

// renderStateChange prints what moved in the campaign state during a turn.
func renderStateChange(p func(string, ...any), before, after *trace.Snapshot) {
	for _, line := range trace.Changes(before, after) {
		p("- %s", line)
	}
}

func renderCalls(w io.Writer, calls []trace.ModelCall) {
	for _, c := range calls {
		fmt.Fprintf(w, "\n<details><summary>%s call: prompt and output</summary>\n\n", c.Role)
		if c.System != "" {
			fmt.Fprintf(w, "**System**\n\n```\n%s\n```\n\n", c.System)
		}
		for _, m := range c.Messages {
			var text []string
			for _, b := range m.Content {
				text = append(text, b.Text)
			}
			fmt.Fprintf(w, "**%s**\n\n```\n%s\n```\n\n", m.Role, strings.Join(text, "\n"))
		}
		if c.Input != "" {
			fmt.Fprintf(w, "**Input**\n\n```\n%s\n```\n\n", c.Input)
		}
		fmt.Fprintf(w, "**Output**\n\n```\n%s\n```\n\n</details>\n", c.Output)
	}
}

func frameSummary(frames []trace.Frame) string {
	if len(frames) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(frames))
	for _, f := range frames {
		s := f.Type
		if f.Count > 1 {
			s = fmt.Sprintf("%s×%d", s, f.Count)
		}
		if f.To != "" {
			s += "→" + f.To
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func compactJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
