package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"math/rand/v2"

	_ "embed"
	_ "image/png" // for png encoder

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/render"
)

type Game struct {
	// DiceSprite *render.Sprite
	Dice     []*Die
	TileSize int
	//   can be updated with LocateCursor()
	x, y float64 // the x/y coordinates of the cursor
}

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
	diceImg := ebiten.NewImageFromImage(DiceImage)
	dieImgSize := diceImg.Bounds().Dx() / 6

	SetBounds(dieImgSize)
	render.SetZones()

	diceSheet := &render.Sprite{
		Image:       diceImg,
		SpriteSheet: render.NewSpriteSheet(6, 7, dieImgSize),
	}

	dice := SetupPlayerDice(diceSheet, dieImgSize)

	g := &Game{
		TileSize: dieImgSize,
		// DiceSprite: diceSheet,
		Dice: dice,
	}

	return g
}

func (g *Game) String() string {
	return fmt.Sprintf("GAMEBOUNDS X %d\nGAMEBOUNDS Y %ds\nROLLZONE %#v\n", GAME_BOUNDS_X, GAME_BOUNDS_Y, render.ROLLZONE)
}

func SetupPlayerDice(diceSheet *render.Sprite, dieImgSize int) []*Die {
	var dice []*Die
	for i := range 7 {
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
			X: render.ROLLZONE.MinWidth + tileSize*float64(i)*2.0,
			Y: render.ROLLZONE.MaxHeight/2 - tileSize*0.5,
		}
		dieRenderable := render.DieRenderable{
			Fixed: pos,
			Vec2:  pos,
			Velocity: render.Vec2{
				X: (rand.Float64()*40 + 20) * float64(i) * -1.0,
				Y: (rand.Float64()*40 + 20) * float64(i) * -1.0,
			},
			TileSize:  float64(dieImgSize),
			ColorSpot: i * 6,
		}

		die := Die{
			sprite:        diceSheet,
			DieRenderable: dieRenderable,
			Mode:          ROLLING,
		}
		dice = append(dice, &die)
	}

	return dice
}
