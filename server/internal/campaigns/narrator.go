package campaigns

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// NarratorBrief renders the authored scene package the narrator must stay
// inside: who the player is, where they are, who and what exists in this
// campaign, the current scene's contract, and the active objectives.
//
// Without it the narrator only sees a generic D&D prompt and improvises its
// own world, which defeats the point of an authored campaign.
func NarratorBrief(def *CampaignDefinition, g *game.Game) string {
	if def == nil || g == nil || g.Campaign == nil {
		return ""
	}
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	line("## Authored campaign: %s", def.Title)
	line("Premise: %s", def.Premise)
	if def.Tone != "" {
		line("Tone: %s", def.Tone)
	}

	line("\n## The player character")
	if def.PlayerRole.Name != "" {
		line("Role: %s. %s", def.PlayerRole.Name, def.PlayerRole.RoleSummary)
	}
	if def.CharacterCreation.LockedBackgroundText != "" {
		line("Background (always true, never contradict it): %s", def.CharacterCreation.LockedBackgroundText)
	}

	node, ok := def.StoryNodes[g.Campaign.ActiveNodeID]
	if ok {
		line("\n## Current scene: %s", node.Title)
		if node.Summary != "" {
			line("What is happening: %s", node.Summary)
		}
		if node.ObjectiveText != "" {
			line("The player's current objective: %s", node.ObjectiveText)
		}
		c := node.StageDirections
		if isZeroDirections(c) {
			// Definitions built in code (tests) skip Normalize.
			nc := node.NarratorContract
			c = StageDirections{Focus: nc.SceneFocus, MustMention: nc.MustMention, MustNotReveal: nc.MustNotReveal, Tone: nc.ToneNotes}
		}
		if SceneControl(node) == ControlControlled {
			line("This scene is controlled: it unfolds as written. Don't add events or let it go elsewhere.")
		} else {
			line("This scene is bounded: improvise freely, but stay inside its boundaries.")
		}
		if c.Focus != "" {
			line("Scene focus: %s", c.Focus)
		}
		if c.Setting != "" {
			line("Setting: %s", c.Setting)
		}
		if len(c.MustHappen) > 0 {
			line("These must happen, in order: %s", strings.Join(c.MustHappen, "; "))
		}
		if len(c.MustMention) > 0 {
			line("Work these into the scene: %s", strings.Join(c.MustMention, "; "))
		}
		if len(c.Boundaries) > 0 {
			line("Boundaries (never go outside these): %s", strings.Join(c.Boundaries, "; "))
		}
		if len(c.MustNotReveal) > 0 {
			line("Never reveal or hint at: %s", strings.Join(c.MustNotReveal, "; "))
		}
		for _, note := range c.Tone {
			line("Tone note: %s", note)
		}
		for _, m := range node.Cast {
			if a, ok := def.Actors[m.ActorID]; ok && (m.Part != "" || m.Wants != "") {
				line("In this scene, %s: %s%s", a.Name, m.Part, prefixed(" Wants: ", m.Wants))
			}
		}
	}

	if objectives := activeObjectiveLines(def, g.Campaign); len(objectives) > 0 {
		line("\n## Active objectives")
		for _, o := range objectives {
			line("- %s", o)
		}
	}

	line("\n## Cast (the only named characters in this scene)")
	for _, id := range sortedKeys(def.Actors) {
		a := def.Actors[id]
		if !appearsIn(a, g.Campaign.ActiveNodeID) {
			continue
		}
		// Where the character is: the live world state for cast who exist as
		// characters, otherwise their rail position.
		where := ""
		locationID := ""
		if npc, ok := g.NPCs[id]; ok {
			locationID = npc.LocationID
		} else if st, ok := g.Campaign.ActorStates[id]; ok {
			locationID = st.CurrentLocationID
		}
		if room, ok := def.Map.Rooms[locationID]; ok {
			where = " Currently at: " + room.Name + "."
		}
		who := ""
		if a.Description != "" {
			who = " " + a.Description
		}
		line("- %s (%s):%s Wants: %s%s", a.Name, a.Kind, who, a.Goal, where)
	}

	line("\n## Places (the only locations that exist)")
	for _, id := range sortedKeys(def.Map.Rooms) {
		r := def.Map.Rooms[id]
		exits := make([]string, 0, len(r.Connections))
		for _, dir := range sortedKeys(r.Connections) {
			if target, ok := def.Map.Rooms[r.Connections[dir]]; ok {
				exits = append(exits, dir+" to "+target.Name)
			}
		}
		desc := r.Description
		if len(exits) > 0 {
			desc += " Exits: " + strings.Join(exits, ", ") + "."
		}
		line("- %s: %s", r.Name, strings.TrimSpace(desc))
	}

	line(`
## Rules for this authored campaign
- Stay inside the story above. Do NOT invent new quests, plot threads, rewards, named characters,
  or places. Unnamed background villagers are fine; new named people or locations are not.
- If the player wanders off-script, let them, but steer them back toward the current objective
  through the world (a character calls out, a sound draws them, something they need is elsewhere).
- Never resolve the current objective on the player's behalf, and never skip ahead to later scenes.
- End every reply with a clear, concrete hook toward the current objective: who is nearby, what
  they want, or where the player could go next. Name the specific characters or places involved.`)
	return b.String()
}

func appearsIn(a ActorTemplate, nodeID string) bool {
	if len(a.AppearsIn) == 0 {
		return true
	}
	for _, id := range a.AppearsIn {
		if id == nodeID {
			return true
		}
	}
	return false
}

func activeObjectiveLines(def *CampaignDefinition, state *game.CampaignRuntimeState) []string {
	var out []string
	for _, o := range state.ActiveObjectives {
		if o.Status != "active" {
			continue
		}
		tmpl, ok := def.Objectives[o.ID]
		if !ok {
			continue
		}
		text := tmpl.Title
		if tmpl.Description != "" {
			text += ": " + tmpl.Description
		}
		if tmpl.Optional {
			text += " (optional)"
		}
		out = append(out, text)
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func prefixed(prefix, s string) string {
	if s == "" {
		return ""
	}
	return prefix + s
}
