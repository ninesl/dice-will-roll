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
	g.UpdateCusor()

	DEBUGTitleFPS(g.x, g.y, g.DEBUG.rolling, g.DEBUG.held)

	action := g.Controls()

	g.ControlAction(action)
	g.UpdateDice()

	return nil
}

func (g *Game) UpdateDice() {
	var (
		rolling []*render.DieRenderable
		held    []*render.DieRenderable
		moving  []*render.DieRenderable
		hand    []dice.Die
	)

	for i := 0; i < len(g.Dice); i++ {
		d := g.Dice[i]
		die := &d.DieRenderable

		// when logic for a d.Mode gets too complex put it in render/
		if d.Mode == ROLLING {
			d.Velocity.X *= render.BounceFactor
			d.Velocity.Y *= render.BounceFactor
			d.Vec2.X += d.Velocity.X
			d.Vec2.Y += d.Velocity.Y

			rolling = append(rolling, die)
		} else if d.Mode == DRAG {
			d.Velocity.X = 0.0
			d.Velocity.Y = 0.0
			d.Vec2.X = g.x - render.XOffset
			d.Vec2.Y = g.y - render.YOffset
			moving = append(moving, die)
		} else if d.Mode == HELD {
			hand = append(hand, d.Die)
			held = append(held, die)
		}
	}
	g.Hand = dice.DetermineHandRank(hand) // better to collect here than a loop somewhere else

	render.HandleMovingHeldDice(held)

	moving = append(moving, rolling...)
	render.HandleDiceCollisions(moving)
	render.BounceAndClamp(rolling)

}

// always is called at the beginning of the update loop
func (g *Game) UpdateCusor() {
	x, y := ebiten.CursorPosition()
	g.x = float64(x)
	g.y = float64(y)
}

func (g *Game) cursorWithin(zone render.ZoneRenderable) bool {
	return g.x > zone.MinWidth && g.x < zone.MaxWidth && g.y > zone.MinHeight && g.y < zone.MaxHeight
}
