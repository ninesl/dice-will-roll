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
	var dieRenderables []*render.DieRenderable

	for i := range g.Dice {
		die := &g.Dice[i].DieRenderable
		dieRenderables = append(dieRenderables, die)

		render.UpdateDie(die)
	}

	render.HandleDiceCollisions(dieRenderables)

	for _, die := range g.Dice {
		if die.Mode == ROLLING {
			render.BounceAndClamp(&die.DieRenderable)
		} else if die.Mode == DRAG {
			die.Velocity.X = 0
			die.Velocity.Y = 0

			die.Vec2.X = g.x - render.XOffset // + (die.Vec2.X + die.TileSize - g.x)
			die.Vec2.Y = g.y - render.YOffset // + (die.Vec2.Y + die.TileSize - g.y)
		}
	}

}
