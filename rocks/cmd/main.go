package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
	"github.com/ninesl/dice-will-roll/rocks"
	"github.com/ninesl/dice-will-roll/settings"
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
	Rocks                   *rocks.Rocks
	Debug                   *rocks.RockDebug
	animTick, ticksPerFrame int
	fontFace                *text.GoTextFace
	textOpts                *text.DrawOptions
	input                   rocks.DebugInput
}

func main() {
	settings.InitScreenSettings(ebiten.Monitor())
	textOptions := &text.DrawOptions{}
	textOptions.GeoM.Translate(settings.Screen.FontSize, settings.Screen.FontSize)
	textOptions.ColorScale.ScaleWithColor(color.White)
	textOptions.LayoutOptions.LineSpacing = settings.Screen.FontSize * 1.25

	shaderMap := shaders.LoadShaders()
	packedPosition := rocks.PackPosition(
		uint32(settings.Screen.ResolutionX/4),
		uint32(settings.Screen.ResolutionY/2),
		0, 0)
	packedPosition2 := rocks.PackPosition(
		uint32(settings.Screen.ResolutionX/4*3),
		uint32(settings.Screen.ResolutionY/2),
		8, 8)
	packedSprite := uint32(0b0001_0001_0000_1000_0000000000000000)

	positions := rocks.RockPositions{packedPosition, packedPosition2}
	sprites := rocks.RockSprites{packedSprite, packedSprite}

	rockAtlas := rocks.InitRockAtlas(
		shaderMap[shaders.RocksShaderKey],
		settings.Screen.Tiles.HalfTileSize32,
		len(positions))
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.ArcadeN_ttf))
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowTitle("Packed Rock Sprite Rotations")

	game := &game{
		Rocks: &rocks.Rocks{Positions: positions, Sprites: sprites, Atlas: rockAtlas},
		fontFace: &text.GoTextFace{
			Source: fontSource, Size: settings.Screen.FontSize},
		textOpts:      textOptions,
		ticksPerFrame: 2,
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func (g *game) Update() error {
	g.Debug = g.Rocks.DebugSnapshot()

	for intervalIndex, key := range tickIntervalKeys {
		if inpututil.IsKeyJustPressed(key) {
			g.ticksPerFrame = intervalIndex + 1
		}
	}

	g.input.IncreaseSizeScale = inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)
	g.input.DecrementSizeScale = !g.input.IncreaseSizeScale && inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)

	g.input.IncrementAmountScale = inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)
	g.input.DecrementAmountScale = !g.input.IncrementAmountScale && inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)

	g.animTick++

	if g.animTick >= g.ticksPerFrame {
		g.animTick = 0
		if g.Debug.VisitedFrames == g.Debug.TotalFrames {
			fmt.Printf("visited all %d packed rock frames\n", g.Debug.VisitedFrames)
			return ebiten.Termination
		}
		rocks.UpdateDEBUG(g.Rocks, g.input)
		g.input = rocks.DebugInput{}
	}

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	rocks.Draw(g.Rocks, screen)
	g.drawDebugInfo(screen)
}

func (g *game) drawDebugInfo(screen *ebiten.Image) {
	message := fmt.Sprintf(
		"FPS: %.2f\nTPS: %.2f\nTicksPerFrame: %d\nVisitedFrames: %d/%d\nSpriteSheetMemory: %.2f MB\nPositionsMemory: %.4f KB\nSpritesMemory: %.4f KB\nSpriteSlopes: [%d][%d]\nRotationFrame: %d\nRotationSteps: [%d][%d]\nAnimationTick: %d\nPackedSlopes: [%04b][%04b]\nSizeCode: %04b (%d)\nDrawScale: %.3f\nDrawSize: %.0fx%.0f\nPackedPosition: %032b\nPackedSprite: %032b\n\n1-9: ticks per frame",
		g.Debug.FPS,
		g.Debug.TPS,
		g.ticksPerFrame,
		g.Debug.VisitedFrames,
		g.Debug.TotalFrames,
		g.Debug.SpriteSheetMB,
		g.Debug.PositionsKB,
		g.Debug.SpritesKB,
		g.Debug.SlopeX,
		g.Debug.SlopeY,
		g.Debug.RotationFrame,
		g.Debug.RotationStepsX,
		g.Debug.RotationStepsY,
		g.Debug.AnimationTick,
		g.Debug.SlopeX+8,
		g.Debug.SlopeY+8,
		g.Debug.Size,
		g.Debug.Size,
		g.Debug.Scale,
		g.Debug.DrawSize,
		g.Debug.DrawSize,
		g.Debug.PackedPosition,
		g.Debug.PackedSprite,
	)
	text.Draw(screen, message, g.fontFace, g.textOpts)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
