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

// returns the first Die found that is within the cursor's bounds
//
// used to later set the die's mode to DRAG
//
// inputs:
//
//	dice []*Die // usually g.Dice
//	x, y int    // cursor should be from ebiten.CursorPosition()
func (g *Game) PickDie() *Die {
	if len(g.Dice) == 0 {
		return nil
	}

	x := g.cx
	y := g.cy

	var index int // to put on last element of g.Dice to have it render on top
	var PickedDie *Die
	//tempDie := g.Dice[index]

	// the last one rendered is on top
	for i := len(g.Dice) - 1; i >= 0; i -= 1 {
		die := g.Dice[i]
		withinX := x > die.Vec2.X && x < die.Vec2.X+TileSize
		withinY := y > die.Vec2.Y && y < die.Vec2.Y+TileSize

		if withinX && withinY {
			render.XOffset = x - die.Vec2.X
			render.YOffset = y - die.Vec2.Y
			index = i
			PickedDie = die
			break
		}
	}

	//clicked nothing
	if PickedDie == nil {
		return nil
	}

	// shift left
	for i := index; i < len(g.Dice)-1; i++ {
		g.Dice[i] = g.Dice[i+1]
	}

	// set top to picked die
	g.Dice[len(g.Dice)-1] = PickedDie

	return PickedDie
}

func (g *Game) ControlAction(action Action) {
	if action == ROLLING {
		return
	}

	// cant make an action if scoring
	if g.ActiveLevel.scoringState != SCORING_IDLE {
		return
	}

	switch action {
	case ROLL:
		if g.ActiveLevel.RollsLeft > 0 {
			g.ActiveLevel.RollsLeft--
			for _, die := range g.Dice {
				die.Roll()
			}
		} else {
			for _, die := range g.Dice {
				if die.Mode == ROLLING {
					// specific impl if roll was pressed and no more rolls
					die.ZRotation = rand.Float32() + -rand.Float32() // rotate changes
				} else {
					die.Roll()
				}
			}
		}
	case PRESS:
		g.Press()
	case SELECT:
		g.Select()
	case SCORE:
		if g.ActiveLevel.HandsLeft > 0 {
			g.ActiveLevel.HandsLeft--
			g.ActiveLevel.RollsLeft = g.ActiveLevel.MaxRolls
			g.SetToScore()
		}
	}
}

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
			die.Roll()
		}
	}
}

// when a die gets clicked on for the first time
//
// turn's that Die's mode to DRAG
func (g *Game) Press() {
	die := g.PickDie()
	if die != nil {

		// g.Time = time.Now()

		// where the mouse was clicked
		// die.Fixed = render.Vec2{
		// 	X: g.x,
		// 	Y: g.y,
		// } // set the fixed position to the current position
		die.Mode = DRAG
		die.Height = 0 //reset height if needed
		// die.Modifier = .25 // for speeding up if needed
	}
	// if g.cursorWithin(render.ROLLZONE) {
	// 	// render.Zones
	// }
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

	if g.cursorWithin(render.SCOREZONE) {
		die.Mode = HELD
		return
	}
	// check if die was flicked

	// since := time.Since(g.Time)
	// if since < time.Second {
	// 	flickBuffer := d.TileSize * 3

	// 	above := g.y < d.Fixed.Y+d.TileSize/2
	// 	below := g.y > d.Fixed.Y-flickBuffer*2
	// 	left := g.x > d.Fixed.X-flickBuffer
	// 	right := g.x < d.Fixed.X+flickBuffer

	// 	flicked := above && below && left && right

	// 	if flicked {
	// 		d.Mode = HELD
	// 		return
	// 	}
	// }

	// let go of die
	die.Mode = ROLLING

	// clamp workaround, needed if no more rolls
	if !g.cursorWithin(render.ROLLZONE) {
		render.ClampInZone(&die.DieRenderable, render.ROLLZONE)
	}
	// if g.cursorWithin(render.SmallRollZone) {
	// d.HoverFromFromFixed()
	// d.Fixed = render.Vec2{}
	// 	return
	// }
}
