package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math/rand/v2"
	"strconv"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/controls"
	"github.com/ninesl/dice-will-roll/render/shaders"
	"github.com/ninesl/dice-will-roll/rocks"
	"github.com/ninesl/dice-will-roll/settings"
)

const (
	sizeScoreCount      = rocks.BitSpriteSlopeCodeCount - 1
	defaultRocksPerSize = 1
)

type game struct {
	rocks       rocks.Rocks
	rockAtlas   *rocks.RockSpriteAtlas
	amountScale int
	fontFace    *text.GoTextFace

	debugInput  rocks.DebugInput
	pressedKeys []ebiten.Key
	mouse       controls.MouseInfo
	debugText   string

	updateTick     int
	rocksPerSet    int
	FPS, TPS       int
	dampingEnabled bool
}

func main() {
	rocksPerSize := flag.Int("n", defaultRocksPerSize, "rocks generated for each of the 15 size scores")
	flag.Parse()
	if flag.NArg() > 1 {
		log.Fatal("usage: go run . [rocks-per-size] or go run . -n rocks-per-size")
	}
	if flag.NArg() == 1 {
		value, err := strconv.Atoi(flag.Arg(0))
		if err != nil {
			log.Fatalf("invalid rocks-per-size %q: %v", flag.Arg(0), err)
		}
		*rocksPerSize = value
	}
	if *rocksPerSize <= 0 {
		log.Fatal("rocks-per-size must be greater than zero")
	}
	if *rocksPerSize > int(^uint(0)>>1)/sizeScoreCount {
		log.Fatal("requested rock count is too large")
	}
	rockCount := sizeScoreCount * *rocksPerSize

	settings.InitScreenSettings(ebiten.Monitor())
	rocks.Init(settings.Screen)

	rockState := appendRandomRockSet(rocks.Rocks{}, rockCount, 0)

	shaderMap := shaders.LoadShaders()
	rockAtlas := rocks.InitRockAtlas(
		shaderMap[shaders.RocksShaderKey],
	)
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.ArcadeN_ttf))
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowTitle("Packed Rock Wall Collisions")
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(&game{
		rocks:          rockState,
		rockAtlas:      rockAtlas,
		amountScale:    rocks.RockAmountScaleIndex(rockState.Len()),
		rocksPerSet:    rockCount,
		dampingEnabled: true,
		fontFace: &text.GoTextFace{
			Source: fontSource,
			Size:   settings.Screen.FontSize,
		},
	}); err != nil {
		log.Fatal(err)
	}
}

func appendRandomRockSet(
	state rocks.Rocks,
	rockCount int,
	forceStepping uint32,
) rocks.Rocks {
	for i := range rockCount {
		velocityX, velocityY := randomVelocities()
		x, y := rocks.PackPosition(
			randomCoordinate(settings.Screen.ResolutionX),
			randomCoordinate(settings.Screen.ResolutionY),
			velocityX,
			velocityY,
		)
		slope, stepping := rocks.PackSprite(
			velocityX, velocityY, uint32(rand.N(rocks.AtlasSlopeZFrames)),
			uint32(i%sizeScoreCount+0x1), 0, 0, 0, 0, forceStepping, 0)
		state.PosX, state.PosY = append(state.PosX, x), append(state.PosY, y)
		state.Slope = append(state.Slope, slope)
		state.Stepping = append(state.Stepping, stepping)
	}
	return state
}

func randomCoordinate(maximum int) uint32 {
	return uint32(rand.N(maximum-0x2) + 0x1)
}

func randomVelocities() (int32, int32) {
	for {
		velocityX := int32(rand.N(0xF) - 0x7)
		velocityY := int32(rand.N(0xF) - 0x7)
		if velocityX != 0 || velocityY != 0 {
			return velocityX, velocityY
		}
	}
}

func (g *game) randomizeRockSlopes() {
	for i := range g.rocks.PosX {
		positionX, positionY, _, _ := rocks.UnpackPosition(g.rocks.PosX[i], g.rocks.PosY[i])
		sizeScore, _, _, _, _, _, _, _, _, _ := rocks.UnpackSprite(g.rocks.Slope[i], g.rocks.Stepping[i])
		velocityX, velocityY := randomVelocities()
		g.rocks.PosX[i], g.rocks.PosY[i] = rocks.PackPosition(
			uint32(positionX), uint32(positionY), velocityX, velocityY)
		g.rocks.Slope[i], g.rocks.Stepping[i] = rocks.PackSprite(
			velocityX, velocityY, uint32(rand.N(rocks.AtlasSlopeZFrames)),
			uint32(sizeScore), 0, 0, 0, 0, g.forceSteppingFlag(), 0)
	}
}

func (g *game) forceSteppingFlag() uint32 {
	if g.dampingEnabled {
		return 0x0
	}
	return 0x1
}

func (g *game) toggleDamping() {
	g.dampingEnabled = !g.dampingEnabled
	for i, slope := range g.rocks.Slope {
		sizeScore, slopeZ, _, _, _, _, _, _, slopeX, slopeY := rocks.UnpackSprite(slope, g.rocks.Stepping[i])
		g.rocks.Slope[i], g.rocks.Stepping[i] = rocks.PackSprite(
			int32(slopeX), int32(slopeY), uint32(slopeZ), uint32(sizeScore),
			0, 0, 0, 0, g.forceSteppingFlag(), 0)
	}
}

func (g *game) addRandomRockSet() {
	g.rocks = appendRandomRockSet(g.rocks, g.rocksPerSet, g.forceSteppingFlag())
	g.amountScale = rocks.RockAmountScaleIndex(g.rocks.Len())
}

func hasChanges(di rocks.DebugInput) bool {
	return di.CycleFilter || di.DecrementAmountScale || di.DecrementSizeScale ||
		di.IncreaseSizeScale || di.IncrementAmountScale || di.ToggleMipmaps
}

func (g *game) Update() error {
	g.debugInput = rocks.DebugInput{}
	g.mouse.Update()

	g.pressedKeys = inpututil.AppendJustPressedKeys(g.pressedKeys[:0])
	if len(g.pressedKeys) == 1 {
		switch g.pressedKeys[0] {
		case ebiten.KeyV:
			ebiten.SetVsyncEnabled(!ebiten.IsVsyncEnabled())
		case ebiten.Key0:
			g.toggleDamping()
		case ebiten.KeySpace:
			g.randomizeRockSlopes()
		case ebiten.KeyQ:
			g.addRandomRockSet()
		case ebiten.KeyArrowLeft:
			g.debugInput.IncreaseSizeScale = true
		case ebiten.KeyArrowRight:
			g.debugInput.DecrementSizeScale = true
		case ebiten.KeyArrowUp:
			g.debugInput.IncrementAmountScale = true
		case ebiten.KeyArrowDown:
			g.debugInput.DecrementAmountScale = true
		case ebiten.KeyF:
			g.debugInput.CycleFilter = true
		case ebiten.KeyG:
			g.debugInput.ToggleMipmaps = true
		}
		//g.updateDebugText(rocks.DebugSnapshot(g.rocks, g.rockAtlas, g.amountScale, settings.Screen.DrawOptions.DrawImageOptions))
	}
	if hasChanges(g.debugInput) {
		if g.debugInput.IncrementAmountScale {
			g.amountScale++
			if g.amountScale >= len(g.rockAtlas.Scales) {
				g.amountScale = 0
			}
		} else if g.debugInput.DecrementAmountScale {
			g.amountScale--
			if g.amountScale < 0 {
				g.amountScale = len(g.rockAtlas.Scales) - 1
			}
		}
		settings.Screen.DrawOptions.DrawImageOptions = rocks.ApplyDebugInput(
			g.rocks, settings.Screen.DrawOptions.DrawImageOptions, g.debugInput)
	}

	rocks.UpdateRocks(&g.rocks, g.amountScale, g.updateTick, g.mouse)
	g.updateTick++
	if g.updateTick >= rocks.UpdateStride {
		g.updateTick = 0
	}

	g.debugText = fpsFormating()
	return nil
}

const frmt = "FPS: %.2f, TPS: %.2f"

func fpsFormating() string {
	return fmt.Sprintf(frmt, ebiten.ActualFPS(), ebiten.ActualTPS())
}

func (g *game) Draw(screen *ebiten.Image) {
	settings.Screen.DrawOptions.ColorScale.Reset()
	rocks.Draw(
		g.rocks.PosX, g.rocks.PosY, g.rocks.Slope,
		g.rockAtlas, g.amountScale, screen, settings.Screen.DrawOptions.DrawImageOptions)
	settings.Screen.DrawOptions.GeoM.Reset()
	settings.Screen.DrawOptions.GeoM.Translate(settings.Screen.FontSize, settings.Screen.FontSize)
	settings.Screen.DrawOptions.ColorScale.Reset()
	settings.Screen.DrawOptions.ColorScale.SetWithColor(color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF})
	text.Draw(screen, g.debugText, g.fontFace, settings.Screen.DrawOptions)
}

func (g *game) drawDebugtext(screen *ebiten.Image) {
	for offsetY := -1; offsetY <= 1; offsetY++ {
		for offsetX := -1; offsetX <= 1; offsetX++ {
			if offsetX == 0 && offsetY == 0 {
				continue
			}
			settings.Screen.DrawOptions.GeoM.Reset()
			settings.Screen.DrawOptions.GeoM.Translate(
				settings.Screen.FontSize+float64(offsetX),
				settings.Screen.FontSize+float64(offsetY))
			settings.Screen.DrawOptions.ColorScale.Reset()
			settings.Screen.DrawOptions.ColorScale.SetWithColor(color.Black)
			text.Draw(screen, g.debugText, g.fontFace, settings.Screen.DrawOptions)
		}
	}
	settings.Screen.DrawOptions.GeoM.Reset()
	settings.Screen.DrawOptions.GeoM.Translate(settings.Screen.FontSize, settings.Screen.FontSize)
	settings.Screen.DrawOptions.ColorScale.Reset()
	settings.Screen.DrawOptions.ColorScale.SetWithColor(color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF})
	text.Draw(screen, g.debugText, g.fontFace, settings.Screen.DrawOptions)
}

func (g *game) updateDebugText(debug *rocks.RockDebug) {
	g.debugText = fmt.Sprintf(
		"FPS: %.2f\nTPS: %.2f\nRocks: %d\nSpriteSheetMemory: %.2f MB\nPositionsMemory: %.4f KB\nSpritesMemory: %.4f KB\n\nSize scale (Left/Right): %.1f-%.1f px\nAmount scale (Up/Down): %d (%.1fx)\nMipmaps (G): %t\nFilter (F): %s\nDamping (0): %t\nVSync (V): %t\n\nSpace: randomize rock slopes\nQ: add 1x random rock set",
		debug.FPS,
		debug.TPS,
		g.rocks.Len(),
		debug.SpriteSheetMB,
		debug.PositionsKB,
		debug.SpritesKB,
		debug.MinDrawSize,
		debug.MaxDrawSize,
		debug.AmountScale,
		debug.AmountScaleMultiplier,
		debug.MipmapsEnabled,
		debug.Filter,
		g.dampingEnabled,
		ebiten.IsVsyncEnabled(),
	)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
