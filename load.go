package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"math/rand/v2"
	"time"

	_ "embed"
	"image/color"
	_ "image/png" // for png encoder

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// return the pixels in the game
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return GAME_BOUNDS_X, GAME_BOUNDS_Y
}

// TODO: embedded FS for loading assets
//
//go:embed assets/images/dice.png
var dicePng []byte

func LoadImage() image.Image {
	img, _, err := image.Decode(bytes.NewReader(dicePng))
	if err != nil {
		log.Fatalf("failed to decode embedded image: %v", err)
	}
	return img
}

var (
	DiceImage  image.Image = LoadImage()
	DEBUG_FONT *text.GoTextFaceSource
)

func SetFonts() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.ArcadeN_ttf))
	if err != nil {
		log.Fatal(err)
	}
	DEBUG_FONT = s
}

func SetBounds(tileSize int) {
	GAME_BOUNDS_X = tileSize * 16
	GAME_BOUNDS_Y = tileSize * 9
	render.GAME_BOUNDS_X = float64(GAME_BOUNDS_X)
	render.GAME_BOUNDS_Y = float64(GAME_BOUNDS_Y)
}

func LoadGame() *Game {
	SetFonts()
	const tileSize = TILE_SIZE
	diceImg := ebiten.NewImage(tileSize, tileSize)

	dieImgSize := tileSize
	SetBounds(dieImgSize)

	render.SetZones()

	diceSheet := &render.Sprite{
		Image: diceImg,
		// SpriteSheet: render.NewSpriteSheet(6, 7, dieImgSize),
	}

	dice := SetupPlayerDice(diceSheet, dieImgSize)

	g := &Game{
		TileSize: float64(dieImgSize),
		// DiceSprite: diceSheet,
		Dice:      dice,
		Shaders:   shaders.LoadShaders(),
		startTime: time.Now(),
	}

	g.DEBUG.dieImgTransparent = render.CreateImage(dieImgSize, dieImgSize, color.RGBA{56, 56, 56, 100})

	// g.DieShader = s

	return g
}

func (g *Game) String() string {
	return fmt.Sprintf("GAMEBOUNDS X %d\nGAMEBOUNDS Y %ds\nROLLZONE %#v\n", GAME_BOUNDS_X, GAME_BOUNDS_Y, render.ROLLZONE)
}

func SetupNewDie(dieImgSize int, color render.Vec3) *Die {
	directionX := float64(rand.IntN(2))
	directionY := float64(rand.IntN(2))
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

	// // draws shader to image, the uniforms
	// var vertices []ebiten.Vertex
	// var indicies []uint16
	// var opts *ebiten.DrawTrianglesShaderOptions
	// image.DrawTrianglesShader(vertices, indicies, shader, opts)

	return &Die{
		Die:           dice.NewDie(6),
		image:         image,
		DieRenderable: dieRenderable,
		Mode:          ROLLING,
	}
}

var NUM_PLAYER_DICE = 7

func SetupPlayerDice(diceSheet *render.Sprite, dieImgSize int) []*Die {
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
