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
	// screen.Clear()
	screen.Clear()

	opts := &ebiten.DrawImageOptions{}

	// DrawROLLZONE(screen, opts)
	g.DrawRocks(screen)

	if g.cursorWithin(render.SCOREZONE) {
		//TODO:FIXME: have to make this work for standard input, etc. will probably change with shaders anyways later
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			DrawSCOREZONE(screen, opts)
		}
	}

	g.DrawDice(screen)

	// s.DrawRectShader(
	// 	screen.Bounds().Dx(), screen.Bounds().Dy(),
	// 	g.Shaders[shaders.FXAAShaderKey],
	// 	&ebiten.DrawRectShaderOptions{
	// 		Images: [4]*ebiten.Image{screen},
	// 	},
	// )

	DEBUGDrawMessage(screen, g.ActiveLevel.String(), 0.0)
	DEBUGDrawMessage(screen, fmt.Sprintf("%.2f fps / %.2f tps\n", ebiten.ActualFPS(), ebiten.ActualTPS()), FONT_SIZE)
	DEBUGDiceValues(screen, g.Dice)

	s.DrawImage(screen, opts)
	opts.GeoM.Reset()
}

func (g *Game) DrawRocks(screen *ebiten.Image) {
	// Use the new efficient rocks renderer
	g.RocksRenderer.Draw(screen)

	// Display stats
	visible, total := g.RocksRenderer.GetStats()
	statsText := fmt.Sprintf("Rocks: %d/%d visible", visible, total)

	textOpts := &text.DrawOptions{}
	textOpts.GeoM.Translate(0, FONT_SIZE*2)
	textOpts.ColorScale.ScaleWithColor(color.White)

	text.Draw(screen, statsText, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   FONT_SIZE,
	}, textOpts)
}

func (g *Game) DrawDice(screen *ebiten.Image) {
	s := int(g.Dice[0].image.Bounds().Dx())

	shader := g.Shaders[shaders.DieShaderKey]

	u := map[string]any{
		"Time":            g.time,
		"DieScale":        1.15,
		"HoveringSpeedUp": 0,
		// "Cursor": []float32{float32(cx), float32(cy)},
	}

	// TODO: die specific shader uniforms, gems, power ups, etc. this is the fun part
	opts := &ebiten.DrawRectShaderOptions{
		Uniforms: u,
	}

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]
		die.image.Clear()

		if die.Mode == DRAG && g.cursorWithin(render.SCOREZONE) {
			opts.Uniforms["HoveringSpeedUp"] = 1

		} else {
			opts.Uniforms["HoveringSpeedUp"] = 0
		}

		opts.Uniforms["FaceLayouts"] = die.LocationsPips()
		opts.Uniforms["ActiveFace"] = die.ActiveFaceIndex()
		opts.Uniforms["Height"] = die.Height
		opts.Uniforms["Direction"] = die.Direction.KageVec2()
		opts.Uniforms["Velocity"] = die.Velocity.KageVec2()
		opts.Uniforms["DieColor"] = die.Color.KageVec3()
		opts.Uniforms["ZRotation"] = die.ZRotation
		opts.Uniforms["Mode"] = int(die.Mode)

		die.image.DrawRectShader(s, s, shader, opts)

		ops := &ebiten.DrawImageOptions{}
		ops.GeoM.Translate(die.Vec2.X, die.Vec2.Y)
		screen.DrawImage(die.image, ops)
	}
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

//
// printf debugging for the window lmao
//

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
