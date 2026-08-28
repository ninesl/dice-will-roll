package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"

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
	defaultRocksPerSize = 100
)

var tickIntervalKeys = [...]ebiten.Key{
	ebiten.Key1,
	ebiten.Key2,
	ebiten.Key3,
	ebiten.Key4,
	ebiten.Key5,
	ebiten.Key6,
	ebiten.Key7,
	ebiten.Key8,
	ebiten.Key9,
}

type game struct {
	rocks                *rocks.Rocks
	debug                *rocks.RockDebug
	debugLines           []string
	updateTick, tickSize int
	rocksPerSet          int
	dampingEnabled       bool
	mouse                controls.MouseInfo
	fontFace             *text.GoTextFace
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
	textOptions := settings.Screen.DrawOptions

	ebiten.SetWindowTitle("Packed Rock Wall Collisions")
	if err := ebiten.RunGame(&game{
		rocks: &rocks.Rocks{
			Positions:   positions,
			Sprites:     sprites,
			Atlas:       rockAtlas,
			DrawOptions: &textOptions.DrawImageOptions,
			AmountScale: rocks.RockAmountScaleIndex(len(positions)),
		},
		rocksPerSet:    rockCount,
		tickSize:       2,
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
	permaSpin := g.permaSpinFlag()
	for i, packedSprite := range g.rocks.Sprites {
		sizeScore, slopeZ, _, _, _, _, _, _, slopeX, slopeY := rocks.UnpackSprite(packedSprite)
		g.rocks.Sprites[i] = rocks.PackSprite(rocks.InputSprite{
			SlopeX:    int32(slopeX),
			SlopeY:    int32(slopeY),
			SlopeZ:    uint32(slopeZ),
			SizeScore: uint32(sizeScore),
			PermaSpin: permaSpin,
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

	if inpututil.IsKeyJustPressed(ebiten.Key0) {
		g.toggleDamping()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.randomizeRockSlopes()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.addRandomRockSet()
	}

	for intervalIndex, key := range tickIntervalKeys {
		if inpututil.IsKeyJustPressed(key) {
			g.tickSize = intervalIndex + 1
			g.updateTick = 0
		}
	}

	increaseSizeScale := inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)
	incrementAmountScale := inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)
	input := rocks.DebugInput{
		IncreaseSizeScale:    increaseSizeScale,
		DecrementSizeScale:   !increaseSizeScale && inpututil.IsKeyJustPressed(ebiten.KeyArrowRight),
		IncrementAmountScale: incrementAmountScale,
		DecrementAmountScale: !incrementAmountScale && inpututil.IsKeyJustPressed(ebiten.KeyArrowDown),
		CycleFilter:          inpututil.IsKeyJustPressed(ebiten.KeyF),
		ToggleMipmaps:        inpututil.IsKeyJustPressed(ebiten.KeyG),
	}
	rocks.ApplyDebugInput(g.rocks, input)

	rocks.UpdateRocks(g.rocks, g.tickSize, g.updateTick, &g.mouse)
	g.updateTick = (g.updateTick + 1) % g.tickSize
	g.debug = g.rocks.DebugSnapshot()
	g.updateDebugLines()
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	rocks.Draw(g.rocks, screen)
	//g.drawDebugInfo(screen)
}

func (g *game) drawDebugInfo(screen *ebiten.Image) {
	message := strings.Join(g.debugLines, "\n")
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
			text.Draw(screen, message, g.fontFace, drawOptions)
		}
	}
	drawOptions.GeoM.Reset()
	drawOptions.GeoM.Translate(settings.Screen.FontSize, settings.Screen.FontSize)
	drawOptions.ColorScale.Reset()
	drawOptions.ColorScale.SetWithColor(color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF})
	text.Draw(screen, message, g.fontFace, drawOptions)
}

func (g *game) updateDebugLines() {
	g.debugLines = []string{
		fmt.Sprintf("FPS: %.2f", g.debug.FPS),
		fmt.Sprintf("TPS: %.2f", g.debug.TPS),
		fmt.Sprintf("TickSize: %d", g.tickSize),
		fmt.Sprintf("Rocks: %d", len(g.rocks.Positions)),
		fmt.Sprintf("SpriteSheetMemory: %.2f MB", g.debug.SpriteSheetMB),
		fmt.Sprintf("PositionsMemory: %.4f KB", g.debug.PositionsKB),
		fmt.Sprintf("SpritesMemory: %.4f KB", g.debug.SpritesKB),
		fmt.Sprintf("AmountScaleMode: %d (%.1fx)", g.debug.AmountScale, g.debug.AmountScaleMultiplier),
		fmt.Sprintf("Filter: %s", g.debug.Filter),
		fmt.Sprintf("Mipmaps: %t", g.debug.MipmapsEnabled),
		fmt.Sprintf("Slopes: [%d][%d][%d]", g.debug.SlopeX, g.debug.SlopeY, g.debug.SlopeZ),
		fmt.Sprintf("Velocities: [%d][%d]", g.debug.VelocityX, g.debug.VelocityY),
		fmt.Sprintf("Steps: [%d][%d][%d]", g.debug.StepX, g.debug.StepY, g.debug.StepZ),
		fmt.Sprintf("StepTick: %d", g.debug.StepTick),
		fmt.Sprintf("PermaSpin: %d", g.debug.PermaSpin),
		fmt.Sprintf("SpinAgain: %d", g.debug.SpinAgain),
		fmt.Sprintf("PackedSlopes: [%04b][%04b]", g.debug.SlopeX+8, g.debug.SlopeY+8),
		fmt.Sprintf("SizeScore: %04b (%d)", g.debug.SizeScore, g.debug.SizeScore),
		fmt.Sprintf("DrawScale: %.3f", g.debug.Scale),
		fmt.Sprintf("DrawSize: %.0fx%.0f", g.debug.DrawSize, g.debug.DrawSize),
		fmt.Sprintf("CollisionRadius: %d", g.debug.CollisionRadius),
		fmt.Sprintf("PackedPosition: %032b", g.debug.PackedPosition),
		fmt.Sprintf("PackedSprite: %032b", g.debug.PackedSprite),
		"",
		fmt.Sprintf("0: toggle damping (%t)", g.dampingEnabled),
		"Space: randomize rock slopes",
		"Q: add 1x random rock set",
		"1-9: index tick size",
		"F: cycle filter",
		"G: toggle mipmaps",
		"Left/Right: size scale",
		"Up/Down: amount scale mode",
	}
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
