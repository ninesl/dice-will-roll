package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/render"
)

var (
	GAME_BOUNDS_X int
	GAME_BOUNDS_Y int
)

// Mode is a representation different game states that modify
// controls, what is getting displayed, etc.
type Action uint16

const (
	ROLLING Action = iota
	ROLL
	SCORE
	HELD
	DRAG   // locked to mouse cursor
	PRESS  // when the mouse is pressed
	SELECT // when the mouse is released ie. clicked
)

func (g *Game) UpdateCusor() {
	x, y := ebiten.CursorPosition()
	g.x = float64(x)
	g.y = float64(y)
}
func (g *Game) cursorWithin(zone render.ZoneRenderable) bool {
	return g.x > zone.MinWidth && g.x < zone.MaxWidth && g.y > zone.MinHeight && g.y < zone.MaxHeight
}

func (g *Game) Bounds() (int, int) {
	return g.TileSize * 16, g.TileSize * 9
}

// returns an Action based on player input
//
// input for the controller scheme? TODO:FIXME: idk if this is final
func (g *Game) Controls() Action {
	g.UpdateCusor()

	var action Action = ROLLING // the animation of rolling
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		action = ROLL
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		if g.cursorWithin(render.ROLLZONE) {
			action = PRESS
		}
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0) {
		action = SELECT
	}

	return action
}

func (g *Game) Update() error {
	action := g.Controls()

	g.ControlAction(action)
	g.UpdateDice()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	msg := fmt.Sprintf("T%0.2f F%0.2f x%0.0f y%0.0f", ebiten.ActualTPS(), ebiten.ActualFPS(), g.x, g.y)
	op := &text.DrawOptions{}
	// op.GeoM.Translate(0, 0)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   20,
	}, op)

	opts := &ebiten.DrawImageOptions{}

	if g.cursorWithin(render.SCOREZONE) {
		//TODO:FIXME: have to make this work for standard input, etc. will probably change with shaders anyways later
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			opts.GeoM.Translate(render.SCOREZONE.MinWidth, render.SCOREZONE.MinHeight)
			screen.DrawImage(
				render.SCOREZONE.Sprite(),
				opts,
			)
			opts.GeoM.Reset()
		}
	}

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]
		opts.GeoM.Translate(die.Vec2.X, die.Vec2.Y)
		screen.DrawImage(
			die.Sprite(),
			opts,
		)
		opts.GeoM.Reset()
	}

	// if die.Mode == DRAG {
	// 	opts.GeoM.Translate(float64(g.x), float64(g.y))
	// }

}

func main() {
	ebiten.SetWindowSize(1600, 900) // resolution
	ebiten.SetWindowTitle("Dice Will Roll")

	game := LoadGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
