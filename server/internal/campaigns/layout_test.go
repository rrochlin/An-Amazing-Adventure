package campaigns_test

import (
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/game"
)

// Playtest 3: every Briarwell room sat at (0, 0), so the map drew one node.
func TestBootstrap_LaysRoomsOutFromTheirExits(t *testing.T) {
	_, g, _ := bootstrapBriarwell(t)
	want := map[string]game.Coordinates{
		"family_cottage": {X: 0, Y: 0},
		"village_square": {X: 100, Y: 0},
		"market_lane":    {X: 100, Y: -100},
		"old_mill":       {X: 200, Y: 0},
		"meadow":         {X: 100, Y: 100},
		"woods_edge":     {X: 100, Y: 200},
	}
	seen := map[game.Coordinates]string{}
	for id, w := range want {
		got := g.Rooms[id].Coordinates
		if got.X != w.X || got.Y != w.Y {
			t.Errorf("%s at (%v, %v), want (%v, %v)", id, got.X, got.Y, w.X, w.Y)
		}
		if other, dup := seen[got]; dup {
			t.Errorf("%s and %s share a map position", id, other)
		}
		seen[got] = id
	}
}
