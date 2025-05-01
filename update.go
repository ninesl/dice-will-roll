package main

import (
	"github.com/ninesl/dice-will-roll/render"
)

/*

dice has a mode, update Dice is based on the mode

*/

// does keeping this in the struct improve performance/cycles?
func (g *Game) UpdateDice() {

	// Unsure if this is a good idea,probably wasting CPU cycles
	// maybe a pointer to this during loading just to access it?
	// I'm not a fan of that abstraction it'd be hard to keep track of
	var rolling []*render.DieRenderable
	var held []*render.DieRenderable

	for i := 0; i < len(g.Dice); i++ {
		d := g.Dice[i]
		die := &d.DieRenderable

		if d.Mode == ROLLING {
			d.Velocity.X *= .95
			d.Velocity.Y *= .95

			d.Vec2.X += d.Velocity.X
			d.Vec2.Y += d.Velocity.Y
			render.BounceAndClamp(die)

			rolling = append(rolling, die)
		} else if d.Mode == DRAG {
			d.Velocity.X = 0
			d.Velocity.Y = 0

			d.Vec2.X = g.x - render.XOffset // + (die.Vec2.X + die.TileSize - g.x)
			d.Vec2.Y = g.y - render.YOffset // + (die.Vec2.Y + die.TileSize - g.y)
		} else if d.Mode == HELD {
			held = append(held, die)
		}
	}

	render.HandleDiceCollisions(rolling)
	render.HandleHeldDice(held)
}
