package main

import (
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
	g.refreshDEBUG()

	opts := &ebiten.DrawImageOptions{}

	DrawROLLZONE(screen, opts)

	if g.cursorWithin(render.SCOREZONE) {
		//TODO:FIXME: have to make this work for standard input, etc. will probably change with shaders anyways later
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			DrawSCOREZONE(screen, opts)
		}
	}

	g.DrawDice(screen)

	DEBUGDrawHandRank(screen, g.Hand)

	// DEBUGDrawCenterSCOREZONE(screen, opts, float64(g.TileSize), g.DEBUG.dieImgTransparent)
	opts.GeoM.Reset()
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

		//could be a loop, but procedural is likely faster
		opts.Uniforms["FaceLayouts"] = die.LocationsPips()
		// opts.Uniforms["FrontFace"] = pipLocations[dice.FrontFace]
		// opts.Uniforms["LeftFace"] = pipLocations[dice.LeftFace]
		// opts.Uniforms["BottomFace"] = pipLocations[dice.BottomFace]
		// opts.Uniforms["TopFace"] = pipLocations[dice.TopFace]
		// opts.Uniforms["RightFace"] = pipLocations[dice.RightFace]
		// opts.Uniforms["BehindFace"] = pipLocations[dice.BehindFace]
		// opts.Uniforms["NumPips"] = die.NumPips()
		opts.Uniforms["ActiveFace"] = die.ActiveFaceIndex()

		// opts.Uniforms["Height"] = die.Height
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

func DEBUGDrawHandRank(screen *ebiten.Image, rank dice.HandRank) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, 0)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, rank.String(), &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   render.GAME_BOUNDS_X * .01,
	}, op)
}
