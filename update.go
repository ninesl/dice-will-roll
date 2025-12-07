package main

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

func DEBUGTitleFPS(x, y float32) {
	msg := fmt.Sprintf("T%0.2f F%0.2f x%4.0f y%4.0f ", ebiten.ActualTPS(), ebiten.ActualFPS(), x, y)
	ebiten.SetWindowTitle("Dice Will Roll " + msg)
}

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

	g.ControlAction(g.Controls())

	g.UpdateDice()
	g.UpdateRocks()

	return nil
}

func (g *Game) UpdateDice() {
	var (
		rolling     []*render.DieRenderable
		held        []*render.DieRenderable
		moving      []*render.DieRenderable
		resetting   []*render.DieRenderable
		scoringDice []*Die
	)

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]
		d := &die.DieRenderable

		// when logic for a d.Mode gets too complex put it in render/
		if die.Mode == ROLLING {
			// Check if this die has a Fixed position set (meaning it's resetting)
			if d.Fixed.X != 0 || d.Fixed.Y != 0 {
				resetting = append(resetting, d)
			} else {
				// Normal rolling behavior
				d.Velocity.X *= render.BounceFactor
				d.Velocity.Y *= render.BounceFactor
				d.Vec2.X += d.Velocity.X
				d.Vec2.Y += d.Velocity.Y
			}

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

	render.HandleResettingDice(resetting)
	render.HandleMovingHeldDice(held)
	render.HandleDiceCollisions(moving)
	render.BounceAndClamp(rolling)

	g.ActiveLevel.HandleScoring(scoringDice, g.RocksRenderer)

	// Populate dice data buffer after all dice physics are resolved
	// X=centerX, Y=centerY, Z=speed (velocity magnitude / BaseVelocity for slope compatibility)
	g.diceDataBuffer = g.diceDataBuffer[:0]
	for _, d := range g.Dice {
		// Compute speed as average of absolute velocities, scaled to slope range
		speed := (float32(math.Abs(float64(d.Velocity.X))) + float32(math.Abs(float64(d.Velocity.Y)))) / (2 * render.BaseVelocity)
		g.diceDataBuffer = append(g.diceDataBuffer, render.Vec3{
			X: d.Vec2.X + render.HalfDieTileSize,
			Y: d.Vec2.Y + render.HalfDieTileSize,
			Z: speed,
		})
	}
}

func (g *Game) UpdateRocks() {
	// Pass pre-computed dice collision data to renderer
	g.RocksRenderer.UpdateRocksAndCollide(g.cx, g.cy, g.diceDataBuffer)
}

// always is called at the beginning of the update loop
func (g *Game) UpdateCusor() {
	x, y := ebiten.CursorPosition()
	g.cx = float32(x)
	g.cy = float32(y)
}

func (g *Game) cursorWithin(zone render.ZoneRenderable) bool {
	return g.cx > zone.MinWidth && g.cx < zone.MaxWidth && g.cy > zone.MinHeight && g.cy < zone.MaxHeight
}
