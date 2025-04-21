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

	MinWidth = float64(GAME_BOUNDS_X / 5)
	MaxWidth = float64(GAME_BOUNDS_X) - MinWidth
	MinHeight = float64(GAME_BOUNDS_Y / 5)
	MaxHeight = float64(GAME_BOUNDS_Y) - MinHeight
	DiceBottom = MaxHeight / 4.0

	diceSheet := Sprite{
		Image:       diceImg,
		SpriteSheet: NewSpriteSheet(6, 7, dieImgSize),
	}

	var dice []*DieRenderable
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

		pos := Vec2{
			X: MinWidth + tileSize*float64(i)*2.0,
			Y: MaxHeight/2 - tileSize*0.5,
		}
		dieRenderable := DieRenderable{
			Fixed: pos,
			Vec2:  pos,
			Velocity: Vec2{
				X: (rand.Float64()*40 + 20) * float64(i) * -1.0,
				Y: (rand.Float64()*40 + 20) * float64(i) * -1.0,
			},
			TileSize:  float64(dieImgSize),
			ColorSpot: i * 6,
		}
		dice = append(dice, &dieRenderable)
	}

	// for die := range dice {
	// 	fmt.Println(dice[die])
	// }

	return &Game{
		TileSize:   dieImgSize,
		DiceSprite: &diceSheet,
		Dice:       dice,
	}
}
