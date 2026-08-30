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
	fontFace    *text.GoTextFace
	drawOptions ebiten.DrawImageOptions
	pressedKeys []ebiten.Key
	mouse       controls.MouseInfo
	debugText   string

	updateTick     int
	rocksPerSet    int
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

	positions, sprites := appendRandomRockSet(nil, nil, rockCount, 0)

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
		rocks: rocks.Rocks{
			Positions:   positions,
			Sprites:     sprites,
			Atlas:       rockAtlas,
			AmountScale: rocks.RockAmountScaleIndex(len(positions)),
		},
		drawOptions:    settings.Screen.DrawOptions.DrawImageOptions,
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
	positions rocks.RockPositions,
	sprites rocks.RockSprites,
	rockCount int,
	permaSpin uint32,
) (rocks.RockPositions, rocks.RockSprites) {
	for i := range rockCount {
		velocityX, velocityY := randomVelocities()
		positions = append(positions, rocks.PackPosition(
			randomCoordinate(settings.Screen.ResolutionX),
			randomCoordinate(settings.Screen.ResolutionY),
			velocityX,
			velocityY,
		))
		sprites = append(sprites, rocks.PackSprite(rocks.InputSprite{
			SlopeX:    velocityX,
			SlopeY:    velocityY,
			SlopeZ:    uint32(rand.N(rocks.AtlasSlopeZFrames)),
			SizeScore: uint32(i%sizeScoreCount + 0x1),
			StepY:     0x0,
			StepX:     0x0,
			StepZ:     0x0,
			StepTick:  0x0,
			PermaSpin: permaSpin,
			SpinAgain: 0x0,
		}))
	}
	return positions, sprites
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
	for i, packedPosition := range g.rocks.Positions {
		positionX, positionY, _, _ := rocks.UnpackPosition(packedPosition)
		sizeScore, _, _, _, _, _, _, _, _, _ := rocks.UnpackSprite(g.rocks.Sprites[i])
		velocityX, velocityY := randomVelocities()
		g.rocks.Positions[i] = rocks.PackPosition(
			uint32(positionX), uint32(positionY), velocityX, velocityY)
		g.rocks.Sprites[i] = rocks.PackSprite(rocks.InputSprite{
			SlopeX:    velocityX,
			SlopeY:    velocityY,
			SlopeZ:    uint32(rand.N(rocks.AtlasSlopeZFrames)),
			SizeScore: uint32(sizeScore),
			PermaSpin: g.permaSpinFlag(),
		})
	}
}

func (g *game) permaSpinFlag() uint32 {
	if g.dampingEnabled {
		return 0x0
	}
	return 0x1
}

func (g *game) toggleDamping() {
	g.dampingEnabled = !g.dampingEnabled
	for i, packedSprite := range g.rocks.Sprites {
		sizeScore, slopeZ, _, _, _, _, _, _, slopeX, slopeY := rocks.UnpackSprite(packedSprite)
		g.rocks.Sprites[i] = rocks.PackSprite(rocks.InputSprite{
			SlopeX:    int32(slopeX),
			SlopeY:    int32(slopeY),
			SlopeZ:    uint32(slopeZ),
			SizeScore: uint32(sizeScore),
			PermaSpin: g.permaSpinFlag(),
		})
	}
}

func (g *game) addRandomRockSet() {
	g.rocks.Positions, g.rocks.Sprites = appendRandomRockSet(
		g.rocks.Positions,
		g.rocks.Sprites,
		g.rocksPerSet,
		g.permaSpinFlag(),
	)
	g.rocks.AmountScale = rocks.RockAmountScaleIndex(len(g.rocks.Positions))
}

func (g *game) Update() error {
	mouseX, mouseY := ebiten.CursorPosition()
	g.mouse.LastPosition = g.mouse.Position
	g.mouse.Position.X = float32(mouseX)
	g.mouse.Position.Y = float32(mouseY)
	g.mouse.Down = ebiten.IsMouseButtonPressed(ebiten.MouseButton0)
	g.mouse.Clicked = inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0)
	g.mouse.Released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0)
	g.pressedKeys = inpututil.AppendJustPressedKeys(g.pressedKeys[:0])
	input := rocks.DebugInput{}
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
			input.IncreaseSizeScale = true
		case ebiten.KeyArrowRight:
			input.DecrementSizeScale = true
		case ebiten.KeyArrowUp:
			input.IncrementAmountScale = true
		case ebiten.KeyArrowDown:
			input.DecrementAmountScale = true
		case ebiten.KeyF:
			input.CycleFilter = true
		case ebiten.KeyG:
			input.ToggleMipmaps = true
		}
	}
	g.rocks, g.drawOptions = rocks.ApplyDebugInput(g.rocks, g.drawOptions, input)

	rocks.UpdateRocks(g.rocks, g.updateTick, g.mouse)
	g.updateTick = (g.updateTick + 1) % rocks.UpdateStride
	g.updateDebugText(g.rocks.DebugSnapshot(g.drawOptions))
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	settings.Screen.DrawOptions.ColorScale.Reset()
	rocks.Draw(g.rocks, screen, g.drawOptions)
	g.drawDebugInfo(screen)
}

func (g *game) drawDebugInfo(screen *ebiten.Image) {
	drawOptions := settings.Screen.DrawOptions
	for offsetY := -1; offsetY <= 1; offsetY++ {
		for offsetX := -1; offsetX <= 1; offsetX++ {
			if offsetX == 0 && offsetY == 0 {
				continue
			}
			drawOptions.GeoM.Reset()
			drawOptions.GeoM.Translate(
				settings.Screen.FontSize+float64(offsetX),
				settings.Screen.FontSize+float64(offsetY))
			drawOptions.ColorScale.Reset()
			drawOptions.ColorScale.SetWithColor(color.Black)
			text.Draw(screen, g.debugText, g.fontFace, drawOptions)
		}
	}
	drawOptions.GeoM.Reset()
	drawOptions.GeoM.Translate(settings.Screen.FontSize, settings.Screen.FontSize)
	drawOptions.ColorScale.Reset()
	drawOptions.ColorScale.SetWithColor(color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF})
	text.Draw(screen, g.debugText, g.fontFace, drawOptions)
}

func (g *game) updateDebugText(debug *rocks.RockDebug) {
	g.debugText = fmt.Sprintf(
		"FPS: %.2f\nTPS: %.2f\nRocks: %d\nSpriteSheetMemory: %.2f MB\nPositionsMemory: %.4f KB\nSpritesMemory: %.4f KB\nAmountScaleMode: %d (%.1fx)\nFilter: %s\nMipmaps: %t\nSlopes: [%d][%d][%d]\nVelocities: [%d][%d]\nSteps: [%d][%d][%d]\nStepTick: %d\nPermaSpin: %d\nSpinAgain: %d\nPackedSlopes: [%04b][%04b]\nSizeScore: %04b (%d)\nDrawScale: %.3f\nDrawSize: %.0fx%.0f\nDrawSizeRange: %.1f-%.1f px\nCollisionRadius: %d\nPackedPosition: %032b\nPackedSprite: %032b\n\n0: toggle damping (%t)\nV: toggle VSync (%t)\nSpace: randomize rock slopes\nQ: add 1x random rock set\nF: cycle filter\nG: toggle mipmaps\nLeft/Right: size scale\nUp/Down: amount scale mode",
		debug.FPS,
		debug.TPS,
		len(g.rocks.Positions),
		debug.SpriteSheetMB,
		debug.PositionsKB,
		debug.SpritesKB,
		debug.AmountScale,
		debug.AmountScaleMultiplier,
		debug.Filter,
		debug.MipmapsEnabled,
		debug.SlopeX,
		debug.SlopeY,
		debug.SlopeZ,
		debug.VelocityX,
		debug.VelocityY,
		debug.StepX,
		debug.StepY,
		debug.StepZ,
		debug.StepTick,
		debug.PermaSpin,
		debug.SpinAgain,
		debug.SlopeX+8,
		debug.SlopeY+8,
		debug.SizeScore,
		debug.SizeScore,
		debug.Scale,
		debug.DrawSize,
		debug.DrawSize,
		debug.MinDrawSize,
		debug.MaxDrawSize,
		debug.CollisionRadius,
		debug.PackedPosition,
		debug.PackedSprite,
		g.dampingEnabled,
		ebiten.IsVsyncEnabled(),
	)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
