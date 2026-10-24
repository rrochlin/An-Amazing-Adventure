package gm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Cue is one piece of staging: someone in the scene walks to a mark, or
// reacts, when the narration reaches a paragraph. The stage plays it; the
// world keeps where they now stand.
type Cue struct {
	// Line is the narration paragraph it plays on, from 0.
	Line int `json:"line"`
	// Who is a character here, by name, or "player". Validation turns a
	// name into the character's ID.
	Who string `json:"who"`
	// MoveTo is a mark (point of interest) in the room, by ID.
	MoveTo string `json:"move_to,omitempty"`
	// Toward is someone here (by name, or "player") they walk up to.
	// Validation turns a name into the character's ID.
	Toward string `json:"toward,omitempty"`
	// Emote is one of Emotes.
	Emote string `json:"emote,omitempty"`
}

// PlayerCue is the Who of a cue for the player's own character.
const PlayerCue = "player"

// Emotes are the reactions the stage can play.
var Emotes = campaigns.Emotes

// stagingSchema is the staging array, narrowed to who is here and the
// room's marks.
func stagingSchema(g *game.Game) map[string]any {
	who := []string{PlayerCue}
	marks := []string{}
	if g != nil {
		for _, npc := range present(g) {
			who = append(who, npc.Name)
		}
		if room, ok := playerRoom(g); ok {
			for _, p := range room.PointsOfInterest {
				marks = append(marks, p.ID)
			}
		}
	}
	cue := map[string]any{
		"line":  map[string]any{"type": "integer", "minimum": 0, "description": "The narration paragraph it plays on, from 0."},
		"who":   map[string]any{"type": "string", "enum": who},
		"emote": map[string]any{"type": "string", "enum": Emotes},
	}
	if len(marks) > 0 {
		cue["move_to"] = map[string]any{"type": "string", "enum": marks, "description": "A mark in this room: they walk there."}
	}
	cue["toward"] = map[string]any{"type": "string", "enum": who, "description": "Someone here: they walk up beside them."}
	return map[string]any{
		"type":        "array",
		"description": "How the scene is staged, in order: who walks to which mark, and who reacts, on which paragraph. Only what the narration shows. Leave it empty when nobody moves or reacts.",
		"items": map[string]any{
			"type":       "object",
			"properties": cue,
			"required":   []string{"line", "who"},
		},
	}
}

// validateStaging keeps the cues the room allows: the character is here
// and alive, the mark is one of the room's, the emote is known. Who comes
// back as an ID ("player" stays).
func validateStaging(g *game.Game, cues []Cue) ([]Cue, []string) {
	var kept []Cue
	var dropped []string
	if g == nil {
		return nil, nil
	}
	room, ok := playerRoom(g)
	if !ok {
		return nil, nil
	}
	marks := map[string]bool{}
	for _, p := range room.PointsOfInterest {
		marks[p.ID] = true
	}
	known := map[string]bool{}
	for _, e := range Emotes {
		known[e] = true
	}
	for _, c := range cues {
		who := strings.TrimSpace(c.Who)
		if strings.EqualFold(who, PlayerCue) {
			c.Who = PlayerCue
		} else if npc, ok := presentNamed(g, who); ok {
			c.Who = npc.ID
		} else {
			dropped = append(dropped, fmt.Sprintf("staging: %q is not here", who))
			continue
		}
		if c.MoveTo != "" && !marks[c.MoveTo] {
			dropped = append(dropped, fmt.Sprintf("staging: %s has no mark %q", room.Name, c.MoveTo))
			c.MoveTo = ""
		}
		if c.Toward != "" {
			to := strings.TrimSpace(c.Toward)
			switch npc, ok := presentNamed(g, to); {
			case strings.EqualFold(to, PlayerCue):
				c.Toward = PlayerCue
			case ok:
				c.Toward = npc.ID
			default:
				dropped = append(dropped, fmt.Sprintf("staging: %q is not here to walk up to", to))
				c.Toward = ""
			}
			if c.Toward == c.Who {
				c.Toward = ""
			}
		}
		if c.Emote != "" && !known[c.Emote] {
			dropped = append(dropped, fmt.Sprintf("staging: no emote %q", c.Emote))
			c.Emote = ""
		}
		if c.MoveTo == "" && c.Emote == "" && c.Toward == "" {
			continue
		}
		if c.Line < 0 {
			c.Line = 0
		}
		kept = append(kept, c)
	}
	return kept, dropped
}

// ApplyStaging stands everyone at the marks the cues sent them to, and
// returns the cues as events for the stage. Line is clamped to the
// narration's paragraphs so a cue past the end plays on the last one.
func ApplyStaging(g *game.Game, cues []Cue, paragraphs int) []game.WorldEvent {
	events := make([]game.WorldEvent, 0, len(cues))
	for _, c := range cues {
		if c.MoveTo != "" {
			if c.Who == PlayerCue {
				if owner, ok := g.OwnerCharacter(); ok {
					owner.SetMark(c.MoveTo)
					g.Players[g.OwnerID] = owner
				}
			} else if npc, ok := g.NPCs[c.Who]; ok {
				npc.SetMark(c.MoveTo)
				g.NPCs[c.Who] = npc
			}
		}
		line := c.Line
		if paragraphs > 0 && line >= paragraphs {
			line = paragraphs - 1
		}
		events = append(events, game.WorldEvent{Type: "cue", Character: c.Who, Mark: c.MoveTo, Toward: c.Toward, Emote: c.Emote, Line: line})
	}
	return events
}

// Paragraphs counts the narration's paragraphs as the stage splits them
// (on line breaks, skipping blank ones).
func Paragraphs(narration string) int {
	n := 0
	for _, l := range strings.Split(narration, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

// stagingBrief tells the game master where everyone stands and how to stage.
func stagingBrief(g *game.Game) string {
	room, ok := playerRoom(g)
	if !ok || len(room.PointsOfInterest) == 0 {
		return ""
	}
	var b strings.Builder
	marks := make([]string, 0, len(room.PointsOfInterest))
	for _, p := range room.PointsOfInterest {
		marks = append(marks, fmt.Sprintf("%s (%s)", p.ID, p.Name))
	}
	fmt.Fprintf(&b, "Marks here: %s. They are all in %s: when someone leaves it this turn (move_player, move_character), stage nothing for them; the stage walks them out.\n", strings.Join(marks, ", "), room.Name)
	var where []string
	if owner, ok := g.OwnerCharacter(); ok && owner.MarkHere() != "" {
		where = append(where, "the player at "+owner.MarkHere())
	}
	for _, npc := range present(g) {
		if m := npc.MarkHere(); m != "" {
			where = append(where, npc.Name+" at "+m)
		}
	}
	if len(where) > 0 {
		fmt.Fprintf(&b, "Standing: %s.\n", strings.Join(where, ", "))
	}
	b.WriteString("Stage the scene: when the narration puts someone somewhere in this room (at the barn, by the well), has them walk there or up to someone (toward), or shows them react (the player and the people they meet alike), add a staging cue for the paragraph that shows it, counting paragraphs from 0 (move_to a mark; emote one of " + strings.Join(Emotes, ", ") + "). Put each spoken line on its own paragraph as Name: words, with no quotation marks, so it comes from that character on the stage.\n")
	return b.String()
}

func playerRoom(g *game.Game) (game.Area, bool) {
	owner, ok := g.OwnerCharacter()
	if !ok {
		return game.Area{}, false
	}
	room, err := g.GetRoom(owner.LocationID)
	if err != nil {
		return game.Area{}, false
	}
	return room, true
}

// present is who is alive in the player's room, by name.
func present(g *game.Game) []game.Character {
	room, ok := playerRoom(g)
	if !ok {
		return nil
	}
	var out []game.Character
	for _, id := range room.Occupants {
		if npc, err := g.GetNPC(id); err == nil && npc.Alive {
			out = append(out, npc)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func presentNamed(g *game.Game, name string) (game.Character, bool) {
	for _, npc := range present(g) {
		if strings.EqualFold(npc.Name, name) || npc.ID == name {
			return npc, true
		}
	}
	return game.Character{}, false
}
