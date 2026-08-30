package rocks

import (
	"simd"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/controls"
)

func FilterName(filter ebiten.Filter) string {
	switch filter {
	case ebiten.FilterNearest:
		return "Nearest"
	case ebiten.FilterLinear:
		return "Linear"
	case ebiten.FilterPixelated:
		return "Pixelated"
	default:
		return "Unknown"
	}
}

// Draw renders every packed rock in slice order using atlas.
func Draw(rocks Rocks, screen *ebiten.Image, drawOptions ebiten.DrawImageOptions) {
	for i, pos := range rocks.Positions {
		positionX, positionY, _, _ := UnpackPosition(pos)
		sizeScore, slopeZ, _, _, _, _, _, _, slopeX, slopeY := UnpackSprite(rocks.Sprites[i])
		scale := rocks.Atlas.Scales[rocks.AmountScale][sizeScore]
		halfDrawSize := rocks.Atlas.HalfDrawSizes[rocks.AmountScale][sizeScore]

		drawOptions.GeoM.Reset()
		drawOptions.GeoM.Scale(float64(scale), float64(scale))
		drawOptions.GeoM.Translate(
			float64(positionX)-float64(halfDrawSize),
			float64(positionY)-float64(halfDrawSize),
		)
		screen.DrawImage(rocks.Atlas.Frames[filterIndex(slopeX, slopeY, slopeZ)], &drawOptions)
	}
}

type RockDebug struct {
	Filter                       string
	MipmapsEnabled               bool
	FPS, TPS                     float64
	Scale, DrawSize              float64
	MinDrawSize, MaxDrawSize     float64
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

func (rocks Rocks) DebugSnapshot(drawOptions ebiten.DrawImageOptions) *RockDebug {
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
	minScale := rocks.Atlas.Scales[rocks.AmountScale][1]
	maxScale := rocks.Atlas.Scales[rocks.AmountScale][BitSpriteSlopeCodeCount-1]
	tileSize := float64(rocks.Atlas.SpriteSheet.TileSize)
	bounds := rocks.Atlas.Image.Bounds()

	return &RockDebug{
		Filter:                FilterName(drawOptions.Filter),
		MipmapsEnabled:        !drawOptions.DisableMipmaps,
		FPS:                   ebiten.ActualFPS(),
		TPS:                   ebiten.ActualTPS(),
		Scale:                 float64(scale),
		DrawSize:              tileSize * float64(scale),
		MinDrawSize:           tileSize * float64(minScale),
		MaxDrawSize:           tileSize * float64(maxScale),
		AmountScale:           rocks.AmountScale,
		AmountScaleMultiplier: float64(rockAmountScales[rocks.AmountScale]),
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
func ApplyDebugInput(rocks Rocks, drawOptions ebiten.DrawImageOptions, in DebugInput) (Rocks, ebiten.DrawImageOptions) {
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
		return rocks, drawOptions
	}

	increaseSize := simd.BroadcastUint32s(inputValue(in.IncreaseSizeScale))
	decreaseSize := simd.BroadcastUint32s(inputValue(in.DecrementSizeScale))
	for startIndex := 0; startIndex < len(rocks.Sprites); startIndex += RocksPerPackedVector {
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
	return rocks, drawOptions
}

func inputValue(active bool) uint32 {
	if active {
		return 1
	}
	return 0
}

// UpdateDEBUG advances packed rocks through every generated sprite frame.
func UpdateDEBUG(rocks Rocks, drawOptions ebiten.DrawImageOptions, in DebugInput) (Rocks, ebiten.DrawImageOptions) {
	rocks, drawOptions = ApplyDebugInput(rocks, drawOptions, in)

	for startIndex := 0; startIndex < len(rocks.Positions); startIndex += RocksPerPackedVector {
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

	for phase := range UpdateStride {
		UpdateRocks(rocks, phase, controls.MouseInfo{})
	}
	return rocks, drawOptions
}
