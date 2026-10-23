package gm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Brief is the game master's system prompt for one turn: its role and the
// off-script policy for this scene's control level, the campaign brief
// (scene, stage directions, cast, places), the world as it stands, and
// how to hand in its decision.
func Brief(def *campaigns.CampaignDefinition, g *game.Game) string {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	owner, _ := g.OwnerCharacter()
	line("You are the game master of a text adventure, running one turn for the player, %s.", owner.Name)
	line("You hand in the whole turn with the %s tool: first the narration (vivid prose, 2-4 paragraphs: sights, sounds, dialogue, action; no option lists, no tallies), then the world changes that actually happened, then the beats that were settled.", ToolName)
	line("Narrate only what the player's action achieves this turn, where they are. Stop at the next decision point and leave the next step to the player: never carry them through a second errand or on to a second place in one turn.")

	node := scene(def, g)
	switch {
	case node == nil:
		line("\nThere is no script: the world is yours to run. Any world change is allowed.")
	case campaigns.SceneControl(*node) == campaigns.ControlControlled:
		line("\n## This scene is CONTROLLED")
		line("It unfolds as written: play its events in order and don't add events of your own.")
		line("If the player tries something the script doesn't cover: acknowledge it in a line, then steer back to the scripted event (off_script: steered). If it can't happen here at all, say so in the fiction (off_script: refused). Never let it change the plot.")
	default:
		line("\n## This scene is BOUNDED")
		line("Improvise freely inside its boundaries (off_script: improvised when the action isn't scripted). An action that would cross a boundary is steered back (off_script: steered); one that can't happen here is refused in the fiction (off_script: refused).")
	}

	if campaignBrief := campaigns.NarratorBrief(def, g); campaignBrief != "" {
		line("\n%s", strings.TrimSpace(campaignBrief))
	}
	line("\n%s", strings.TrimSpace(worldSlice(g)))
	if node != nil && node.SceneMap != "" {
		if sm, ok := def.Map.SceneMaps[node.SceneMap]; ok && owner.LocationID == sm.RoomID {
			line("Indoors: %s. %s", sm.Name, sm.Description)
		}
	}

	line("\n## Handing in your decision")
	line("Allowed world changes in this scene: %s. Report only changes the narration made happen, never ones it mentioned or foreshadowed. Money is handled by the story: never create or give coins.", strings.Join(AllowedWorldEvents(node), ", "))
	if node != nil {
		specs := campaigns.AllowedConditionSpecsForNode(*node)
		if len(specs) > 0 {
			line("Beats you can settle this turn, each only when its meaning has clearly happened:")
			for _, s := range specs {
				line("- %s", describeSpec(s))
			}
		}
	}
	return b.String()
}

func scene(def *campaigns.CampaignDefinition, g *game.Game) *campaigns.StoryNode {
	if def == nil || g.Campaign == nil {
		return nil
	}
	if node, ok := def.StoryNodes[g.Campaign.ActiveNodeID]; ok {
		return &node
	}
	return nil
}

func describeSpec(s campaigns.AllowedConditionSpec) string {
	switch s.Type {
	case "label_equals":
		parts := make([]string, 0, len(s.AllowedValues))
		for _, v := range s.AllowedValues {
			if m, ok := s.Meanings[v]; ok {
				parts = append(parts, fmt.Sprintf("%q when %s", v, strings.TrimSuffix(m, ".")))
			} else {
				parts = append(parts, fmt.Sprintf("%q", v))
			}
		}
		return fmt.Sprintf("%s (%s): %s", s.Key, s.Description, strings.Join(parts, "; "))
	case "flag_true":
		if m, ok := s.Meanings["true"]; ok {
			return fmt.Sprintf("%s (flag): true when %s", s.Key, strings.TrimSuffix(m, "."))
		}
		return s.Key + " (flag)"
	}
	return s.Key
}

// worldSlice is the world as the game master may see it: the player, their
// room, what and who is there. Characters elsewhere are not listed: need to
// know.
func worldSlice(g *game.Game) string {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	owner, _ := g.OwnerCharacter()
	line("## The world right now")
	line("Player: %s, health %d, carrying: %s", owner.Name, owner.Health, names(g, owner.Inventory, "nothing"))
	room, err := g.GetRoom(owner.LocationID)
	if err != nil {
		return b.String()
	}
	exits := make([]string, 0, len(room.Connections))
	for dir, id := range room.Connections {
		if dest, err := g.GetRoom(id); err == nil {
			exits = append(exits, dir+" to "+dest.Name)
		}
	}
	sort.Strings(exits)
	line("Place: %s. %s Exits: %s.", room.Name, room.Description, strings.Join(exits, ", "))
	if len(room.PointsOfInterest) > 0 {
		pois := make([]string, 0, len(room.PointsOfInterest))
		for _, p := range room.PointsOfInterest {
			pois = append(pois, p.Name)
		}
		line("Here: %s.", strings.Join(pois, ", "))
	}
	line("Items here: %s.", names(g, room.Items, "none"))
	people := []string{}
	for _, id := range room.Occupants {
		if npc, err := g.GetNPC(id); err == nil {
			state := ""
			if !npc.Alive {
				state = ", dead"
			}
			people = append(people, fmt.Sprintf("%s (%s%s)", npc.Name, npc.Description, state))
		}
	}
	sort.Strings(people)
	if len(people) == 0 {
		line("No one else is here.")
	} else {
		line("Also here: %s.", strings.Join(people, "; "))
	}
	return b.String()
}

func names(g *game.Game, ids []string, empty string) string {
	out := []string{}
	for _, id := range ids {
		if item, err := g.GetItem(id); err == nil {
			out = append(out, item.Name)
		}
	}
	if len(out) == 0 {
		return empty
	}
	return strings.Join(out, ", ")
}
