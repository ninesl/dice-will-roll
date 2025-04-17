package main

import (
	"log"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	GAME_BOUNDS_X int
	GAME_BOUNDS_Y int
	MinWidth      float64
	MaxWidth      float64
	MaxHeight     float64
)

type Game struct {
	DiceSheet *SpriteSheet
	Dice      *DieSprite
	TileSize  int
}

func (g *Game) Bounds() (int, int) {
	return g.TileSize * 16, g.TileSize * 9
}

func (g *Game) Update() error {
	UpdateDieSprite(g.Dice)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// ebitenutil.DebugPrint(screen, "Hello, World!")
	opts := &ebiten.DrawImageOptions{}

	opts.GeoM.Translate(g.Dice.Vec2.X, g.Dice.Vec2.Y)
	screen.DrawImage(
		g.Dice.Image.SubImage(
			g.Dice.SpriteSheet.Rect(g.Dice.SpriteSheet.ActiveFrame),
		).(*ebiten.Image),
		opts,
	)
}

// return the pixels in the game
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return GAME_BOUNDS_X, GAME_BOUNDS_Y
}

func loadGame() *Game {
	diceImg, _, err := ebitenutil.NewImageFromFile("assets/images/dice.png")
	if err != nil {
		log.Fatal(err)
	}

	dieImgSize := diceImg.Bounds().Dx() / 6
	tileSize := float64(dieImgSize)

	diceSprite := DieSprite{
		Sprite: Sprite{
			Image:       diceImg,
			SpriteSheet: NewSpriteSheet(6, 7, dieImgSize),
			Vec2: Vec2{
				X: tileSize,
				Y: tileSize,
			},
		},
		// Direction: DOWNRIGHT,
		Velocity: Vec2{
			X: 40 + rand.Float64(),
			Y: 40 + rand.Float64(),
		},
		TileSize: float64(dieImgSize),
	}

	// GAME BOUNDARY ASSIGNMENT
	GAME_BOUNDS_X = dieImgSize * 16
	GAME_BOUNDS_Y = dieImgSize * 9

	MinWidth = float64(GAME_BOUNDS_X / 5)
	MaxWidth = float64(GAME_BOUNDS_X) - MinWidth
	MaxHeight = float64(GAME_BOUNDS_Y / 4)

	return &Game{
		TileSize: dieImgSize,
		Dice:     &diceSprite,
	}
}

func main() {
	ebiten.SetWindowSize(1600, 900) // resolution
	ebiten.SetWindowTitle("Dice Will Roll")

	if err := ebiten.RunGame(loadGame()); err != nil {
		log.Fatal(err)
	}
}
