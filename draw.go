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
	DEBUGDrawFPS(screen, g.x, g.y)
	opts := &ebiten.DrawImageOptions{}

	DrawROLLZONE(screen, opts)

	if g.cursorWithin(render.SCOREZONE) {
		//TODO:FIXME: have to make this work for standard input, etc. will probably change with shaders anyways later
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			DrawSCOREZONE(screen, opts)
		}
	}

	DrawDice(screen, opts, g.Dice)

	// DEBUGDrawCenterSCOREZONE(screen, opts, float64(g.TileSize))

	opts.GeoM.Reset()
}

func DrawDice(screen *ebiten.Image, opts *ebiten.DrawImageOptions, dice []*Die) {
	for i := 0; i < len(dice); i++ {
		die := dice[i]
		opts.GeoM.Translate(die.Vec2.X, die.Vec2.Y)
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

func DEBUGDrawCenterSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions, tileSize float64) {
	opts.GeoM.Translate(render.GAME_BOUNDS_X/2.0-tileSize/2.0, render.SCOREZONE.MaxHeight/2.0-tileSize/2.0)
	screen.DrawImage(
		dieImgTransparent,
		opts,
	)
}

func DEBUGDrawFPS(screen *ebiten.Image, x float64, y float64) {
	msg := fmt.Sprintf("T%0.2f F%0.2f x%0.0f y%0.0f", ebiten.ActualTPS(), ebiten.ActualFPS(), x, y)
	op := &text.DrawOptions{}
	// op.GeoM.Translate(0, 0)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   20,
	}, op)
}
