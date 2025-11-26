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

const BaseVelocity = 2.0

func (g *Game) UpdateRocks() {
	g.RocksRenderer.ActiveRockType++
	if g.RocksRenderer.ActiveRockType >= render.NUM_ROCK_TYPES {
		g.RocksRenderer.ActiveRockType = 0
	}
	for _, rock := range g.RocksRenderer.Rocks[g.RocksRenderer.ActiveRockType] {
		// Update transition system for smooth sprite rotation during direction changes
		// rock.UpdateTransition()

		// if rock.TransitionSpeedY >= render.MAX_SPEED {
		// 	rock.TransitionSpeedY = render.MIN_SPEED + 1
		// } else {
		// 	rock.TransitionSpeedY++
		// }

		rock.Update()

		// if rock.SpriteSlopeX >= render.DIRECTIONS_TO_SNAP {
		// 	rock.SpriteSlopeX =
		// } else if rock.SpriteSlopeX < 0 {
		// 	rock.SpriteSlopeY = 0
		// }

		// if rock.TransitionSpeedY > render.MAX_SPEED {
		// 	rock.TransitionSpeedY = render.MIN_SPEED + 1
		// } else {
		// 	rock.TransitionSpeedY++
		// }

		// rock.TransitionSpeedX = int8(i)
		// rock.TransitionSpeedY = int8(i)

		// if rock.TransitionSpeedX >= render.MAX_SPEED {
		// 	rock.TransitionSpeedX -= render.MAX_SPEED * 2
		// }
		// if rock.TransitionSpeedY >= render.MAX_SPEED {
		// 	rock.TransitionSpeedY -= render.MAX_SPEED * 2
		// }

		// fmt.Printf("%d : X,Y{%d/%d}\n", i, rock.SlopeX, rock.SlopeX)

		// Update sprite rotation frame (creates rolling animation)
		// rock.SpriteIndex++
		// if rock.SpriteIndex >= render.ROTATION_FRAMES {
		// 	rock.SpriteIndex = 0
		// }

		// Calculate velocity directly from signed speed values
		// speedX := BaseVelocity * float64(rock.SpeedX)
		// speedY := BaseVelocity * float64(rock.SpeedY)
		// // rock.Position.X += speedX
		// // rock.Position.Y += speedY

		// Handle X-axis boundary collisions
		if rock.Position.X+g.RocksRenderer.FSpriteSize >= render.GAME_BOUNDS_X {
			rock.Position.X = render.GAME_BOUNDS_X - g.RocksRenderer.FSpriteSize
			rock.BounceX() // Bounce off right wall
		} else if rock.Position.X <= 0 {
			rock.Position.X = 0
			rock.BounceX() // Bounce off left wall
		}

		// Handle Y-axis boundary collisions
		if rock.Position.Y+g.RocksRenderer.FSpriteSize >= render.GAME_BOUNDS_Y {
			rock.Position.Y = render.GAME_BOUNDS_Y - g.RocksRenderer.FSpriteSize
			rock.BounceY() // Bounce off bottom wall
		} else if rock.Position.Y <= 0 {
			rock.Position.Y = 0
			rock.BounceY() // Bounce off top wall
		}

		// g.RocksRenderer.Rocks[g.RocksRenderer.ActiveRockType][i] = rock
	}
}

// interface impl
func (g *Game) Update() error {
	g.UpdateCusor()
	g.time = float32(time.Since(g.startTime).Milliseconds()) / float32(ebiten.TPS())

	// Update rocks (frame-based, no deltaTime needed)
	// g.RocksRenderer.Update()

	// // Update camera (simple forward movement for now)
	// cameraPos := render.Vector3{
	// 	X: 0,
	// 	Y: 0,
	// 	Z: 0,
	// 	// Z: g.time * 10.0,
	// } // Move forward over time
	// cameraForward := render.Vector3{X: 0, Y: 0, Z: 1}
	// cameraUp := render.Vector3{X: 0, Y: 1, Z: 0}
	// g.RocksRenderer.UpdateCamera(cameraPos, cameraForward, cameraUp)

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
