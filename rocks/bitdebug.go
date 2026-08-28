package rocks

import (
	"simd"

	"github.com/hajimehoshi/ebiten/v2"
)

type RockDebug struct {
	Filter                       string
	MipmapsEnabled               bool
	FPS, TPS                     float64
	Scale, DrawSize              float64
	AmountScaleMultiplier        float64
	SpriteSheetMB                float64
	PositionsKB, SpritesKB       float64
	TotalFrames, VisitedFrames   int
	AmountScale                  int
	CollisionRadius              int
	SizeScore, SlopeZ            int
	StepX, StepY, StepZ          int
	StepTick                     int
	PermaSpin                    int
	SpinAgain                    int
	SlopeX, SlopeY               int
	VelocityX, VelocityY         int
	PackedPosition, PackedSprite uint32
}

func (rocks Rocks) DebugSnapshot() *RockDebug {
	packedPosition := rocks.Positions[0]
	packedSprite := rocks.Sprites[0]
	_, _, velocityX, velocityY := UnpackPosition(packedPosition)
	sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY := UnpackSprite(packedSprite)
	scale := rocks.Atlas.Scales[rocks.AmountScale][sizeScore]
	bounds := rocks.Atlas.Image.Bounds()

	return &RockDebug{
		Filter:                FilterName(rocks.DrawOptions.Filter),
		MipmapsEnabled:        !rocks.DrawOptions.DisableMipmaps,
		FPS:                   ebiten.ActualFPS(),
		TPS:                   ebiten.ActualTPS(),
		Scale:                 scale,
		DrawSize:              float64(rocks.Atlas.SpriteSheet.TileSize) * scale,
		AmountScale:           rocks.AmountScale,
		AmountScaleMultiplier: rockAmountScales[rocks.AmountScale],
		CollisionRadius:       int(rocks.Atlas.CollisionLookups[rocks.AmountScale][sizeScore]),
		SpriteSheetMB:         float64(bounds.Dx()*bounds.Dy()*4) / (1024 * 1024),
		PositionsKB:           float64(len(rocks.Positions)*4) / 1024,
		SpritesKB:             float64(len(rocks.Sprites)*4) / 1024,
		TotalFrames:           atlasSlopeStates * atlasSlopeStates * AtlasSlopeZFrames,
		VisitedFrames:         filterIndex(slopeX, slopeY, slopeZ) + 1,
		SizeScore:             sizeScore,
		SlopeZ:                slopeZ,
		StepX:                 stepX,
		StepY:                 stepY,
		StepZ:                 stepZ,
		StepTick:              stepTick,
		PermaSpin:             permaSpin,
		SpinAgain:             spinAgain,
		SlopeX:                slopeX,
		SlopeY:                slopeY,
		VelocityX:             velocityX,
		VelocityY:             velocityY,
		PackedPosition:        packedPosition,
		PackedSprite:          packedSprite,
	}
}

type DebugInput struct {
	IncreaseSizeScale    bool
	DecrementSizeScale   bool
	IncrementAmountScale bool
	DecrementAmountScale bool
	CycleFilter          bool
	ToggleMipmaps        bool
}

// ApplyDebugInput changes debug-controlled scale settings without advancing the simulation.
func ApplyDebugInput(rocks *Rocks, in DebugInput) {
	drawOptions := rocks.DrawOptions
	if in.CycleFilter {
		switch drawOptions.Filter {
		case ebiten.FilterPixelated:
			drawOptions.Filter = ebiten.FilterNearest
		case ebiten.FilterNearest:
			drawOptions.Filter = ebiten.FilterLinear
		default:
			drawOptions.Filter = ebiten.FilterPixelated
		}
		if drawOptions.Filter != ebiten.FilterLinear {
			drawOptions.DisableMipmaps = true
		}
	}
	if in.ToggleMipmaps {
		if drawOptions.DisableMipmaps {
			drawOptions.DisableMipmaps = false
			drawOptions.Filter = ebiten.FilterLinear
		} else {
			drawOptions.DisableMipmaps = true
		}
	}

	if in.IncrementAmountScale {
		rocks.AmountScale++
		if rocks.AmountScale >= len(rocks.Atlas.Scales) {
			rocks.AmountScale = 0
		}
	} else if in.DecrementAmountScale {
		rocks.AmountScale--
		if rocks.AmountScale < 0 {
			rocks.AmountScale = len(rocks.Atlas.Scales) - 1
		}
	}

	if !in.IncreaseSizeScale && !in.DecrementSizeScale {
		return
	}

	increaseSize := simd.BroadcastUint32s(inputValue(in.IncreaseSizeScale))
	decreaseSize := simd.BroadcastUint32s(inputValue(in.DecrementSizeScale))
	for startIndex := 0; startIndex < len(rocks.Sprites); startIndex += SIMDVectorSize {
		sprite, numRocksLoaded := simd.LoadUint32sPart(rocks.Sprites[startIndex:])
		sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY := unpackSpritesSIMD(sprite)

		sizeScore = sizeScore.Add(increaseSize).Sub(decreaseSize)
		sizeScore = oneCoordinates.IfElse(sizeScore.Greater(maximumSizeScore), sizeScore)
		sizeScore = maximumSizeScore.IfElse(sizeScore.Equal(zeroCoordinates), sizeScore)

		packSpritesSIMD(
			sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY,
		).StorePart(rocks.Sprites[startIndex : startIndex+numRocksLoaded])
	}
}

func inputValue(active bool) uint32 {
	if active {
		return 1
	}
	return 0
}

// UpdateDEBUG advances packed rocks through every generated sprite frame.
func UpdateDEBUG(rocks *Rocks, in DebugInput) {
	if len(rocks.Positions) != len(rocks.Sprites) {
		panic("positions and sprites must have equal lengths")
	}

	ApplyDebugInput(rocks, in)

	for startIndex := 0; startIndex < len(rocks.Positions); startIndex += SIMDVectorSize {
		sprite, numRocksLoaded := simd.LoadUint32sPart(rocks.Sprites[startIndex:])
		sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY := unpackSpritesSIMD(sprite)

		slopeZ = slopeZ.Add(oneCoordinates)
		slopeZWrapped := slopeZ.GreaterEqual(slopeZFrameCount)
		slopeZ = zeroCoordinates.IfElse(slopeZWrapped, slopeZ)

		xWrapped := slopeZWrapped.And(slopeX.Equal(maximumSlope))
		slopeX = incrementSlope(slopeX).IfElse(slopeZWrapped, slopeX)
		slopeY = incrementSlope(slopeY).IfElse(xWrapped, slopeY)

		packSpritesSIMD(
			sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY,
		).StorePart(rocks.Sprites[startIndex : startIndex+numRocksLoaded])
	}

	UpdateRocks(rocks, 1, 0, nil)
}
