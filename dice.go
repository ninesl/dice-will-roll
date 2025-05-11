package main

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

type Die struct {
	*dice.Die
	render.DieRenderable
	sprite *render.Sprite
	Mode   Action // Current mode of the die, is modified thru player Controls()
}

func (d *Die) Sprite() *ebiten.Image {
	// g.DiceSprite.Image.SubImage(
	// 			g.DiceSprite.SpriteShe// held in handet.Rect(die.IndexOnSheet),
	// 		).(*ebiten.Image)

	return d.sprite.Image.SubImage(
		d.sprite.SpriteSheet.Rect(d.IndexOnSheet),
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

		// d.Theta += rand.Float64() * direction.X

		d.Velocity.X = d.TileSize * rand.Float64() * direction.X
		d.Velocity.Y = d.TileSize * rand.Float64() * direction.Y
	}
}

func (d *Die) RollDirection(rollingLeft bool) {

}
