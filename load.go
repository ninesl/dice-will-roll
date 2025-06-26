package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"time"

	_ "embed"
	"image/color"
	_ "image/png" // for png encoder

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
	// diceImg := ebiten.NewImage(tileSize, tileSize)

	dieImgSize := tileSize * 2
	SetBounds(dieImgSize)

	render.SetZones()

	// diceSheet := &render.Sprite{
	// 	Image: diceImg,
	// 	// SpriteSheet: render.NewSpriteSheet(6, 7, dieImgSize),
	// }

	dice := SetupPlayerDice(dieImgSize)

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
