package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	TileSize int
	Dice     *Sprite
}

func (g *Game) Update() error {

	g.Dice.X++
	g.Dice.Y++
	g.Dice.SpriteSheet.Animate()

	// fmt.Println("HELLO?")
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// ebitenutil.DebugPrint(screen, "Hello, World!")
	opts := &ebiten.DrawImageOptions{}

	opts.GeoM.Translate(g.Dice.X, g.Dice.Y)
	screen.DrawImage(
		g.Dice.Image.SubImage(
			g.Dice.SpriteSheet.Rect(g.Dice.SpriteSheet.ActiveFrame),
		).(*ebiten.Image),
		opts,
	)
}

// 320 x 240 is the pixels in the game
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.TileSize * 16, g.TileSize * 9
}

func loadGame() *Game {
	diceImg, _, err := ebitenutil.NewImageFromFile("assets/images/dice.png")
	if err != nil {
		log.Fatal(err)
	}

	dieImgSize := diceImg.Bounds().Dx() / 6

	diceSprite := Sprite{
		Image:       diceImg,
		SpriteSheet: NewSpriteSheet(6, 7, dieImgSize),
	}

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
