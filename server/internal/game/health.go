package game

import (
	"context"
	"fmt"

	"github.com/KirkDiggler/rpg-toolkit/events"
	dnd5echar "github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/character"
	"github.com/KirkDiggler/rpg-toolkit/rulebooks/dnd5e/combat"
)

// A player's health is their D&D character's hit points when they have one
// (every campaign session builds one, even from a quick start); otherwise the
// character's 0-100 health. One system per player, so what the client shows,
// what the story changes and when the party is wiped all agree.

// PlayerHitPoints returns a player's current and maximum health.
func (g *Game) PlayerHitPoints(userID string) (hp, max int) {
	if dnd, ok := g.GetDnDCharacter(userID); ok && dnd != nil {
		return dnd.GetHitPoints(), dnd.GetMaxHitPoints()
	}
	c, _ := g.GetPlayerCharacter(userID)
	return c.Health, 100
}

// DamagePlayer takes hit points from a player.
func (g *Game) DamagePlayer(ctx context.Context, userID string, amount int) (hp, max int, err error) {
	if dnd, ok := g.GetDnDCharacter(userID); ok && dnd != nil {
		dnd.ApplyDamage(ctx, &combat.ApplyDamageInput{Instances: []combat.DamageInstance{{Amount: amount}}})
		g.syncStub(userID, dnd)
		return dnd.GetHitPoints(), dnd.GetMaxHitPoints(), nil
	}
	c, ok := g.GetPlayerCharacter(userID)
	if !ok {
		return 0, 0, fmt.Errorf("player %s not found", userID)
	}
	if err := c.TakeDamage(amount); err != nil {
		return 0, 0, err
	}
	g.SetPlayerCharacter(userID, c)
	return c.Health, 100, nil
}

// HealPlayer gives a player hit points back, up to their maximum.
func (g *Game) HealPlayer(ctx context.Context, userID string, amount int) (hp, max int, err error) {
	if dnd, ok := g.GetDnDCharacter(userID); ok && dnd != nil {
		// The character only heals through its event bus; a round trip
		// through its data does the same without one.
		data := dnd.ToData()
		data.HitPoints = min(data.HitPoints+amount, data.MaxHitPoints)
		healed, err := dnd5echar.LoadFromData(ctx, data, events.NewEventBus())
		if err != nil {
			return 0, 0, fmt.Errorf("heal: %w", err)
		}
		g.SetDnDCharacter(userID, healed)
		g.syncStub(userID, healed)
		return healed.GetHitPoints(), healed.GetMaxHitPoints(), nil
	}
	c, ok := g.GetPlayerCharacter(userID)
	if !ok {
		return 0, 0, fmt.Errorf("player %s not found", userID)
	}
	if err := c.Heal(amount); err != nil {
		return 0, 0, err
	}
	g.SetPlayerCharacter(userID, c)
	return c.Health, 100, nil
}

// syncStub keeps the plain character's alive flag with the D&D character's
// hit points, for code that only looks at the stub.
func (g *Game) syncStub(userID string, dnd *dnd5echar.Character) {
	c, ok := g.GetPlayerCharacter(userID)
	if !ok {
		return
	}
	c.Alive = dnd.GetHitPoints() > 0
	g.SetPlayerCharacter(userID, c)
}
