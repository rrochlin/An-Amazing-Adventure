package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
	"github.com/rrochlin/an-amazing-adventure/internal/trace"
)

// render writes the side-by-side Markdown report.
func render(w io.Writer, save game.SaveState, candidate string, results []result) {
	p := func(format string, args ...any) { fmt.Fprintf(w, format+"\n", args...) }

	replayed, skipped, origOver, candOver, missing := 0, 0, 0, 0, 0
	var origTotal, candTotal time.Duration
	for _, r := range results {
		if r.Skipped != "" {
			skipped++
			continue
		}
		replayed++
		o := originalLatency(r.Trace)
		origTotal += o
		candTotal += r.Elapsed
		if o > Budget {
			origOver++
		}
		if r.Elapsed > Budget {
			candOver++
		}
		if len(r.Missing) > 0 {
			missing++
		}
	}

	p("# Replay of %s — %s", save.SessionID, orDash(save.Title))
	p("")
	p("Candidate pipeline: **%s**. Latency budget: %s per turn.", candidate, Budget)
	p("")
	p("- turns replayed: %d, skipped: %d, rebuilt with missing data: %d", replayed, skipped, missing)
	if replayed > 0 {
		p("- average latency: original %s, candidate %s", seconds(origTotal/time.Duration(replayed)), seconds(candTotal/time.Duration(replayed)))
		p("- over budget: original %d, candidate %d", origOver, candOver)
	}

	for i, r := range results {
		t := r.Trace
		p("")
		p("## %d. %s `%s` — %s", i+1, t.Kind, oneLine(t.Input), time.UnixMilli(t.StartedAt).UTC().Format("2006-01-02 15:04:05Z"))
		if r.Skipped != "" {
			p("")
			p("_Skipped: %s_", r.Skipped)
			continue
		}
		for _, m := range r.Missing {
			p("")
			p("> **Not in the trace:** %s", m)
		}

		p("")
		p("| | Original | Candidate |")
		p("|---|---|---|")
		p("| Latency | %s | %s |", budgeted(originalLatency(t)), budgeted(r.Elapsed))
		oIn, oOut := callTokens(t.Calls)
		cIn, cOut := callTokens(r.Output.Calls)
		p("| Tokens in / out | %d / %d | %d / %d |", oIn, oOut, cIn, cOut)
		p("| Model calls | %s | %s |", callSummary(t.Calls), callSummary(r.Output.Calls))
		p("| Errors | %s | %s |", errorSummary(t.Errors, nil), errorSummary(r.Output.Errors, r.Err))

		p("")
		p("**State, original**")
		p("")
		for _, line := range trace.Changes(t.Before, t.After) {
			p("- %s", line)
		}
		p("")
		p("**State, candidate**")
		p("")
		for _, line := range trace.Changes(r.Before, r.After) {
			p("- %s", line)
		}

		p("")
		p("**Narration, original**")
		p("")
		quote(w, originalNarration(t))
		p("")
		p("**Narration, candidate**")
		p("")
		quote(w, r.Output.Narrative)
		if r.Output.AuthoredDialogue != "" {
			p("")
			p("**Authored dialogue, candidate**")
			p("")
			quote(w, r.Output.AuthoredDialogue)
		}
	}
}

// originalLatency is the time the original turn spent in model calls, or
// the whole turn's duration when no calls were recorded.
func originalLatency(t *trace.Turn) time.Duration {
	var ms int64
	for _, c := range t.Calls {
		ms += c.DurationMs
	}
	if ms == 0 {
		ms = t.DurationMs
	}
	return time.Duration(ms) * time.Millisecond
}

func originalNarration(t *trace.Turn) string {
	for _, c := range t.Calls {
		if c.Role == "narrator" {
			return c.Output
		}
	}
	return ""
}

func callTokens(calls []trace.ModelCall) (in, out int) {
	for _, c := range calls {
		in += c.InputTok
		out += c.OutputTok
	}
	return in, out
}

func callSummary(calls []trace.ModelCall) string {
	if len(calls) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(calls))
	for _, c := range calls {
		s := fmt.Sprintf("%s %s", c.Role, seconds(time.Duration(c.DurationMs)*time.Millisecond))
		if c.Error != "" {
			s += " (failed)"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func errorSummary(errs []string, err error) string {
	if err != nil {
		errs = append(append([]string(nil), errs...), "failed: "+err.Error())
	}
	if len(errs) == 0 {
		return "none"
	}
	return strings.ReplaceAll(strings.Join(errs, "; "), "|", "\\|")
}

func budgeted(d time.Duration) string {
	mark := "within budget"
	if d > Budget {
		mark = "**over budget**"
	}
	return seconds(d) + " (" + mark + ")"
}

func seconds(d time.Duration) string {
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func quote(w io.Writer, s string) {
	if strings.TrimSpace(s) == "" {
		fmt.Fprintln(w, "> _(none)_")
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		fmt.Fprintln(w, "> "+line)
	}
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
