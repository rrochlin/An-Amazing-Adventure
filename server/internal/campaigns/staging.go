package campaigns

import (
	"fmt"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/dialogue"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Staging commands in authored dialogue, the same cues the game master
// hands in (internal/story/gm):
//
//	<<move father front_door>>   the character walks to a mark in their room
//	<<emote ros hop>>            the character reacts
//
// The character is an actor ID or "player". A cue plays with the line
// after the command.
const (
	stageMove  = "move"
	stageEmote = "emote"
)

// Emotes are the reactions the stage can play.
var Emotes = []string{"hop", "shake", "startle", "bow", "laugh", "cry"}

// validateStaging checks the staging commands in the campaign's dialogue:
// a known character, a mark that is one of the map's points of interest,
// a known emote.
func (d *CampaignDefinition) validateStaging() error {
	if d.SourceFS == nil {
		return nil
	}
	marks := map[string]bool{}
	for _, room := range d.Map.Rooms {
		for _, p := range room.PointsOfInterest {
			marks[p.ID] = true
		}
	}
	emotes := map[string]bool{}
	for _, e := range Emotes {
		emotes[e] = true
	}
	for id, asset := range d.DialogueAssets {
		commands, err := dialogue.Commands(d.SourceFS, asset.Program)
		if err != nil {
			continue // a missing program is reported where it's used
		}
		for _, cmd := range commands {
			parts := strings.Fields(cmd)
			if len(parts) == 0 || (parts[0] != stageMove && parts[0] != stageEmote) {
				continue
			}
			if len(parts) != 3 {
				return fmt.Errorf("dialogue %q: <<%s>> needs a character and a %s", id, cmd, map[string]string{stageMove: "mark", stageEmote: "emote"}[parts[0]])
			}
			if _, ok := d.Actors[parts[1]]; !ok && parts[1] != "player" {
				return fmt.Errorf("dialogue %q: <<%s>>: no actor %q", id, cmd, parts[1])
			}
			if parts[0] == stageMove && !marks[parts[2]] {
				return fmt.Errorf("dialogue %q: <<%s>>: no point of interest %q", id, cmd, parts[2])
			}
			if parts[0] == stageEmote && !emotes[parts[2]] {
				return fmt.Errorf("dialogue %q: <<%s>>: emote must be one of %s", id, cmd, strings.Join(Emotes, ", "))
			}
		}
	}
	return nil
}

// takeStaging splits a dialogue run's staging commands from the rest: the
// moves are applied (the character stands at the mark), the cues are kept on
// g for the message the text ends up in, and the other commands are returned
// for the action executor.
func takeStaging(g *game.Game, result dialogue.RunResult) []string {
	var rest []string
	var cues []game.WorldEvent
	for i, cmd := range result.Commands {
		parts := strings.Fields(cmd)
		if len(parts) != 3 || (parts[0] != stageMove && parts[0] != stageEmote) {
			rest = append(rest, cmd)
			continue
		}
		line := 0
		if i < len(result.CommandLines) {
			line = result.CommandLines[i]
		}
		who := parts[1]
		cue := game.WorldEvent{Type: "cue", Character: who, Line: line}
		if parts[0] == stageMove {
			cue.Mark = parts[2]
			standAt(g, who, parts[2])
		} else {
			cue.Emote = parts[2]
		}
		cues = append(cues, cue)
	}
	if len(cues) > 0 {
		g.PendingStaging = append(g.PendingStaging, game.StagingBatch{Text: strings.Join(result.Lines, "\n"), Cues: cues})
	}
	return rest
}

func standAt(g *game.Game, who, mark string) {
	if who == "player" {
		if owner, ok := g.OwnerCharacter(); ok {
			owner.SetMark(mark)
			g.SetPlayerCharacter(g.OwnerID, owner)
		}
		return
	}
	if npc, ok := g.NPCs[who]; ok {
		npc.SetMark(mark)
		g.NPCs[who] = npc
	}
}

// DrainStaging hands over the staging authored dialogue ran into, placed in
// the message it is shown in: each cue's Line becomes a line of text (blank
// lines don't count, as the stage splits them), clamped to its last line.
func DrainStaging(g *game.Game, text string) []game.WorldEvent {
	if g == nil || len(g.PendingStaging) == 0 {
		return nil
	}
	total := countLines(text)
	var out []game.WorldEvent
	for _, b := range g.PendingStaging {
		offset := 0
		if at := strings.Index(text, b.Text); at > 0 {
			offset = countLines(text[:at])
		}
		for _, c := range b.Cues {
			c.Line += offset
			if total > 0 && c.Line >= total {
				c.Line = total - 1
			}
			out = append(out, c)
		}
	}
	g.PendingStaging = nil
	return out
}

func countLines(text string) int {
	n := 0
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
