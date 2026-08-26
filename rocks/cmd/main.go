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

const packedSprite = 0b0001_0001_0000_1000_0000000000000000

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
	Rocks         rocks.Rocks
	frameTick     int
	ticksPerFrame int
	visitedFrames int
	spriteSheetMB float64
	positionsKB   float64
	spritesKB     float64
	fontFace      *text.GoTextFace
}

func main() {
	settings.InitScreenSettings(ebiten.Monitor())

	shaderMap := shaders.LoadShaders()

	rockSprite := rocks.InitRockAtlas(shaderMap[shaders.RocksShaderKey],
		settings.Screen.Tiles.TileSize32)
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.ArcadeN_ttf))
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowTitle("Packed Rock Sprite Rotations")
	spriteBounds := rockSprite.Image.Bounds()
	spriteSheetMB := float64(spriteBounds.Dx()*spriteBounds.Dy()*4) / (1024 * 1024)
	packedPosition := rocks.PackPosition(
		uint32(settings.Screen.ResolutionX/2),
		uint32(settings.Screen.ResolutionY/2),
		0,
		0,
	)

	game := &game{
		Rocks: rocks.Rocks{
			Positions: rocks.RockPositions{packedPosition},
			Sprites:   rocks.RockSprites{packedSprite},
			Atlas:     rockSprite,
		},
		ticksPerFrame: 3,
		visitedFrames: 1,
		spriteSheetMB: spriteSheetMB,
		fontFace:      &text.GoTextFace{Source: fontSource, Size: settings.Screen.FontSize},
	}
	game.positionsKB = float64(32*len(game.Rocks.Positions)) / (8 * 1024)
	game.spritesKB = float64(32*len(game.Rocks.Sprites)) / (8 * 1024)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func (g *game) Update() error {
	for intervalIndex, key := range tickIntervalKeys {
		if inpututil.IsKeyJustPressed(key) {
			g.ticksPerFrame = intervalIndex + 1
		}
	}

	g.frameTick++
	if g.frameTick >= g.ticksPerFrame {
		g.frameTick = 0
		totalFrames := 15 * 15 * rocks.BitSpriteRotationFrames
		if g.visitedFrames == totalFrames {
			fmt.Printf("visited all %d packed rock frames\n", g.visitedFrames)
			return ebiten.Termination
		}
		rocks.UpdateDEBUG(g.Rocks.Positions, g.Rocks.Sprites)
		g.visitedFrames++
	}

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	rocks.Draw(&g.Rocks, screen)
	g.drawDebugInfo(screen)
}

func (g *game) drawDebugInfo(screen *ebiten.Image) {
	size, rotation, slopeX, slopeY := rocks.UnpackSprite(g.Rocks.Sprites[0])
	scale := g.Rocks.Atlas.Scales[size]
	drawSize := float64(g.Rocks.Atlas.SpriteSheet.TileSize) * scale

	message := fmt.Sprintf(
		"FPS: %.2f\nTPS: %.2f\nTicksPerFrame: %d\nVisitedFrames: %d/%d\nSpriteSheetMemory: %.2f MB\nPositionsMemory: %.4f KB\nSpritesMemory: %.4f KB\nSpriteSlopes: [%d][%d]\nRotationFrame: %d\nPackedSlopes: [%04b][%04b]\nSizeCode: %04b (%d)\nDrawScale: %.3f\nDrawSize: %.0fx%.0f\nPackedPosition: %032b\nPackedSprite: %032b\n\n1-9: ticks per frame",
		ebiten.ActualFPS(),
		ebiten.ActualTPS(),
		g.ticksPerFrame,
		g.visitedFrames,
		15*15*rocks.BitSpriteRotationFrames,
		g.spriteSheetMB,
		g.positionsKB,
		g.spritesKB,
		slopeX,
		slopeY,
		rotation,
		slopeX+8,
		slopeY+8,
		size,
		size,
		scale,
		drawSize,
		drawSize,
		g.Rocks.Positions[0],
		g.Rocks.Sprites[0],
	)
	textOptions := &text.DrawOptions{}
	textOptions.GeoM.Translate(settings.Screen.FontSize, settings.Screen.FontSize)
	textOptions.ColorScale.ScaleWithColor(color.White)
	textOptions.LayoutOptions.LineSpacing = settings.Screen.FontSize * 1.25
	text.Draw(screen, message, g.fontFace, textOptions)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
