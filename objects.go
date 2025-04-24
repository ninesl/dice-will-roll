package main

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

type Die struct {
	*render.Sprite
	*dice.Die
	render.DieRenderable
	Mode Mode // Current mode of the die, is modified thru player Controls()
}

func (d *Die) Draw() *ebiten.Image {
	// g.DiceSprite.Image.SubImage(
	// 			g.DiceSprite.SpriteSheet.Rect(die.IndexOnSheet),
	// 		).(*ebiten.Image)

	return d.Image.SubImage(
		d.SpriteSheet.Rect(d.IndexOnSheet),
	).(*ebiten.Image)
}

// When spacebar/roll is pressed
//
// moves die around on the screen if applicable
//
// changes the face of the die if applicable
//
// logic based on Mode
func (d *Die) Roll() {
	switch d.Mode {
	case ROLLING:
		dir := render.Direction(rand.IntN(2) + render.UPLEFT) // random direction
		direction := render.DirectionMap[dir]

		d.Velocity.X = d.TileSize * direction.X
		d.Velocity.Y = d.TileSize * direction.Y

	}
}
