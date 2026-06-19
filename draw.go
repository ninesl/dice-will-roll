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
var (
	opts     = &ebiten.DrawImageOptions{}
	textOpts = &text.DrawOptions{}

	// TODO: die specific shader uniforms, gems, power ups, etc. this is the fun part
	shaderOpts = &ebiten.DrawRectShaderOptions{}
)

func (g *Game) Draw(s *ebiten.Image) {
	s.Clear()

	s.DrawRectShader(GAME_BOUNDS_X, GAME_BOUNDS_Y,
		g.Shaders[shaders.BackgroundShaderKey],
		shaderOpts)

	DrawROLLZONE(s, opts)

	g.RocksRenderer.DrawRocks(s)

	DEBUGView(s, g, textOpts, DEBUGGameView)

	g.DrawDice(s, opts)

	// g.DrawUI(s)

	//s.DrawImage(s, opts)
	opts.GeoM.Reset()
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
