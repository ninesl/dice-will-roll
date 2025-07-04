package main

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

func (g *Game) Draw(screen *ebiten.Image) {
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

	// DEBUGDrawHandRank(screen, g.Hand, g.ActiveLevel.Rocks)
	var (
		Rolling []*Die
		Held    []*Die
		Scoring []*Die
	)

	for i := 0; i < len(g.Dice); i++ {
		d := g.Dice[i]
		switch d.Mode {
		case ROLLING:
			Rolling = append(Rolling, d)
		case HELD:
			Held = append(Held, d)
		case SCORING:
			Scoring = append(Scoring, d)
		}
	}
	DEBUGDrawMessage(screen, fmt.Sprintf("%v", DEBUGValuesFromDice(Rolling)), FONT_SIZE)
	DEBUGDrawMessage(screen, fmt.Sprintf("%v", DEBUGValuesFromDice(Held)), FONT_SIZE*2)
	DEBUGDrawMessage(screen, fmt.Sprintf("%v", DEBUGValuesFromDice(Scoring)), FONT_SIZE*3)
	DEBUGDrawMessage(screen, g.ActiveLevel.String(), 0.0)

	// DEBUGDrawCenterSCOREZONE(screen, opts, float64(g.TileSize), g.DEBUG.dieImgTransparent)
	opts.GeoM.Reset()
}

func (g *Game) DrawRocks(screen *ebiten.Image) {
	g.RocksImage.Clear()

	shader := g.Shaders[shaders.RocksShaderKey]

	opts := &ebiten.DrawRectShaderOptions{}

	g.RocksImage.DrawRectShader(GAME_BOUNDS_X, GAME_BOUNDS_Y, shader, opts)
	screen.DrawImage(g.RocksImage, &ebiten.DrawImageOptions{})
}

func (g *Game) DrawDice(screen *ebiten.Image) {
	s := int(g.Dice[0].image.Bounds().Dx())

	shader := g.Shaders[shaders.DieShaderKey]

	time := float32(time.Since(g.startTime).Milliseconds()) / float32(ebiten.TPS())
	u := map[string]any{
		// "Time": g.tick,
		// "TargetFace": 0.0,
		"Time":     time,
		"DieScale": 1.15,
		// opts.Uniforms["HoveringSpeedUp"] = false
		"HoveringSpeedUp": 0,
		// "Cursor": []float32{float32(cx), float32(cy)},
	}

	// fmt.Println(u["Time"])
	// fmt.Println(curTime)

	// TODO: die specific shader uniforms, gems, power ups, etc. this is the fun part
	opts := &ebiten.DrawRectShaderOptions{
		Uniforms: u,
	}

	for i := 0; i < len(g.Dice); i++ {
		die := g.Dice[i]
		// fmt.Printf("%d ] %s\n", i, die.Mode)

		die.image.Clear()

		// if die.Mode == DRAG && render.SCOREZONE.ContainsDie(&die.DieRenderable) {
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

		// fmt.Println(opts.Uniforms["Velocity"])
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
// DEBUG
//

type DEBUG struct {
	rolling, held int
}

// func (g *Game) refreshDEBUG() {
// 	g.DEBUG.rolling = 0
// 	g.DEBUG.held = 0
// 	for _, die := range g.Dice {
// 		if die.Mode == ROLLING {
// 			g.DEBUG.rolling++
// 		} else if die.Mode == HELD {
// 			g.DEBUG.held++
// 		}
// 	}
// }

func DEBUGDrawCenterSCOREZONE(screen *ebiten.Image, opts *ebiten.DrawImageOptions, tileSize float64, dieImgTransparent *ebiten.Image) {
	opts.GeoM.Translate(render.GAME_BOUNDS_X/2.0-tileSize/2.0, render.SCOREZONE.MaxHeight/2.0-tileSize/2.0)
	screen.DrawImage(
		dieImgTransparent,
		opts,
	)
}

func DEBUGDrawHandRank(screen *ebiten.Image, rank dice.HandRank, currentRocks int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, 0)
	op.ColorScale.ScaleWithColor(color.White)
	msg := fmt.Sprintf("%-4d rocks : %s %.2fx", currentRocks, rank.String(), rank.Multiplier())
	text.Draw(screen, msg, &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   32,
	}, op)
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
