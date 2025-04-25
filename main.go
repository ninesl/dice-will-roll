package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	GAME_BOUNDS_X int
	GAME_BOUNDS_Y int
)

// Mode is a representation different game states that modify
// controls, what is getting displayed, etc.
type Mode uint16

const (
	ROLLING Mode = iota
	SCORE
	HELD
)

func (g *Game) Bounds() (int, int) {
	return g.TileSize * 16, g.TileSize * 9
}

// will change global variables that manage state?
func (g *Game) Controls() {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		for i := range g.Dice {
			g.Dice[i].Roll()
		}
	}
}

func (g *Game) Update() error {
	g.Controls()

	UpdateDice(g.Dice)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	msg := fmt.Sprintf("T%0.2f F%0.2f", ebiten.ActualTPS(), ebiten.ActualFPS())
	op := &text.DrawOptions{}
	// op.GeoM.Translate(0, 0)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   20,
	}, op)

	opts := &ebiten.DrawImageOptions{}

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]

		opts.GeoM.Translate(die.Vec2.X, die.Vec2.Y)
		screen.DrawImage(
			die.Draw(),
			opts,
		)
		opts.GeoM.Reset()

	}

}

func main() {
	ebiten.SetWindowSize(1600, 900) // resolution
	ebiten.SetWindowTitle("Dice Will Roll")

	game := LoadGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
