package main

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

// TODO: better abstraction than this
func DEBUGTitleFPS(x, y float64) {
	msg := fmt.Sprintf("T%0.2f F%0.2f x%4.0f y%4.0f ", ebiten.ActualTPS(), ebiten.ActualFPS(), x, y)
	ebiten.SetWindowTitle("Dice Will Roll " + msg)
}

// interface impl
func (g *Game) Update() error {
	g.UpdateCusor()
	g.time = float32(time.Since(g.startTime).Milliseconds()) / float32(ebiten.TPS())

	DEBUGTitleFPS(g.cx, g.cy)

	// find/assign handrank
	var held []*Die
	var hold []dice.Die
	for i := 0; i < len(g.Dice); i++ {
		d := g.Dice[i]
		if d.Mode == HELD {
			d.Height = -.5
			hold = append(hold, d.Die)
			held = append(held, d)
		}
	}
	g.ActiveLevel.Hand = dice.DetermineHandRank(hold)
	g.ActiveLevel.ScoringHand = FindHandRankDice(held, g.ActiveLevel.Hand)
	for _, die := range g.ActiveLevel.ScoringHand {
		die.Height = .1
	}

	action := g.Controls()
	g.ControlAction(action)

	g.UpdateDice()

	return nil
}

func (g *Game) UpdateDice() {
	var (
		rolling     []*render.DieRenderable
		held        []*render.DieRenderable
		moving      []*render.DieRenderable
		scoringDice []*Die
	)

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]
		d := &die.DieRenderable

		// when logic for a d.Mode gets too complex put it in render/
		if die.Mode == ROLLING {
			d.Velocity.X *= render.BounceFactor
			d.Velocity.Y *= render.BounceFactor
			d.Vec2.X += d.Velocity.X
			d.Vec2.Y += d.Velocity.Y

			rolling = append(rolling, d)
		} else if die.Mode == DRAG {
			d.Vec2.X = g.cx - render.XOffset
			d.Vec2.Y = g.cy - render.YOffset
			d.Velocity.X *= render.BounceFactor
			d.Velocity.Y *= render.BounceFactor

			moving = append(moving, d)
		} else if die.Mode == HELD {
			held = append(held, d)
		} else if die.Mode == SCORING {
			scoringDice = append(scoringDice, die)
		}
	}
	moving = append(moving, rolling...)

	render.HandleMovingHeldDice(held)
	render.HandleDiceCollisions(moving)
	render.BounceAndClamp(rolling)

	g.ActiveLevel.HandleScoring(scoringDice)
}

// always is called at the beginning of the update loop
func (g *Game) UpdateCusor() {
	x, y := ebiten.CursorPosition()
	g.cx = float64(x)
	g.cy = float64(y)
}

func (g *Game) cursorWithin(zone render.ZoneRenderable) bool {
	return g.cx > zone.MinWidth && g.cx < zone.MaxWidth && g.cy > zone.MinHeight && g.cy < zone.MaxHeight
}
