package main

import (
	"bytes"
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

func (g *Game) UpdateCusor() {
	x, y := ebiten.CursorPosition()
	g.x = float64(x)
	g.y = float64(y)
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

func LoadGame() *Game {
	SetFonts()

	// img := *ebiten.NewImageFromImage()
	// diceImg, _, err := ebitenutil.NewImageFromFile("assets/images/dice.png")
	diceImg := ebiten.NewImageFromImage(DiceImage)

	dieImgSize := diceImg.Bounds().Dx() / 6
	// tileSize := float64(dieImgSize)

	// GAME BOUNDARY ASSIGNMENT
	GAME_BOUNDS_X = dieImgSize * 16
	GAME_BOUNDS_Y = dieImgSize * 9

	render.MinWidth = float64(GAME_BOUNDS_X / 5)
	render.MaxWidth = float64(GAME_BOUNDS_X) - render.MinWidth
	render.MinHeight = float64(GAME_BOUNDS_Y / 5)
	render.MaxHeight = float64(GAME_BOUNDS_Y) - render.MinHeight
	render.DiceBottom = render.MaxHeight / 4.0

	diceSheet := &render.Sprite{
		Image:       diceImg,
		SpriteSheet: render.NewSpriteSheet(6, 7, dieImgSize),
	}

	dice := SetupPlayerDice(diceSheet, dieImgSize)

	return &Game{
		TileSize: dieImgSize,
		// DiceSprite: diceSheet,
		Dice: dice,
	}
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
			X: render.MinWidth + tileSize*float64(i)*2.0,
			Y: render.MaxHeight/2 - tileSize*0.5,
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
			Sprite:        diceSheet,
			DieRenderable: dieRenderable,
			Mode:          ROLLING,
		}
		dice = append(dice, &die)
	}

	return dice
}
