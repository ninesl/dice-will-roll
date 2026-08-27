package main

import (
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render/shaders"
	"github.com/ninesl/dice-will-roll/rocks"
	"github.com/ninesl/dice-will-roll/settings"
)

const rockCount = rocks.BitSpriteSlopeCodeCount - 1

type game struct {
	rocks *rocks.Rocks
}

func main() {
	settings.InitScreenSettings(ebiten.Monitor())
	rocks.Init(settings.Screen)

	positions := make(rocks.RockPositions, rockCount)
	sprites := make(rocks.RockSprites, rockCount)
	for i := range rockCount {
		slopeX, slopeY := randomSlopes()
		positions[i] = rocks.PackPosition(
			randomCoordinate(settings.Screen.ResolutionX),
			randomCoordinate(settings.Screen.ResolutionY),
			slopeX,
			slopeY,
		)
		sprites[i] = rocks.PackSprite(
			uint32(i+1),
			uint32(rand.Intn(rocks.AtlasRotationFrames)),
			0,
			0,
			0,
			slopeX,
			slopeY,
		)
	}

	shaderMap := shaders.LoadShaders()
	rockAtlas := rocks.InitRockAtlas(
		shaderMap[shaders.RocksShaderKey],
		settings.Screen.Tiles.HalfTileSize32,
		rockCount,
	)

	ebiten.SetWindowTitle("Packed Rock Wall Collisions")
	if err := ebiten.RunGame(&game{rocks: &rocks.Rocks{
		Positions: positions,
		Sprites:   sprites,
		Atlas:     rockAtlas,
	}}); err != nil {
		log.Fatal(err)
	}
}

func randomCoordinate(maximum int) uint32 {
	return uint32(rand.Intn(maximum-2) + 1)
}

func randomSlopes() (int32, int32) {
	for {
		slopeX := int32(rand.Intn(15) - 7)
		slopeY := int32(rand.Intn(15) - 7)
		if slopeX != 0 || slopeY != 0 {
			return slopeX, slopeY
		}
	}
}

func (g *game) Update() error {
	rocks.UpdateRocks(g.rocks)
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	rocks.Draw(g.rocks, screen)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
