package trace

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Changes describes what moved between two snapshots, one line per change:
// the player's room, then the campaign's node, objective, objectives, flags,
// labels, counters, dialogue and run status. The room and node lines always
// appear, so an unchanged turn still says where it happened.
func Changes(before, after *Snapshot) []string {
	if before == nil || after == nil {
		return nil
	}
	var out []string
	add := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }

	if before.PlayerRoomID != after.PlayerRoomID {
		add("player room: %s → %s", roomLabel(before), roomLabel(after))
	} else {
		add("player room: %s", roomLabel(after))
	}
	var b, a game.CampaignRuntimeState
	_ = json.Unmarshal(before.Campaign, &b)
	_ = json.Unmarshal(after.Campaign, &a)
	if b.ActiveNodeID != a.ActiveNodeID {
		add("node: %s → **%s**", b.ActiveNodeID, a.ActiveNodeID)
	} else {
		add("node: %s (unchanged)", a.ActiveNodeID)
	}
	if b.CurrentObjective != a.CurrentObjective {
		add("objective: %q → %q", b.CurrentObjective, a.CurrentObjective)
	}
	for _, d := range diffObjectives(b.ActiveObjectives, a.ActiveObjectives) {
		add("objective %s", d)
	}
	out = append(out, diffMap("flag", b.BoolFlags, a.BoolFlags)...)
	out = append(out, diffMap("label", b.Labels, a.Labels)...)
	out = append(out, diffMap("counter", b.Counters, a.Counters)...)
	if bd, ad := dialogueLabel(b.ActiveDialogue), dialogueLabel(a.ActiveDialogue); bd != ad {
		add("dialogue: %s → %s", bd, ad)
	}
	if a.RunStatus != b.RunStatus {
		add("run: %s → %s (ending %s)", orDash(b.RunStatus), a.RunStatus, orDash(a.EndingID))
	}
	return out
}

func diffMap[V comparable](kind string, before, after map[string]V) []string {
	keys := map[string]bool{}
	for k := range before {
		keys[k] = true
	}
	for k := range after {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	var out []string
	for _, k := range sorted {
		b, inB := before[k]
		a, inA := after[k]
		switch {
		case !inB:
			out = append(out, fmt.Sprintf("%s %s set: %v", kind, k, a))
		case !inA:
			out = append(out, fmt.Sprintf("%s %s cleared (was %v)", kind, k, b))
		case a != b:
			out = append(out, fmt.Sprintf("%s %s: %v → %v", kind, k, b, a))
		}
	}
	return out
}

func diffObjectives(before, after []game.ObjectiveState) []string {
	was := map[string]string{}
	for _, o := range before {
		was[o.ID] = o.Status
	}
	var out []string
	for _, o := range after {
		if prev, ok := was[o.ID]; !ok {
			out = append(out, fmt.Sprintf("%s added: %s", o.ID, o.Status))
		} else if prev != o.Status {
			out = append(out, fmt.Sprintf("%s: %s → %s", o.ID, prev, o.Status))
		}
	}
	return out
}

func roomLabel(s *Snapshot) string {
	if s.PlayerRoomName != "" {
		return fmt.Sprintf("%s (%s)", s.PlayerRoomName, s.PlayerRoomID)
	}
	return orDash(s.PlayerRoomID)
}

func dialogueLabel(d *game.DialogueRuntimeState) string {
	if d == nil {
		return "none"
	}
	s := d.AssetID + "/" + d.CurrentNode
	if d.AwaitingChoice {
		s += fmt.Sprintf(" (awaiting choice, %d options)", len(d.PendingChoices))
	}
	return s
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
