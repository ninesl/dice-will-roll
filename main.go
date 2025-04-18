package main

import (
	"bytes"
	"image"
	"log"
	"math/rand/v2"

	_ "embed"
	_ "image/png" // for png encoder

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	GAME_BOUNDS_X int
	GAME_BOUNDS_Y int
	MinWidth      float64
	MaxWidth      float64
	MaxHeight     float64
)

type Game struct {
	DiceSprite *Sprite
	Dice       []*DieRenderable
	TileSize   int
}

func (g *Game) Bounds() (int, int) {
	return g.TileSize * 16, g.TileSize * 9
}

func (g *Game) Update() error {
	for i := range g.Dice {
		UpdateDie(g.Dice[i])
		// UpdateDie(die)
	}

	CheckCollisions(g.Dice)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// ebitenutil.DebugPrint(screen, "Hello, World!")
	opts := &ebiten.DrawImageOptions{}

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]

		opts.GeoM.Translate(die.Vec2.X, die.Vec2.Y)
		screen.DrawImage(
			g.DiceSprite.Image.SubImage(
				g.DiceSprite.SpriteSheet.Rect(die.IndexOnSheet),
			).(*ebiten.Image),
			opts,
		)
		opts.GeoM.Reset()

	}

}

// return the pixels in the game
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return GAME_BOUNDS_X, GAME_BOUNDS_Y
}

// TODO: embedded FS for loading assets
//
//go:embed assets/images/dice.png
var dicePng []byte
var DiceImage image.Image = func() image.Image {
	img, _, err := image.Decode(bytes.NewReader(dicePng))
	if err != nil {
		log.Fatalf("failed to decode embedded image: %v", err)
	}
	return img
}()

func loadGame() *Game {

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
	MaxHeight = float64(GAME_BOUNDS_Y / 4)

	diceSheet := Sprite{
		Image:       diceImg,
		SpriteSheet: NewSpriteSheet(6, 7, dieImgSize),
	}

	var dice []*DieRenderable
	for range 2 {
		directionX := float64(rand.IntN(1) + 1)
		directionY := float64(rand.IntN(1) + 1)
		if directionX == 2 {
			directionX = -1.0
		}
		if directionY == 2 {
			directionY = -1.0
		}

		dieRenderable := DieRenderable{
			Vec2: Vec2{
				X: MinWidth + rand.Float64()*MaxWidth,
				Y: 0 + rand.Float64()*MaxHeight,
				// X: MinWidth*float64(i) + MinWidth,
				// Y: float64(rand.IntN(int(MaxHeight))) - tileSize,
			},
			// Direction: DOWNRIGHT,
			Velocity: Vec2{
				X: (rand.Float64()*40 + 20) * directionX,
				Y: (rand.Float64()*40 + 20) * directionY,
			},
			TileSize: float64(dieImgSize),
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

func main() {
	ebiten.SetWindowSize(1600, 900) // resolution
	ebiten.SetWindowTitle("Dice Will Roll")

	if err := ebiten.RunGame(loadGame()); err != nil {
		log.Fatal(err)
	}
}
