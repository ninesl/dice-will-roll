package rocks

import (
	"fmt"
	"math"
	"simd"

	"github.com/ninesl/dice-will-roll/settings"
)

const (
	packedCoordinateMask  uint16 = 0x0FFF
	spriteSizeScoreMask16 uint16 = 0x000F
	forceSteppingMask16   uint16 = 0x0002
	forceStepXMask32      uint32 = 0x0000_0003
	forceStepYMask32      uint32 = 0x0000_000C

	// These shifts are deliberately fixed for the packed size range 1..15.
	// Startup fitting verifies that the geometry-derived radii remain exact and
	// that every multiply/add intermediate fits in a uint16 lane.
	collisionRadiusShift = 5
	hoverRadiusShift     = 6
	mouseForceBands      = 7
)

type Rocks struct {
	PosX     []uint16
	PosY     []uint16
	Slope    []uint16
	Stepping []uint16

	drawGroups     []bool
	dirtyLayers    []bool
	groupsPerLayer int
	anyDirtyLayer  bool
	redraw         bool
	updateState    rockUpdateState
}

type rockUpdateRoutine uint8

const (
	rockUpdateDisabled rockUpdateRoutine = iota
	rockUpdateHover
	rockUpdateButton
)

type rockUpdateState struct {
	initialized bool
	amountScale int
	routine     rockUpdateRoutine
	positionX   int16
	positionY   int16
	mouseActive bool
	leftDown    bool
	rightDown   bool
	rockCount   int
}

type rockUpdateInput16 struct {
	constants *rockUpdateConstants16
	routine   rockUpdateRoutine
	mouseX    simd.Int16s
	mouseY    simd.Int16s
	farForce  simd.Mask16s
	attract   simd.Mask16s
}

type rockUpdateConstants16 struct {
	collisionMultiplier simd.Uint16s
	collisionBias       simd.Uint16s
	hoverMultiplier     simd.Uint16s
	hoverBias           simd.Uint16s
	mouseRadius         simd.Uint16s
	// Six explicit transitions implement the seven fixed force magnitudes.
	// Changing that range requires updating initialization, the unrolled force
	// routine, and its equivalence tests together.
	buttonThresholds [mouseForceBands - 1]simd.Uint16s
}

func (rocks Rocks) Len() int { return len(rocks.PosX) }

var (
	RocksPerPackedVector int

	zeroUint16, oneUint16, maximumSizeUint16, coordinateMask16, velocityMask16, nibbleMask16 simd.Uint16s
	maximumStepZ16, slopeCycleU16                                                            simd.Uint16s

	zeroInt16, oneInt16, minimumSlopeI16, maximumSlopeI16 simd.Int16s
	slopeCycleI16, halfSlopeCycleI16                      simd.Int16s
	nibbleValues16                                        [BitSpriteSlopeCodeCount + 1]simd.Uint16s

	screenWidth16, screenHeight16 simd.Int16s

	rockUpdateConstants [len(rockAmountScales)]rockUpdateConstants16
)

func init() {
	RocksPerPackedVector = simd.Uint16s{}.Len()
	zeroUint16 = simd.BroadcastUint16s(0)
	oneUint16 = simd.BroadcastUint16s(1)
	maximumSizeUint16 = simd.BroadcastUint16s(BitSpriteSlopeCodeCount - 1)
	coordinateMask16 = simd.BroadcastUint16s(packedCoordinateMask)
	velocityMask16 = simd.BroadcastUint16s(0xF000)
	nibbleMask16 = simd.BroadcastUint16s(0xF)
	maximumStepZ16 = simd.BroadcastUint16s(0xF)
	slopeCycleU16 = simd.BroadcastUint16s(atlasSlopeStates)
	zeroInt16 = simd.BroadcastInt16s(0)
	oneInt16 = simd.BroadcastInt16s(1)
	minimumSlopeI16 = simd.BroadcastInt16s(-7)
	maximumSlopeI16 = simd.BroadcastInt16s(7)
	slopeCycleI16 = simd.BroadcastInt16s(atlasSlopeStates)
	halfSlopeCycleI16 = simd.BroadcastInt16s(7)
	for value := range nibbleValues16 {
		nibbleValues16[value] = simd.BroadcastUint16s(uint16(value))
	}
	initializeRockUpdateConstants(PackedRockSpritePixels)
}

func initializeRockUpdateConstants(pixelSize int) {
	for amountScale, scale := range rockAmountScales {
		var hoverRadii, collisionRadii [BitSpriteSlopeCodeCount]uint16
		for size := 1; size < BitSpriteSlopeCodeCount; size++ {
			halfDrawSize := float32(pixelSize) * rockSizeScale(size) * scale / 2
			hoverRadii[size] = uint16(math.Ceil(float64(halfDrawSize)))
			collisionRadii[size] = uint16(math.Ceil(float64(halfDrawSize * 2.0 / 3.0)))
		}

		collisionMultiplier, collisionBias := fitRadiusFormula(collisionRadii, collisionRadiusShift)
		hoverMultiplier, hoverBias := fitRadiusFormula(hoverRadii, hoverRadiusShift)
		constants := &rockUpdateConstants[amountScale]
		constants.collisionMultiplier = simd.BroadcastUint16s(collisionMultiplier)
		constants.collisionBias = simd.BroadcastUint16s(collisionBias)
		constants.hoverMultiplier = simd.BroadcastUint16s(hoverMultiplier)
		constants.hoverBias = simd.BroadcastUint16s(hoverBias)
		mouseRadius := hoverRadii[BitSpriteSlopeCodeCount-1]
		constants.mouseRadius = simd.BroadcastUint16s(mouseRadius)
		for band := 1; band < mouseForceBands; band++ {
			threshold := (mouseRadius*uint16(band) + mouseForceBands - 1) / mouseForceBands
			constants.buttonThresholds[band-1] = simd.BroadcastUint16s(threshold)
		}
	}
}

func fitRadiusFormula(radii [BitSpriteSlopeCodeCount]uint16, shift uint) (uint16, uint16) {
	unit := 1 << shift
	for multiplier := 0; multiplier <= math.MaxUint16/(BitSpriteSlopeCodeCount-1); multiplier++ {
		minimumBias, maximumBias := 0, math.MaxUint16
		for size := 1; size < BitSpriteSlopeCodeCount; size++ {
			minimumBias = max(minimumBias, int(radii[size])*unit-size*multiplier)
			maximumBias = min(maximumBias, (int(radii[size])+1)*unit-1-size*multiplier)
		}
		if minimumBias <= maximumBias && minimumBias >= 0 &&
			(BitSpriteSlopeCodeCount-1)*multiplier+minimumBias <= math.MaxUint16 {
			return uint16(multiplier), uint16(minimumBias)
		}
	}
	panic(fmt.Sprintf("rock radii do not fit exact uint16 affine formula with shift %d: %v", shift, radii))
}

func Init(screen settings.ScreenSettings) {
	if screen.ResolutionX > int(packedCoordinateMask) || screen.ResolutionY > int(packedCoordinateMask) {
		panic(fmt.Sprintf("rock screen resolution exceeds 12-bit coordinates: %dx%d",
			screen.ResolutionX, screen.ResolutionY))
	}
	screenWidth16 = simd.BroadcastInt16s(int16(screen.ResolutionX))
	screenHeight16 = simd.BroadcastInt16s(int16(screen.ResolutionY))
}
