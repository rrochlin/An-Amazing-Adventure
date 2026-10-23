package campaigns

import (
	"fmt"
	"sort"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

func errorf(format string, args ...any) error { return fmt.Errorf(format, args...) }

func quote(s string) string { return fmt.Sprintf("%q", s) }

// Normalize fills in the scene-script defaults after loading: each scene's
// control level, and stage directions from the older narrator_contract.
func (d *CampaignDefinition) Normalize() {
	for id, node := range d.StoryNodes {
		node.Control = SceneControl(node)
		if isZeroDirections(node.StageDirections) {
			c := node.NarratorContract
			node.StageDirections = StageDirections{
				Focus:         c.SceneFocus,
				MustMention:   c.MustMention,
				MustNotReveal: c.MustNotReveal,
				Tone:          c.ToneNotes,
			}
		}
		d.StoryNodes[id] = node
	}
}

// SceneControl is a scene's control level, defaulting to controlled for
// yarn_dialogue scenes and bounded otherwise.
func SceneControl(node StoryNode) string {
	if node.Control != "" {
		return node.Control
	}
	if node.Mode == "yarn_dialogue" {
		return ControlControlled
	}
	return ControlBounded
}

func isZeroDirections(s StageDirections) bool {
	return s.Focus == "" && s.Setting == "" && len(s.MustMention) == 0 && len(s.MustHappen) == 0 &&
		len(s.MustNotReveal) == 0 && len(s.Boundaries) == 0 && len(s.Tone) == 0
}

// Tier returns the agent tier for an actor in a scene: the scene cast's
// override, else the actor's default, else actor.
func (d *CampaignDefinition) Tier(node StoryNode, actorID string) string {
	for _, m := range node.Cast {
		if m.ActorID == actorID && m.Tier != "" {
			return m.Tier
		}
	}
	if a, ok := d.Actors[actorID]; ok && a.DefaultTier != "" {
		return a.DefaultTier
	}
	return TierActor
}

// beatSpecs turns a scene's beats into allowed condition specs carrying
// their meanings.
func beatSpecs(beats []Beat) []AllowedConditionSpec {
	specs := make([]AllowedConditionSpec, 0, len(beats))
	for _, b := range beats {
		switch b.Kind {
		case "label":
			spec := AllowedConditionSpec{Type: "label_equals", Key: b.Key, Description: b.Description, Meanings: map[string]string{}}
			for _, o := range b.Outcomes {
				spec.AllowedValues = append(spec.AllowedValues, o.Value)
				spec.Meanings[o.Value] = o.Meaning
			}
			specs = append(specs, spec)
		case "flag":
			specs = append(specs, AllowedConditionSpec{
				Type: "flag_true", Key: b.Key, Description: b.Description,
				Meanings: map[string]string{"true": b.Meaning},
			})
		}
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Key < specs[j].Key })
	return specs
}

var validTiers = map[string]bool{TierActor: true, TierPlayer: true}

// validateScript checks the scene-script parts of a story node.
func (d *CampaignDefinition) validateScript(nodeID string, node StoryNode) error {
	where := "story node " + quote(nodeID)
	control := SceneControl(node)
	if control != ControlControlled && control != ControlBounded {
		return errorf("%s control %q must be %q or %q", where, control, ControlControlled, ControlBounded)
	}
	// A controlled scene without exact lines needs events to play as written.
	if control == ControlControlled && node.Mode != "yarn_dialogue" && len(node.StageDirections.MustHappen) == 0 {
		return errorf("%s is controlled but has no dialogue and no stage_directions.must_happen", where)
	}

	for _, ev := range node.StageDirections.AllowedWorldEvents {
		if !containsString(WorldEventTypes, ev) {
			return errorf("%s stage_directions.allowed_world_events has unknown type %q", where, ev)
		}
	}

	seenCast := map[string]bool{}
	for _, m := range node.Cast {
		actor, ok := d.Actors[m.ActorID]
		if !ok {
			return errorf("%s cast references unknown actor %q", where, m.ActorID)
		}
		if seenCast[m.ActorID] {
			return errorf("%s cast lists %q twice", where, m.ActorID)
		}
		seenCast[m.ActorID] = true
		if m.Tier != "" && !validTiers[m.Tier] {
			return errorf("%s cast %q tier %q must be %q or %q", where, m.ActorID, m.Tier, TierActor, TierPlayer)
		}
		if len(actor.AppearsIn) > 0 && !containsString(actor.AppearsIn, nodeID) {
			return errorf("%s casts %q, whose appears_in doesn't include this scene", where, m.ActorID)
		}
	}

	beats := map[string]Beat{}
	for _, b := range node.Beats {
		if b.Key == "" {
			return errorf("%s has a beat without a key", where)
		}
		if _, dup := beats[b.Key]; dup {
			return errorf("%s has duplicate beat %q", where, b.Key)
		}
		beats[b.Key] = b
		if b.Description == "" {
			return errorf("%s beat %q needs a description", where, b.Key)
		}
		switch b.Kind {
		case "label":
			if len(b.Outcomes) == 0 {
				return errorf("%s label beat %q needs outcomes", where, b.Key)
			}
			values := map[string]bool{}
			for _, o := range b.Outcomes {
				if o.Value == "" || o.Meaning == "" {
					return errorf("%s beat %q has an outcome without a value or meaning", where, b.Key)
				}
				if values[o.Value] {
					return errorf("%s beat %q has duplicate outcome %q", where, b.Key, o.Value)
				}
				values[o.Value] = true
			}
		case "flag":
			if b.Meaning == "" {
				return errorf("%s flag beat %q needs a meaning", where, b.Key)
			}
		default:
			return errorf("%s beat %q kind %q must be label or flag", where, b.Key, b.Kind)
		}
	}

	// Once a scene has beats, every label and flag its rules read must be a
	// beat, so every outcome has a written meaning.
	if len(beats) > 0 {
		for _, rule := range append(append([]TransitionRule{}, node.CompletionRules...), node.FailureRules...) {
			for _, c := range rule.Conditions {
				switch c.Type {
				case "label_equals", "label_in":
					b, ok := beats[c.Key]
					if !ok || b.Kind != "label" {
						return errorf("%s rules read label %q, which isn't one of its label beats", where, c.Key)
					}
					for _, v := range labelValues(c) {
						if !hasOutcome(b, v) {
							return errorf("%s rules read %q = %q, which beat %q doesn't define", where, c.Key, v, c.Key)
						}
					}
				case "flag_true", "flag_false":
					if b, ok := beats[c.Key]; !ok || b.Kind != "flag" {
						return errorf("%s rules read flag %q, which isn't one of its flag beats", where, c.Key)
					}
				}
			}
		}
	}
	return nil
}

// validateSheet checks an actor's character sheet.
func (d *CampaignDefinition) validateSheet(actorID string, a ActorTemplate) error {
	if !game.ValidScale(a.Scale) {
		return errorf("actor %q scale %v must be between 0 and %v", actorID, a.Scale, game.MaxScale)
	}
	if a.DefaultTier != "" && !validTiers[a.DefaultTier] {
		return errorf("actor %q default_tier %q must be %q or %q", actorID, a.DefaultTier, TierActor, TierPlayer)
	}
	for other := range a.Relationships {
		if other == "player" {
			continue
		}
		if _, ok := d.Actors[other]; !ok {
			return errorf("actor %q has a relationship with unknown actor %q", actorID, other)
		}
	}
	return nil
}

func labelValues(c Condition) []string {
	values := append(append([]string{}, c.Values...), c.AllowedVals...)
	if s := stringifyValue(c.Value); s != "" {
		values = append(values, s)
	}
	return values
}

func hasOutcome(b Beat, value string) bool {
	for _, o := range b.Outcomes {
		if o.Value == value {
			return true
		}
	}
	return false
}

func containsString(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
