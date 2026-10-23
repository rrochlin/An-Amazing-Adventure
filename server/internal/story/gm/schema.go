package gm

import (
	"github.com/rrochlin/an-amazing-adventure/internal/ai"
	"github.com/rrochlin/an-amazing-adventure/internal/campaigns"
)

// ToolName is the single tool the game master calls to hand in its Decision.
const ToolName = "end_turn"

// Schema is the JSON schema of the whole turn, narrowed to the scene: the
// narration, then only the allowed world events (each with the exact
// arguments its tool takes) and the scene's beats with their outcomes, so
// the model is constrained before validation ever runs.
func Schema(node *campaigns.StoryNode) map[string]any {
	events := []any{}
	for _, name := range AllowedWorldEvents(node) {
		input, ok := ai.ToolInputSchema(name)
		if !ok {
			continue
		}
		events = append(events, map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type":  map[string]any{"const": name},
				"input": input,
			},
			"required": []string{"type", "input"},
		})
	}
	beats := map[string]any{}
	if node != nil {
		for _, s := range campaigns.AllowedConditionSpecsForNode(*node) {
			desc := s.Description
			switch s.Type {
			case "label_equals":
				beats[s.Key] = map[string]any{"type": "string", "enum": s.AllowedValues, "description": withMeanings(desc, s)}
			case "flag_true":
				beats[s.Key] = map[string]any{"type": "boolean", "enum": []bool{true}, "description": withMeanings(desc, s)}
			}
		}
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"narration": map[string]any{
				"type":        "string",
				"description": "The turn's prose, 2-4 paragraphs: what the player experiences. Write this first.",
			},
			"world_events": map[string]any{
				"type":        "array",
				"description": "World changes this turn's narration made happen, in order. Only what actually happened; nothing the narration merely mentioned.",
				"items":       map[string]any{"anyOf": events},
			},
			"beats": map[string]any{
				"type":                 "object",
				"description":          "Beats this turn settled, by key. Settle a beat only when its meaning has clearly happened.",
				"properties":           beats,
				"additionalProperties": false,
			},
			"off_script": map[string]any{
				"type":        "object",
				"description": "Present only when the player's action wasn't covered by the script.",
				"properties": map[string]any{
					"kind":   map[string]any{"type": "string", "enum": []string{OffScriptImprovised, OffScriptSteered, OffScriptRefused}},
					"reason": map[string]any{"type": "string"},
				},
				"required": []string{"kind"},
			},
		},
		"required": []string{"narration", "world_events", "beats"},
	}
}

func withMeanings(desc string, s campaigns.AllowedConditionSpec) string {
	if len(s.Meanings) == 0 {
		return desc
	}
	out := desc
	for _, v := range s.AllowedValues {
		if m, ok := s.Meanings[v]; ok {
			out += " " + v + ": " + m
		}
	}
	if m, ok := s.Meanings["true"]; ok && s.Type == "flag_true" {
		out += " true: " + m
	}
	return out
}
