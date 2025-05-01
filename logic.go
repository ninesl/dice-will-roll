package main

import (
	"github.com/ninesl/dice-will-roll/render"
)

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
	x := g.x
	y := g.y

	var index int // to put on last element of g.Dice to have it render on top
	var PickedDie *Die
	//tempDie := g.Dice[index]

	// the last one rendered is on top
	for i := len(g.Dice) - 1; i >= 0; i -= 1 {
		die := g.Dice[i]
		withinX := x > die.Vec2.X && x < die.Vec2.X+die.TileSize
		withinY := y > die.Vec2.Y && y < die.Vec2.Y+die.TileSize

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

	switch action {
	case ROLL:
		for _, die := range g.Dice {
			die.Roll()
		}
	case PRESS:
		die := g.PickDie()
		if die != nil {
			die.Fixed = die.Vec2 // set the fixed position to the current position
			die.Mode = DRAG
		}
	case SELECT:
		var d *Die
		for _, die := range g.Dice {
			if die.Mode == DRAG {
				d = die
				break
			}
		}

		if g.cursorWithin(render.SCOREZONE) {
			d.Mode = HELD
			return
		}

		// let go of die
		d.Mode = ROLLING

		if g.cursorWithin(render.ROLLZONE) {
			if d.Vec2.X > d.Fixed.X {
				d.Velocity.X = 1
			} else {
				d.Velocity.X = -1
			}
			if d.Vec2.Y > d.Fixed.Y {
				d.Velocity.Y = 1
			} else {
				d.Velocity.Y = -1
			}
			return
		}

		// if the die is not within the rollzone, set it to the fixed position
		d.Vec2.X = d.Fixed.X
	}
}
