package campaigns

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Apply enforces the campaign's character creation rules on a player's
// request and returns the character data to actually build.
//
// A preset campaign ignores everything but the name: race, class, scores,
// and skills come from the preset. Otherwise the request's race, subrace, and
// class must be in the allowed lists (when set). A locked background always
// replaces any player-written backstory.
func (r CharacterCreationRules) Apply(in game.CharacterCreationData) (game.CharacterCreationData, error) {
	out := in
	if p := r.Preset; p != nil {
		out = game.CharacterCreationData{
			Name:           strings.TrimSpace(in.Name),
			RaceID:         p.RaceID,
			SubraceID:      p.SubraceID,
			ClassID:        p.ClassID,
			AbilityScores:  copyScores(p.AbilityScores),
			SelectedSkills: append([]string(nil), p.SelectedSkills...),
		}
		if out.Name == "" {
			out.Name = p.DefaultName
		}
	} else {
		if err := checkAllowed("race", in.RaceID, r.AllowedRaces); err != nil {
			return game.CharacterCreationData{}, err
		}
		if in.SubraceID != "" {
			if err := checkAllowed("subrace", in.SubraceID, r.AllowedSubraces); err != nil {
				return game.CharacterCreationData{}, err
			}
		}
		if err := checkAllowed("class", in.ClassID, r.AllowedClasses); err != nil {
			return game.CharacterCreationData{}, err
		}
	}
	if r.LockedBackgroundText != "" {
		out.Backstory = r.LockedBackgroundText
	}
	if r.ThemeHintMode == "disabled" {
		out.ThemeHint = ""
	}
	if r.PreferenceMode == "disabled" {
		out.Preferences = nil
	}
	return out, nil
}

// Manifest describes the rules to the client's create wizard.
func (r CharacterCreationRules) Manifest() CharacterCreationManifest {
	m := CharacterCreationManifest{
		Mode:             "standard",
		AllowedRaces:     r.AllowedRaces,
		AllowedSubraces:  r.AllowedSubraces,
		AllowedClasses:   r.AllowedClasses,
		LockedBackground: r.LockedBackgroundText,
	}
	if r.Preset != nil {
		m.Mode = "preset"
		m.Preset = r.Preset
	}
	return m
}

func (r CharacterCreationRules) validate() error {
	for _, c := range r.AllowedClasses {
		if !slices.Contains(game.SupportedClasses, c) {
			return fmt.Errorf("allowed class %q cannot be built (supported: %v)", c, game.SupportedClasses)
		}
	}
	p := r.Preset
	if p == nil {
		return nil
	}
	if p.RaceID == "" || p.ClassID == "" {
		return fmt.Errorf("preset requires race_id and class_id")
	}
	if !slices.Contains(game.SupportedClasses, p.ClassID) {
		return fmt.Errorf("preset class %q cannot be built (supported: %v)", p.ClassID, game.SupportedClasses)
	}
	if len(p.AbilityScores) != 6 {
		return fmt.Errorf("preset requires all six ability_scores")
	}
	if err := checkAllowed("preset race", p.RaceID, r.AllowedRaces); err != nil {
		return err
	}
	return checkAllowed("preset class", p.ClassID, r.AllowedClasses)
}

func checkAllowed(kind, value string, allowed []string) error {
	if len(allowed) == 0 || slices.Contains(allowed, value) {
		return nil
	}
	return fmt.Errorf("%s %q is not allowed in this campaign (allowed: %s)", kind, value, strings.Join(allowed, ", "))
}

func copyScores(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
