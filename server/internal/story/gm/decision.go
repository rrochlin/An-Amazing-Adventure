// Package gm is the game master's contract: what it is told each turn, what
// it returns, and what the scene lets it do. The model call that produces a
// Decision is the pipeline's job (internal/story); this package owns the
// shape and the rules. See docs/story-engine/game-master.md.
package gm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Decision is the structured half of a game master turn: the prose is
// streamed separately, this says what it changed.
type Decision struct {
	// WorldEvents are applied in order through the engineer's tools, after
	// validation against the scene's allowlist.
	WorldEvents []WorldEvent `json:"world_events"`
	// Beats settles the scene's beats this turn: label key → one of its
	// outcomes, flag key → true. Keys the scene doesn't define are dropped.
	Beats map[string]any `json:"beats"`
	// OffScript says how the game master handled an action the script
	// didn't cover, if it did.
	OffScript *OffScript `json:"off_script,omitempty"`
}

// WorldEvent is one world change, named by the tool that applies it.
type WorldEvent struct {
	Type  string         `json:"type"`
	Input map[string]any `json:"input"`
}

// OffScript kinds.
const (
	// OffScriptImprovised: the action wasn't scripted; the game master
	// played it out inside the scene's boundaries.
	OffScriptImprovised = "improvised"
	// OffScriptSteered: the action was acknowledged, then the scene went on
	// as written.
	OffScriptSteered = "steered"
	// OffScriptRefused: the action can't happen here; the game master said
	// so in the fiction.
	OffScriptRefused = "refused"
)

// OffScript records how an unscripted action was handled.
type OffScript struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason,omitempty"`
}

// Default allowlists by control level. A controlled scene can only move
// the player and pass items; a bounded scene can also shape its room and
// deal in items, movement and harm. Who exists and who dies is never a
// default: a scene must say so.
var (
	controlledDefaults = []string{"move_player", "give_item_to_player", "take_item_from_player"}
	boundedDefaults    = append(append([]string{}, controlledDefaults...),
		"move_character", "create_item", "place_item_in_room", "damage_character", "heal_character", "update_room")
)

// AllowedWorldEvents is what a scene lets the game master change: the
// control level's defaults plus the scene's own allowed_world_events. With
// no scene (a procedural game) everything is allowed.
func AllowedWorldEvents(node *campaigns.StoryNode) []string {
	if node == nil {
		return append([]string{}, campaigns.WorldEventTypes...)
	}
	base := boundedDefaults
	if campaigns.SceneControl(*node) == campaigns.ControlControlled {
		base = controlledDefaults
	}
	set := map[string]bool{}
	for _, t := range base {
		set[t] = true
	}
	for _, t := range node.StageDirections.AllowedWorldEvents {
		set[t] = true
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// Validate keeps what the scene permits and reports what it dropped, so a
// game master that oversteps is corrected by code, not trusted.
func Validate(node *campaigns.StoryNode, g *game.Game, d Decision) (Decision, []string) {
	var dropped []string
	allowed := map[string]bool{}
	for _, t := range AllowedWorldEvents(node) {
		allowed[t] = true
	}
	kept := Decision{OffScript: d.OffScript, Beats: map[string]any{}}
	for _, ev := range d.WorldEvents {
		if !allowed[ev.Type] {
			dropped = append(dropped, fmt.Sprintf("world event %q is not allowed in this scene", ev.Type))
			continue
		}
		if room, ok := alreadyThere(g, ev); ok {
			dropped = append(dropped, fmt.Sprintf("move_player: the player is already in %s", room))
			continue
		}
		kept.WorldEvents = append(kept.WorldEvents, ev)
	}

	specs := map[string]campaigns.AllowedConditionSpec{}
	if node != nil {
		for _, s := range campaigns.AllowedConditionSpecsForNode(*node) {
			specs[s.Key] = s
		}
	}
	for key, value := range d.Beats {
		spec, ok := specs[key]
		if !ok {
			dropped = append(dropped, fmt.Sprintf("beat %q is not one of this scene's beats", key))
			continue
		}
		// A settled beat is final: settling it again would rewrite what
		// already happened (and what it paid).
		if was, settled := Settled(g, spec); settled {
			dropped = append(dropped, fmt.Sprintf("beat %q was already settled as %v", key, was))
			continue
		}
		switch spec.Type {
		case "label_equals":
			v, _ := value.(string)
			if !contains(spec.AllowedValues, v) {
				dropped = append(dropped, fmt.Sprintf("beat %q has no outcome %v", key, value))
				continue
			}
			kept.Beats[key] = v
		case "flag_true":
			if value != true {
				dropped = append(dropped, fmt.Sprintf("beat %q is a flag; only true settles it", key))
				continue
			}
			kept.Beats[key] = true
		default:
			dropped = append(dropped, fmt.Sprintf("beat %q (%s) can't be settled by the game master", key, spec.Type))
		}
	}
	if d.OffScript != nil {
		switch d.OffScript.Kind {
		case OffScriptImprovised, OffScriptSteered, OffScriptRefused:
		default:
			dropped = append(dropped, fmt.Sprintf("off_script kind %q is unknown", d.OffScript.Kind))
			kept.OffScript = nil
		}
	}
	return kept, dropped
}

// Resolution turns settled beats into the campaign's condition resolution,
// which the existing orchestration applies.
func Resolution(d Decision) campaigns.ConditionResolution {
	r := campaigns.ConditionResolution{}
	for key, value := range d.Beats {
		switch v := value.(type) {
		case string:
			if r.Labels == nil {
				r.Labels = map[string]string{}
			}
			r.Labels[key] = v
		case bool:
			if r.BoolFlags == nil {
				r.BoolFlags = map[string]bool{}
			}
			r.BoolFlags[key] = v
		}
	}
	return r
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// Settled reports whether a beat already has its outcome, and which.
func Settled(g *game.Game, spec campaigns.AllowedConditionSpec) (any, bool) {
	if g == nil || g.Campaign == nil {
		return nil, false
	}
	switch spec.Type {
	case "label_equals":
		if v := g.Campaign.Labels[spec.Key]; v != "" {
			return v, true
		}
	case "flag_true":
		if g.Campaign.BoolFlags[spec.Key] {
			return true, true
		}
	}
	return nil, false
}

// alreadyThere reports a move_player to the room the player is in.
func alreadyThere(g *game.Game, ev WorldEvent) (string, bool) {
	if g == nil || ev.Type != "move_player" {
		return "", false
	}
	owner, ok := g.OwnerCharacter()
	if !ok {
		return "", false
	}
	room, err := g.GetRoom(owner.LocationID)
	if err != nil {
		return "", false
	}
	target, _ := ev.Input["room_name"].(string)
	target = strings.TrimSpace(target)
	if strings.EqualFold(target, room.Name) || strings.EqualFold(target, room.ID) {
		return room.Name, true
	}
	return "", false
}

// ParseDecision reads the decision a model handed in. Models sometimes
// write a nested value as a string: empty ("") for nothing, or the JSON
// itself quoted ("{\"deed_doll\": \"kept\"}"). Those are read as what they
// mean, and reported in notes; anything else unreadable is an error.
func ParseDecision(raw string) (Decision, []string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return Decision{}, nil, err
	}
	var notes []string
	for _, key := range []string{"world_events", "beats", "off_script"} {
		v, ok := fields[key]
		if !ok {
			continue
		}
		var s string
		if json.Unmarshal(v, &s) != nil {
			continue // not a string: as it should be
		}
		s = strings.TrimSpace(s)
		switch {
		case s == "" || s == "null" || s == "{}" || s == "[]":
			delete(fields, key)
			notes = append(notes, fmt.Sprintf("%s was handed in as %q; read as empty", key, s))
		case json.Valid([]byte(s)):
			fields[key] = json.RawMessage(s)
			notes = append(notes, fmt.Sprintf("%s was handed in as a quoted string; read as JSON", key))
		}
		// Anything else stays, and fails to decode below.
	}
	fixed, err := json.Marshal(fields)
	if err != nil {
		return Decision{}, notes, err
	}
	var d Decision
	if err := json.Unmarshal(fixed, &d); err != nil {
		return Decision{}, notes, err
	}
	return d, notes, nil
}
