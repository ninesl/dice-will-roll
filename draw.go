package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

var screen = ebiten.NewImage(GAME_BOUNDS_X, GAME_BOUNDS_Y)

func (g *Game) Draw(s *ebiten.Image) {
	screen.Clear()

	screen.DrawRectShader(GAME_BOUNDS_X, GAME_BOUNDS_Y,
		g.Shaders[shaders.BackgroundShaderKey],
		&ebiten.DrawRectShaderOptions{})

	opts := &ebiten.DrawImageOptions{}

	DrawROLLZONE(screen, opts)
	g.RocksRenderer.DrawRocks(screen)

	if g.cursorWithin(render.SCOREZONE) {
		//TODO:FIXME: have to make this work for standard input, etc. will probably change with shaders anyways later
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			DrawSCOREZONE(screen, opts)
		}
	}

	g.DrawDice(screen)

	DEBUGDrawMessage(screen, g.ActiveLevel.String(), 0.0)
	DEBUGDrawMessage(screen, fmt.Sprintf("%.2f fps / %.2f tps\n", ebiten.ActualFPS(), ebiten.ActualTPS()), FONT_SIZE)
	DEBUGDiceValues(screen, g.Dice)

	s.DrawImage(screen, opts)
	opts.GeoM.Reset()
}

func DrawSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	opts.GeoM.Translate(render.SCOREZONE.MinWidth, render.SCOREZONE.MinHeight)
	screen.DrawImage(
		render.SCOREZONE.Image,
		opts,
	)
	opts.GeoM.Reset()
}

func DrawROLLZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	opts.GeoM.Translate(render.ROLLZONE.MinWidth, render.ROLLZONE.MinHeight)
	screen.DrawImage(
		render.ROLLZONE.Image,
		opts,
	)
	opts.GeoM.Reset()

}

// printf debugging for the window lmao

func DEBUGDrawCenterSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions, tileSize float64, dieImgTransparent *ebiten.Image) {
	opts.GeoM.Translate(render.GAME_BOUNDS_X/2.0-tileSize/2.0, render.SCOREZONE.MaxHeight/2.0-tileSize/2.0)
	screen.DrawImage(
		dieImgTransparent,
		opts,
	)
}

func DEBUGDrawMessage(screen *ebiten.Image, msg string, y float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, y)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   FONT_SIZE,
	}, op)
}

func DEBUGValuesFromDice(dice []*Die) []int {
	var track []int
	for i := 0; i < len(dice); i++ {
		track = append(track, dice[i].ActiveFace().NumPips())
	}
	return track
}

func DEBUGDiceValues(screen *ebiten.Image, dice []*Die) {
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
	DEBUGDrawMessage(screen, fmt.Sprintf("%5s%v", "roll", DEBUGValuesFromDice(Rolling)), render.GAME_BOUNDS_Y-FONT_SIZE)
	DEBUGDrawMessage(screen, fmt.Sprintf("%5s%v", "held", DEBUGValuesFromDice(Held)), render.GAME_BOUNDS_Y-FONT_SIZE*2)
	DEBUGDrawMessage(screen, fmt.Sprintf("%5s%v", "score", DEBUGValuesFromDice(Scoring)), render.GAME_BOUNDS_Y-FONT_SIZE*3)
}
