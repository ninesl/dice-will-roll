package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/render"
)

// interface impl
func (g *Game) Draw(screen *ebiten.Image) {
	g.refreshDEBUG()
	DEBUGDrawFPS(screen, g.x, g.y, g.DEBUG.rolling, g.DEBUG.held)
	opts := &ebiten.DrawImageOptions{}

	DrawROLLZONE(screen, opts)

	if g.cursorWithin(render.SCOREZONE) {
		//TODO:FIXME: have to make this work for standard input, etc. will probably change with shaders anyways later
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			DrawSCOREZONE(screen, opts)
		}
	}

	DrawDice(screen, opts, g.Dice)

	// DEBUGDrawCenterSCOREZONE(screen, opts, float64(g.TileSize), g.DEBUG.dieImgTransparent)
	opts.GeoM.Reset()
}

func DrawDice(screen *ebiten.Image, opts *ebiten.DrawImageOptions, dice []*Die) {
	for i := 0; i < len(dice); i++ {
		die := dice[i]
		// halfOut := 0 - die.TileSize/2

		// //lock center to middle to rotate
		// opts.GeoM.Translate(halfOut, halfOut)
		// opts.GeoM.Rotate(-die.Theta) // messes up check for PRESS
		opts.GeoM.Translate(die.Vec2.X, die.Vec2.Y)
		//the whole 'screen' for the sprite.

		// fmt.Println(die.Theta)

		screen.DrawImage(
			die.Sprite(),
			opts,
		)
		opts.GeoM.Reset()
	}
}

func DrawSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	opts.GeoM.Translate(render.SCOREZONE.MinWidth, render.SCOREZONE.MinHeight)
	screen.DrawImage(
		render.SCOREZONE.Sprite(),
		opts,
	)
	opts.GeoM.Reset()
}

func DrawROLLZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	opts.GeoM.Translate(render.ROLLZONE.MinWidth, render.ROLLZONE.MinHeight)
	screen.DrawImage(
		render.ROLLZONE.Sprite(),
		opts,
	)
	opts.GeoM.Reset()

}

//
// printf debugging for the window lmao
// DEBUG
//

type DEBUG struct {
	dieImgTransparent *ebiten.Image
	rolling, held     int
}

func (g *Game) refreshDEBUG() {
	g.DEBUG.rolling = 0
	g.DEBUG.held = 0

	for _, die := range g.Dice {
		if die.Mode == ROLLING {
			g.DEBUG.rolling++
		} else if die.Mode == HELD {
			g.DEBUG.held++
		}
	}
}

func DEBUGDrawCenterSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions, tileSize float64, dieImgTransparent *ebiten.Image) {
	opts.GeoM.Translate(render.GAME_BOUNDS_X/2.0-tileSize/2.0, render.SCOREZONE.MaxHeight/2.0-tileSize/2.0)
	screen.DrawImage(
		dieImgTransparent,
		opts,
	)
}

// TODO: better abstraction than this
func DEBUGDrawFPS(screen *ebiten.Image, x, y float64, rolling, held int) {
	msg := fmt.Sprintf("T%0.2f F%0.2f x%4.0f y%4.0f ", ebiten.ActualTPS(), ebiten.ActualFPS(), x, y)
	msg += fmt.Sprintf("Rolling %d Held %d", rolling, held)
	op := &text.DrawOptions{}
	// op.GeoM.Translate(0, 0)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   20,
	}, op)
}
