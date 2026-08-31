package rocks

import (
	"simd"

	"github.com/hajimehoshi/ebiten/v2"
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

func drawRockRange(
	rocks *Rocks,
	atlas *RockSpriteAtlas,
	amountScale int,
	target *ebiten.Image,
	drawOptions ebiten.DrawImageOptions,
	start, end int,
) {
	for i := start; i < end; i++ {
		packedX := rocks.PosX[i]
		positionX, positionY, _, _ := UnpackPosition(packedX, rocks.PosY[i])
		slope := rocks.Slope[i]
		sizeScore := int(slope & spriteSizeScoreMask16)
		slopeZ := int(slope >> 4 & 0xF)
		slopeX := unpackSignedNibble(slope, 12)
		slopeY := unpackSignedNibble(slope, 8)
		scale := atlas.Scales[amountScale][sizeScore]
		halfDrawSize := atlas.HalfDrawSizes[amountScale][sizeScore]

		drawOptions.GeoM.Reset()
		drawOptions.GeoM.Scale(float64(scale), float64(scale))
		drawOptions.GeoM.Translate(
			float64(positionX)-float64(halfDrawSize),
			float64(positionY)-float64(halfDrawSize),
		)
		target.DrawImage(atlas.Frames[filterIndex(slopeX, slopeY, slopeZ)], &drawOptions)
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
	ForceStepping                int
	Stepping                     int
	SlopeX, SlopeY               int
	VelocityX, VelocityY         int
	PackedPosition, PackedSprite uint32
}

func DebugSnapshot(
	rocks Rocks,
	atlas *RockSpriteAtlas,
	amountScale int,
	drawOptions ebiten.DrawImageOptions,
) *RockDebug {
	minScale := atlas.Scales[amountScale][1]
	maxScale := atlas.Scales[amountScale][BitSpriteSlopeCodeCount-1]
	tileSize := float64(atlas.SpriteSheet.TileSize)
	bounds := atlas.Image.Bounds()

	debug := &RockDebug{
		Filter:                FilterName(drawOptions.Filter),
		MipmapsEnabled:        !drawOptions.DisableMipmaps,
		FPS:                   ebiten.ActualFPS(),
		TPS:                   ebiten.ActualTPS(),
		MinDrawSize:           tileSize * float64(minScale),
		MaxDrawSize:           tileSize * float64(maxScale),
		AmountScale:           amountScale,
		AmountScaleMultiplier: float64(rockAmountScales[amountScale]),
		SpriteSheetMB:         float64(bounds.Dx()*bounds.Dy()*4) / (1024 * 1024),
		PositionsKB:           float64((len(rocks.PosX)+len(rocks.PosY))*2) / 1024,
		SpritesKB:             float64((len(rocks.Slope)+len(rocks.Stepping))*2) / 1024,
		TotalFrames:           atlasSlopeStates * atlasSlopeStates * AtlasSlopeZFrames,
	}
	if rocks.Len() == 0 {
		return debug
	}

	debug.PackedPosition = uint32(rocks.PosX[0])<<16 | uint32(rocks.PosY[0])
	debug.PackedSprite = uint32(rocks.Slope[0])<<16 | uint32(rocks.Stepping[0])
	_, _, debug.VelocityX, debug.VelocityY = UnpackPosition(rocks.PosX[0], rocks.PosY[0])
	debug.SizeScore, debug.SlopeZ, debug.StepX, debug.StepY, debug.StepZ, debug.StepTick,
		debug.ForceStepping, debug.Stepping, debug.SlopeX, debug.SlopeY =
		UnpackSprite(rocks.Slope[0], rocks.Stepping[0])
	scale := atlas.Scales[amountScale][debug.SizeScore]
	debug.Scale = float64(scale)
	debug.DrawSize = tileSize * float64(scale)
	debug.CollisionRadius = int(atlas.CollisionLookups[amountScale][debug.SizeScore])
	debug.VisitedFrames = filterIndex(debug.SlopeX, debug.SlopeY, debug.SlopeZ) + 1
	return debug
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
func ApplyDebugInput(rocks *Rocks, drawOptions ebiten.DrawImageOptions, in DebugInput) ebiten.DrawImageOptions {
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

	if !in.IncreaseSizeScale && !in.DecrementSizeScale {
		return drawOptions
	}
	rocks.redraw = true

	increaseSize, decreaseSize := zeroUint16, zeroUint16
	if in.IncreaseSizeScale {
		increaseSize = oneUint16
	}
	if in.DecrementSizeScale {
		decreaseSize = oneUint16
	}
	for startIndex := 0; startIndex < len(rocks.Slope); startIndex += RocksPerPackedVector {
		slope, numRocksLoaded := simd.LoadUint16sPart(rocks.Slope[startIndex:])
		sizeScore := slope.And(nibbleMask16)
		sizeScore = sizeScore.Add(increaseSize).Sub(decreaseSize)
		sizeScore = oneUint16.IfElse(sizeScore.Greater(maximumSizeUint16), sizeScore)
		sizeScore = maximumSizeUint16.IfElse(sizeScore.Equal(zeroUint16), sizeScore)
		slope.AndNot(nibbleMask16).Or(sizeScore).
			StorePart(rocks.Slope[startIndex : startIndex+numRocksLoaded])
	}
	return drawOptions
}
