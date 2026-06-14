package main

import (
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ninesl/dice-will-roll/render"
)

// returns an Action based on player input
//
// input for the controller scheme? TODO:FIXME: idk if this is final
func (g *Game) Controls() Action {

	// Rolling/mining a cave actions
	var action Action = ROLLING // the animation of rolling
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		action = ROLL
		g.startTime = time.Now()
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		// if g.cursorWithin(render.ROLLZONE) {
		action = PRESS
		// }
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0) {
		action = SELECT
	} else if inpututil.IsKeyJustReleased(ebiten.KeyQ) {
		action = SCORE
	} else if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		for _, die := range g.Dice {
			die.Mode = HELD
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		for _, die := range g.Dice {
			die.Mode = ROLLING
			die.Roll()
		}
	}

	return action
}

// // returns the first Die found that is within the cursor's bounds
// //
// // used to later set the die's mode to DRAG
// //
// // inputs:
// //
// //	dice []*Die // usually g.Dice
// //	x, y int    // cursor should be from ebiten.CursorPosition()
//
//	func (g *Game) PickDie() *Die {
//		if len(g.Dice) == 0 {
//			return nil
//		}
//
//		var index int // to put on last element of g.Dice to have it render on top
//		var PickedDie *Die
//		//tempDie := g.Dice[index]
//
//		// the last one rendered is on top
//		for i := len(g.Dice) - 1; i >= 0; i -= 1 {
//			die := g.Dice[i]
//			withinX := g.cursorPos.X > die.Vec2.X && g.cursorPos.X < die.Vec2.X+TileSize
//			withinY := g.cursorPos.Y > die.Vec2.Y && g.cursorPos.Y < die.Vec2.Y+TileSize
//
//			if withinX && withinY {
//				render.XOffset = g.cursorPos.X - die.Vec2.X
//				render.YOffset = g.cursorPos.Y - die.Vec2.Y
//				index = i
//				PickedDie = die
//				break
//			}
//		}
//
//		//clicked nothing
//		if PickedDie == nil {
//			return nil
//		}
//
//		// shift left
//		for i := index; i < len(g.Dice)-1; i++ {
//			g.Dice[i] = g.Dice[i+1]
//		}
//
//		// set top to picked die
//		g.Dice[len(g.Dice)-1] = PickedDie
//
//		return PickedDie
//	}
//
// assigns hand within ActiveLevel to SCORING
func (g *Game) SetToScore() {
	g.ActiveLevel.ScoreHand = g.ActiveLevel.Hand

	for i := 0; i < len(g.ActiveLevel.ScoringHand); i++ {
		d := g.ActiveLevel.ScoringHand[i]
		d.Mode = SCORING
		d.ZRotation = 0
	}

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]
		if die.Mode == HELD {
			die.Mode = ROLLING
			// Clear fixed position and height
			die.Fixed.X = 0
			die.Fixed.Y = 0
			die.Height = 0
			// Set velocity straight down to bounce into rollzone
			die.Velocity.Y = render.DieTileSize * 2 // downward velocity

			if die.Vec2.X < render.GAME_BOUNDS_X/2 {
				die.Velocity.X = render.DieTileSize * 2 // push right
			} else {
				die.Velocity.X = render.DieTileSize * -2 // push left
			}

			die.Direction = render.DirectionArr[render.DOWN]
			die.ZRotation = rand.Float32()
			// Roll the die face value
			die.Die.Roll()
			// deselect rocks, no color for rocks that aren't in scoring hand
			g.RocksRenderer.DeselectRocks(die.Identifier)
		}
	}
}

// when a die gets clicked on for the first time
//
// turn's that Die's mode to DRAG
func (g *Game) Press(die *Die) {
	if die != nil {
		g.holdTime = time.Now()
		g.holdCx = g.cursorPos.X
		g.holdCy = g.cursorPos.Y

		// g.Time = time.Now()

		// where the mouse was clicked
		// die.Fixed = render.Vec2{
		// 	X: g.x,
		// 	Y: g.y,
		// } // set the fixed position to the current position
		die.Mode = DRAG
		die.Fixed = g.cursorPos
		die.Height = 0 //reset height if needed
		// die.Modifier = .25 // for speeding up if needed
	}
}

// lets go of the die. contextually knows what to do with it
func (g *Game) Select() {
	var die *Die
	for _, d := range g.Dice {
		if d.Mode == DRAG {
			die = d
			break
		}
	}

	if die == nil {
		return
	}

	// if clicked or within ScoreZone
	if (g.cursorWithin(render.SmallRollZone) && time.Since(g.holdTime) < ClickTime) || g.cursorWithin(render.SCOREZONE) {

		if render.SCOREZONE.ContainsPoint(g.holdCx, g.holdCy) && time.Since(g.holdTime) < ClickTime {
			// Calculate center of ROLLZONE and animate die to that position
			centerX := (render.ROLLZONE.MinWidth + render.ROLLZONE.MaxWidth) / 2
			centerY := (render.ROLLZONE.MinHeight + render.ROLLZONE.MaxHeight) / 2
			render.AnimateDieToPosition(&die.DieRenderable, centerX, centerY)

			g.ResetHoldPoint()
			die.Mode = ROLLING

			g.RocksRenderer.DeselectRocks(die.Identifier)

			//TODO:FIXME: when a rock is click and dragged it keeps rolling. it isn't in held dice anymore but it's still spinning
			// likely needs a check when held or stm
			return
		}

		g.ResetHoldPoint()
		die.Mode = HELD

		g.RocksRenderer.SelectRocksColor(die.Color, die.Identifier, len(g.Dice), die.ActiveFace().NumPips())

		return
	}

	if die.Mode == DRAG {
		die.Fixed.X = 0
		die.Fixed.Y = 0
		g.RocksRenderer.DeselectRocks(die.Identifier)
	}

	// let go of die
	die.Mode = ROLLING

	// clamp workaround, needed if no more rolls
	if !g.cursorWithin(render.ROLLZONE) {
		render.ClampInZone(&die.DieRenderable, render.ROLLZONE)
	}
}
