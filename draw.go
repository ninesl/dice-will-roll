package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// var screen = ebiten.NewImage(GAME_BOUNDS_X, GAME_BOUNDS_Y)

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

	DEBUGView(s, g, g.opts.text, DEBUGGameView)

	g.DrawDice(s, g.opts.image)

	// g.DrawUI(s)

	//s.DrawImage(s, opts)
}

type DEBUGViewMode int

const (
	DEBUGGameView DEBUGViewMode = iota
)

func DEBUGView(screen *ebiten.Image, g *Game, textOpts *text.DrawOptions, viewMode DEBUGViewMode) {
	DEBUGDrawMessage(screen, textOpts, g.ActiveLevel.String(), 0.0)
	DEBUGDrawMessage(screen, textOpts, fmt.Sprintf("%.2f fps / %.2f tps\n", ebiten.ActualFPS(), ebiten.ActualTPS()), FONT_SIZE)
	DEBUGDiceValues(screen, textOpts, g.Dice)

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

func DEBUGDrawMessage(screen *ebiten.Image, textOpts *text.DrawOptions, msg string, y float64) {
	textOpts.GeoM.Translate(0, float64(y))
	textOpts.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   FONT_SIZE,
	}, textOpts)
	textOpts.GeoM.Reset()
	textOpts.ColorScale.Reset()
}

func DEBUGValuesFromDice(dice []*Die) []int {
	var track []int
	for i := 0; i < len(dice); i++ {
		track = append(track, dice[i].ActiveFace().NumPips())
	}
	return track
}

func DEBUGDiceValues(screen *ebiten.Image, textOpts *text.DrawOptions, dice []*Die) {
	var (
		Rolling []*Die
		Held    []*Die
		Scoring []*Die
	)
	for i := 0; i < len(dice); i++ {
		d := dice[i]
		switch d.Mode {
		case ROLLING:
			Rolling = append(Rolling, d)
		case HELD:
			Held = append(Held, d)
		case SCORING:
			Scoring = append(Scoring, d)
		}
	}
	y := (float64(render.GAME_BOUNDS_Y) - FONT_SIZE)
	DEBUGDrawMessage(screen, textOpts, fmt.Sprintf("%5s%v", "roll", DEBUGValuesFromDice(Rolling)), y)
	DEBUGDrawMessage(screen, textOpts, fmt.Sprintf("%5s%v", "held", DEBUGValuesFromDice(Held)), y-FONT_SIZE)
	DEBUGDrawMessage(screen, textOpts, fmt.Sprintf("%5s%v", "score", DEBUGValuesFromDice(Scoring)), y-FONT_SIZE*2)
}

func (g *Game) DrawDice(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	//sideLen := int(g.Dice[0].image.Bounds().Dx())
	shader := g.Shaders[shaders.DieShaderKey]

	g.opts.shader.Uniforms = map[string]any{
		"Time":            g.time,
		"DieScale":        1.15,
		"HoveringSpeedUp": 0,
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
