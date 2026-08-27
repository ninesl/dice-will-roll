package rocks

import (
	"simd"

	"github.com/hajimehoshi/ebiten/v2"
)

type RockDebug struct {
	FPS, TPS                       float64
	Scale, DrawSize                float64
	SpriteSheetMB                  float64
	PositionsKB, SpritesKB         float64
	TotalFrames, VisitedFrames     int
	Size, RotationFrame            int
	RotationStepsX, RotationStepsY int
	AnimationTick, SlopeX, SlopeY  int
	PackedPosition, PackedSprite   uint32
}

func (rocks Rocks) DebugSnapshot() *RockDebug {
	packedPosition := rocks.Positions[0]
	packedSprite := rocks.Sprites[0]
	size, rotation,
		rotationStepsX, rotationStepsY, animationTick,
		slopeX, slopeY := UnpackSprite(packedSprite)
	scale := rocks.Atlas.Scales[rocks.Atlas.AmountScale][size]
	bounds := rocks.Atlas.Image.Bounds()

	return &RockDebug{
		FPS:            ebiten.ActualFPS(),
		TPS:            ebiten.ActualTPS(),
		Scale:          scale,
		DrawSize:       float64(rocks.Atlas.SpriteSheet.TileSize) * scale,
		SpriteSheetMB:  float64(bounds.Dx()*bounds.Dy()*4) / (1024 * 1024),
		PositionsKB:    float64(len(rocks.Positions)*4) / 1024,
		SpritesKB:      float64(len(rocks.Sprites)*4) / 1024,
		TotalFrames:    atlasSlopeStates * atlasSlopeStates * AtlasRotationFrames,
		VisitedFrames:  filterIndex(slopeX, slopeY, rotation) + 1,
		Size:           size,
		RotationFrame:  rotation,
		RotationStepsX: rotationStepsX,
		RotationStepsY: rotationStepsY,
		AnimationTick:  animationTick,
		SlopeX:         slopeX,
		SlopeY:         slopeY,
		PackedPosition: packedPosition,
		PackedSprite:   packedSprite,
	}
}

type DebugInput struct {
	IncreaseSizeScale    bool
	DecrementSizeScale   bool
	IncrementAmountScale bool
	DecrementAmountScale bool
}

func inputValue(active bool) uint32 {
	if active {
		return 1
	}
	return 0
}

// UpdateDEBUG advances packed rocks through every generated sprite frame.
//
// it does not update positions, but requires them for behavior
func UpdateDEBUG(rocks *Rocks, in DebugInput) {
	if len(rocks.Positions) != len(rocks.Sprites) {
		panic("positions and sprites must have equal lengths")
	}

	if in.IncrementAmountScale {
		rocks.Atlas.AmountScale++
		if rocks.Atlas.AmountScale >= len(rocks.Atlas.Scales) {
			rocks.Atlas.AmountScale = 0
		}
	} else if in.DecrementAmountScale {
		rocks.Atlas.AmountScale--
		if rocks.Atlas.AmountScale < 0 {
			rocks.Atlas.AmountScale = len(rocks.Atlas.Scales) - 1
		}
	}
	increaseSize := simd.BroadcastUint32s(inputValue(in.IncreaseSizeScale))
	decreaseSize := simd.BroadcastUint32s(inputValue(in.DecrementSizeScale))

	for startIndex := 0; startIndex < len(rocks.Positions); startIndex += SIMDVectorSize {
		sprite, numRocksLoaded := simd.LoadUint32sPart(rocks.Sprites[startIndex:])
		size, rotation,
			rotationStepsX, rotationStepsY, animationTick,
			slopeX, slopeY := unpackSprites(sprite)

		rotation = rotation.Add(oneCoordinates)
		rotationWrapped := rotation.GreaterEqual(rotationFrameCount)
		rotation = zeroCoordinates.IfElse(rotationWrapped, rotation)

		xWrapped := rotationWrapped.And(slopeX.Equal(maximumSlope))
		slopeX = incrementSpriteSlope(slopeX).IfElse(rotationWrapped, slopeX)
		slopeY = incrementSpriteSlope(slopeY).IfElse(xWrapped, slopeY)

		size = size.Add(increaseSize).Sub(decreaseSize)
		size = oneCoordinates.IfElse(size.Greater(maximumRockSize), size)
		size = maximumRockSize.IfElse(size.Equal(zeroCoordinates), size)

		packSprites(
			size, rotation,
			rotationStepsX, rotationStepsY, animationTick,
			slopeX, slopeY,
		).StorePart(rocks.Sprites[startIndex : startIndex+numRocksLoaded])

	}
}
