package main

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

type Die struct {
	image *ebiten.Image
	render.DieRenderable
	dice.Die
	// sprite *render.Sprite
	Mode Action // Current mode of the die, is modified thru player Controls()
}

// func (d *Die) Rect()

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
		d.Direction = direction

		d.ZRotation = rand.Float32()
		// d.Height = 16.0
	}
}
