package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/music"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// var screen = ebiten.NewImage(GAME_BOUNDS_X, GAME_BOUNDS_Y)

// NOTE:
// text itself should be bespoke,
// as in each element will have a text DrawOptions
// and can use this to draw text onitself when needed
// FX shaders can be built within the element as needed bc of shader draw options
//
// debug elements are their own thing,
// bespoke buttons are their own thing,
// random sprites are their own thing,
// random SDFs or shaders are their own thing,

type DrawOptions struct {
	image *ebiten.DrawImageOptions
	text  *text.DrawOptions
	// TODO: die specific shader uniforms, gems, power ups, etc. this is the fun part
	shader *ebiten.DrawRectShaderOptions
}

func (g *Game) Draw(s *ebiten.Image) {
	s.Clear()
	// likely redundant
	g.opts.image.GeoM.Reset()

	s.DrawRectShader(GAME_BOUNDS_X, GAME_BOUNDS_Y,
		g.Shaders[shaders.BackgroundShaderKey],
		g.opts.shader)

	DrawROLLZONE(s, g.opts.image)

	g.RocksRenderer.DrawRocks(s)

	DEBUGView(s, g, g.opts.text, DEBUGPLAYView)

	g.DrawDice(s, g.opts.image)

	//g.DrawUI(s, g.opts)
	// g.DrawUI(s)

	//s.DrawImage(s, opts)
}

func DrawSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	opts.GeoM.Translate(float64(render.SCOREZONE.MinWidth), float64(render.SCOREZONE.MinHeight))
	screen.DrawImage(
		render.SCOREZONE.Image,
		opts,
	)
	opts.GeoM.Reset()
}

func DrawROLLZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	opts.GeoM.Translate(float64(render.ROLLZONE.MinWidth), float64(render.ROLLZONE.MinHeight))
	screen.DrawImage(
		render.ROLLZONE.Image,
		opts,
	)
	opts.GeoM.Reset()

}

// printf debugging for the window lmao

func DEBUGDrawCenterSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions, tileSize float32, dieImgTransparent *ebiten.Image) {
	opts.GeoM.Translate(
		float64(render.GAME_BOUNDS_X/2.0-tileSize/2.0),
		float64(render.SCOREZONE.MaxHeight/2.0-tileSize/2.0),
	)
	screen.DrawImage(
		dieImgTransparent,
		opts,
	)
}

func (g *Game) DrawDice(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	//sideLen := int(g.Dice[0].image.Bounds().Dx())
	shader := g.Shaders[shaders.DieShaderKey]

	g.opts.shader.Uniforms = map[string]any{
		"Time":            g.time,
		"DieScale":        1.15,
		"HoveringSpeedUp": 0,
		"LaneOneMS":       g.laneOneMS(),
		// "Cursor": []float32{float32(cx), float32(cy)},
	}

	for i := 0; i < len(g.Dice); i++ {
		g.Dice[i].image.Clear()

		if g.Dice[i].Mode == DRAG && g.cursorWithin(render.SCOREZONE) {
			g.opts.shader.Uniforms["HoveringSpeedUp"] = 1

		} else {
			g.opts.shader.Uniforms["HoveringSpeedUp"] = 0
		}

		g.opts.shader.Uniforms["FaceLayouts"] = g.Dice[i].LocationsPips()
		g.opts.shader.Uniforms["ActiveFace"] = g.Dice[i].ActiveFaceIndex()
		g.opts.shader.Uniforms["Height"] = g.Dice[i].Height
		g.opts.shader.Uniforms["Direction"] = g.Dice[i].Direction.KageVec2()
		g.opts.shader.Uniforms["Velocity"] = g.Dice[i].Velocity.KageVec2()
		g.opts.shader.Uniforms["DieColor"] = g.Dice[i].Color.KageVec3()
		g.opts.shader.Uniforms["ZRotation"] = g.Dice[i].ZRotation
		g.opts.shader.Uniforms["Mode"] = int(g.Dice[i].Mode)

		g.Dice[i].image.DrawRectShader(TILE_SIZE, TILE_SIZE, shader, g.opts.shader)

		ops := &ebiten.DrawImageOptions{}
		ops.GeoM.Translate(float64(g.Dice[i].Vec2.X), float64(g.Dice[i].Vec2.Y))
		screen.DrawImage(g.Dice[i].image, ops)
		ops.GeoM.Reset()
	}
}

func (g *Game) laneOneMS() float32 {
	if g.Music == nil {
		return 1000
	}
	ms := g.Music.UpcomingMS(music.LaneOne) - g.Music.MS()
	if ms < 0 {
		return 0
	}
	return float32(ms)
}
