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

	// for i := range g.Dice {
	// 	UpdateDie(g.Dice[i])
	// }

	// HandleDiceCollisions(g.Dice)

	// // clamp and bounce
	// for i := range g.Dice {
	// 	die := g.Dice[i]
	// 	if die.Vec2.X+die.TileSize >= MaxWidth {
	// 		die.Vec2.X = MaxWidth - die.TileSize - 1
	// 		die.Velocity.X = math.Abs(die.Velocity.X) * -1
	// 		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	// 	}
	// 	if die.Vec2.X < MinWidth {
	// 		die.Vec2.X = MinWidth + 1
	// 		die.Velocity.X = math.Abs(die.Velocity.X)
	// 		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	// 	}
	// 	if die.Vec2.Y+die.TileSize >= MaxHeight {
	// 		die.Vec2.Y = MaxHeight - die.TileSize - 1
	// 		die.Velocity.Y = math.Abs(die.Velocity.Y) * -1
	// 		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	// 	}
	// 	if die.Vec2.Y < MinHeight {
	// 		die.Vec2.Y = MinHeight + 1
	// 		die.Velocity.Y = math.Abs(die.Velocity.Y)
	// 		die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
	// 	}
	// 	g.Dice[i] = die
	// }
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
