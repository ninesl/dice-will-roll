package main

import "github.com/ninesl/dice-will-roll/render"

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
	// the last one rendered is on top
	for i := len(g.Dice) - 1; i >= 0; i -= 1 {
		die := g.Dice[i]
		withinX := x > die.Vec2.X && x < die.Vec2.X+die.TileSize
		withinY := y > die.Vec2.Y && y < die.Vec2.Y+die.TileSize

		if withinX && withinY {
			render.XOffset = x - die.Vec2.X
			render.YOffset = y - die.Vec2.Y
			return die
		}
	}

	return nil
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
	case PICK_DIE:
		die := g.PickDie()
		if die != nil {
			die.Mode = DRAG
		}
	case SELECT:
		for _, die := range g.Dice {
			if die.Mode == DRAG {
				die.Mode = ROLLING

			}
		}
	}
}
