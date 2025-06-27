package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

// TODO: better abstraction than this
func DEBUGTitleFPS(x, y float64, rolling, held int) {
	msg := fmt.Sprintf("T%0.2f F%0.2f x%4.0f y%4.0f ", ebiten.ActualTPS(), ebiten.ActualFPS(), x, y)
	msg += fmt.Sprintf("Rolling %d Held %d", rolling, held)
	ebiten.SetWindowTitle("Dice Will Roll " + msg)
}

// interface impl
func (g *Game) Update() error {
	DEBUGTitleFPS(g.x, g.y, g.DEBUG.rolling, g.DEBUG.held)

	action := g.Controls()

	g.ControlAction(action)
	g.UpdateDice()

	return nil
}

// does keeping this in the struct improve performance/cycles?
//
// Determine's rendering logic/move for dice
//
// Determine's held
func (g *Game) UpdateDice() {
	// Unsure if this is a good idea,probably wasting CPU cycles
	// maybe a pointer to this during loading just to access it?
	// But I'm not a fan of that abstraction it'd be hard to keep track of
	var rolling []*render.DieRenderable
	var held []*render.DieRenderable
	var moving []*render.DieRenderable
	var hand []dice.Die

	for i := 0; i < len(g.Dice); i++ {
		d := g.Dice[i]
		die := &d.DieRenderable

		if d.Mode == ROLLING {

			d.Velocity.X *= render.BounceFactor
			d.Velocity.Y *= render.BounceFactor

			d.Vec2.X += d.Velocity.X
			d.Vec2.Y += d.Velocity.Y

			rolling = append(rolling, die)
		} else if d.Mode == DRAG {
			d.Velocity.X = 0.0
			d.Velocity.Y = 0.0
			moveX := g.x - render.XOffset
			moveY := g.y - render.YOffset
			d.Vec2.X = moveX
			d.Vec2.Y = moveY
			moving = append(moving, die)
		} else if d.Mode == HELD {
			hand = append(hand, d.Die)
			held = append(held, die)
		}
	}

	render.HandleMovingHeldDice(held)

	g.Hand = dice.DetermineHandRank(hand)

	moving = append(moving, rolling...)
	render.HandleDiceCollisions(moving)
	render.BounceAndClamp(rolling)

}
