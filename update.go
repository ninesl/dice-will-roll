package main

import "github.com/ninesl/dice-will-roll/render"

/*

dice has a mode, update Dice is based on the mode

*/

// does keeping this in the struct improve performance/cycles?
func UpdateDice(dice []*Die) {

	// Unsure if this is a good idea,probably wasting CPU cycles
	// maybe a pointer to this during loading just to access it?
	// I'm not a fan of that abstraction it'd be hard to keep track of
	var dieRenderables []*render.DieRenderable

	for i := range dice {
		die := &dice[i].DieRenderable
		dieRenderables = append(dieRenderables, die)

		render.UpdateDie(die)
		render.BounceAndClamp(die)
	}

	render.HandleDiceCollisions(dieRenderables)
	// for _, die := range dieRenderables {
	// }
	// // clamp and bounce

}
