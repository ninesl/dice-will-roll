package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ninesl/dice-will-roll/render"
)

// returns an Action based on player input
//
// input for the controller scheme? TODO:FIXME: idk if this is final
func (g *Game) Controls() Action {
	g.UpdateCusor()

	var action Action = ROLLING // the animation of rolling
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		action = ROLL
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		if g.cursorWithin(render.ROLLZONE) {
			action = PRESS
		}
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0) {
		action = SELECT
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
			g.Time = time.Now()
			g.Fixed = die.Vec2
			die.Fixed = die.Vec2 // set the fixed position to the current position
			die.Mode = DRAG
		}
		if g.cursorWithin(render.ROLLZONE) {
			// render.Zones
		}
	case SELECT:
		var d *Die
		for _, die := range g.Dice {
			if die.Mode == DRAG {
				d = die
				break
			}
		}

		if d == nil {
			return
		}

		//TODO: make this flick threshold 'feel' good
		flicked := (time.Since(g.Time) > time.Millisecond*200) && (d.Fixed.Y+d.TileSize > d.Vec2.Y)

		if g.cursorWithin(render.SCOREZONE) || flicked {
			d.Mode = HELD
			return
		}

		// let go of die
		d.Mode = ROLLING

		if g.cursorWithin(render.SmallRollZone) {
			d.HoverFromFromFixed()
			d.Fixed = render.Vec2{}
			return
		}
	}
}
