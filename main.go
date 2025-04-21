package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	GAME_BOUNDS_X int
	GAME_BOUNDS_Y int
	MinWidth      float64
	MaxWidth      float64
	MinHeight     float64
	MaxHeight     float64
	// Usage: DirectionMap[Direction].X * math.Abs(renderable.Velocity.X)
	DirectionMap = map[Direction]Vec2{
		UP:        Vec2{X: 0, Y: -1},
		DOWN:      Vec2{X: 0, Y: 1},
		LEFT:      Vec2{X: -1, Y: 0},
		RIGHT:     Vec2{X: 1, Y: 0},
		UPLEFT:    Vec2{X: -1, Y: 1},
		UPRIGHT:   Vec2{X: 1, Y: -1},
		DOWNRIGHT: Vec2{X: 1, Y: 1},
		DOWNLEFT:  Vec2{X: -1, Y: 1},
	}
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
	// if ebiten.(ebiten.KeySpace) {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		halfTile := float64(g.TileSize) // temp
		for i := range g.Dice {
			dir := Direction(rand.IntN(2) + UPLEFT) // random direction
			direction := DirectionMap[dir]

			g.Dice[i].Velocity.X = halfTile * direction.X
			g.Dice[i].Velocity.Y = halfTile * direction.Y
		}
	}
	for i := range g.Dice {
		UpdateDie(g.Dice[i])
		// XVel := (g.Dice[i].Velocity.X + 1.0) * 2.0
		// if XVel > halfTile {
		// 	XVel = halfTile
		// }
		// YVel := (g.Dice[i].Velocity.Y + 1.0) * 2.0
		// if YVel > halfTile {
		// 	YVel = halfTile
		// }
		// UpdateDie(die)
	}

	HandleDiceCollisions(g.Dice)

	// clamp and bounce
	for i := range g.Dice {
		die := g.Dice[i]
		if die.Vec2.X+die.TileSize >= MaxWidth {
			die.Vec2.X = MaxWidth - die.TileSize - 1
			die.Velocity.X = math.Abs(die.Velocity.X) * -1
			die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
		}
		if die.Vec2.X < MinWidth {
			die.Vec2.X = MinWidth + 1
			die.Velocity.X = math.Abs(die.Velocity.X)
			die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
		}
		if die.Vec2.Y+die.TileSize >= MaxHeight {
			die.Vec2.Y = MaxHeight - die.TileSize - 1
			die.Velocity.Y = math.Abs(die.Velocity.Y) * -1
			die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
		}
		if die.Vec2.Y < MinHeight {
			die.Vec2.Y = MinHeight + 1
			die.Velocity.Y = math.Abs(die.Velocity.Y)
			die.IndexOnSheet = die.ColorSpot + rand.IntN(5)
		}
		g.Dice[i] = die
	}
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
			g.DiceSprite.Image.SubImage(
				g.DiceSprite.SpriteSheet.Rect(die.IndexOnSheet),
			).(*ebiten.Image),
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
