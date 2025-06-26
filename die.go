package main

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
)

type Die struct {
	image *ebiten.Image
	// for when height/scale needs to change. is double the canvas of .image
	// bigImage *ebiten.Image
	render.DieRenderable
	dice.Die
	// sprite *render.Sprite
	Mode Action // Current mode of the die, is modified thru player Controls()
	// ImageIndex int    // current one to use. default is 0
}

func SetupNewDie(dieImgSize int, color render.Vec3) *Die {
	directionX := float64(rand.IntN(2)) + 1
	directionY := float64(rand.IntN(2)) + 1
	if directionX == 2 {
		directionX = -1.0
	}
	if directionY == 2 {
		directionY = -1.0
	}

	tileSize := float64(dieImgSize)

	pos := render.Vec2{
		X: render.ROLLZONE.MinWidth + tileSize*float64(rand.IntN(6))*2.0,
		Y: render.ROLLZONE.MaxHeight/2 - tileSize*0.5,
	}
	dieRenderable := render.DieRenderable{
		Fixed: pos,
		Vec2:  pos,
		Velocity: render.Vec2{
			X: (rand.Float64()*40 + 20),
			Y: (rand.Float64()*40 + 20),
		},
		ZRotation: rand.Float32(),
		TileSize:  float64(dieImgSize),
		Color:     color,
		// ColorSpot: 1 * 6,
	}
	image := ebiten.NewImage(dieImgSize, dieImgSize)
	// bigImage := ebiten.NewImage(dieImgSize*2, dieImgSize*2)

	// // draws shader to image, the uniforms
	// var vertices []ebiten.Vertex
	// var indicies []uint16
	// var opts *ebiten.DrawTrianglesShaderOptions
	// image.DrawTrianglesShader(vertices, indicies, shader, opts)

	values := [6]int{}

	for i := range len(values) {
		// values[i] = rand.IntN(8) + 1
		values[i] = 9
	}

	die := &Die{
		Die: dice.NewDie(6),
		// Die:           dice.New6SidedDie(values),
		image: image,
		// bigImage:      bigImage,
		DieRenderable: dieRenderable,
		Mode:          ROLLING,
	}
	die.Roll()

	return die
}

var NUM_PLAYER_DICE = 7

func SetupPlayerDice(dieImgSize int) []*Die {
	var dice []*Die

	var colors = []render.Vec3{
		render.Color(150, 0, 0),    // red
		render.Color(175, 127, 25), // orange
		render.Color(160, 160, 0),  // yellow
		render.Color(0, 150, 50),   // green
		render.Color(50, 50, 200),  // blue
		render.Color(75, 0, 130),   // indigo
		render.Color(125, 50, 183), // purple
	}

	NUM_PLAYER_DICE = len(colors)

	// for i := range colors {
	// 	fmt.Printf("%#v\n", colors[i])
	// }

	for i := range NUM_PLAYER_DICE { // range NUM_PLAYER_DICE {
		dice = append(dice, SetupNewDie(dieImgSize, colors[i]))
	}

	return dice
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
		d.Die.Roll()
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
