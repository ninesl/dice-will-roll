package rocks

import (
	"simd"

	"github.com/ninesl/dice-will-roll/settings"
)

// RockPositions stores packed uint32 values with layout [VX:4][VY:4][X:12][Y:12].
type RockPositions []uint32

// RockSprites stores packed visual slopes, size score, and finite transition state.
type RockSprites []uint32

type InputSprite struct {
	SlopeX, SlopeY    int32
	SlopeZ, SizeScore uint32
	StepY, StepX      uint32
	StepZ, StepTick   uint32
	PermaSpin         uint32
	SpinAgain         uint32
}

const (
	packedVelocityXMask   uint32 = 0xF000_0000
	packedVelocityYMask   uint32 = 0x0F00_0000
	packedSlopeXMask      uint32 = 0xF000_0000
	packedSlopeYMask      uint32 = 0x0F00_0000
	packedPositionXMask   uint32 = 0x00FF_F000
	packedPositionYMask   uint32 = 0x0000_0FFF
	spriteSlopeZMask32    uint32 = 0x00F0_0000
	spriteSizeScoreMask32 uint32 = 0x000F_0000
	stepYMask32           uint32 = 0x0000_E000
	stepXMask32           uint32 = 0x0000_1C00
	stepZMask32           uint32 = 0x0000_03C0
	stepTickMask32        uint32 = 0x0000_003C
	permaSpinMask32       uint32 = 0x0000_0002
	spinAgainMask32       uint32 = 0x0000_0001
	permaXDirectionMask32 uint32 = 0x0000_0003
	permaYDirectionMask32 uint32 = 0x0000_000C
	permaXIncrement32     uint32 = 0x0000_0001
	permaXDecrement32     uint32 = 0x0000_0002
	permaYIncrement32     uint32 = 0x0000_0004
	permaYDecrement32     uint32 = 0x0000_0008
)

type Rocks struct {
	Positions   RockPositions
	Sprites     RockSprites
	Atlas       *RockSpriteAtlas
	AmountScale int
}

var (
	RocksPerPackedVector int

	// PRECOMPUTED REUSED CONSTANTS

	velocityXMask, velocityYMask             simd.Uint32s
	slopeXMask, slopeYMask                   simd.Uint32s
	positionXMask, positionYMask             simd.Uint32s
	spriteSizeScoreMask, spriteSlopeZMask    simd.Uint32s
	stepXMask, stepYMask                     simd.Uint32s
	stepZMask, stepTickMask                  simd.Uint32s
	permaSpinMask                            simd.Uint32s
	spinAgainMask                            simd.Uint32s
	permaXDirectionMask, permaYDirectionMask simd.Uint32s
	permaXIncrement, permaXDecrement         simd.Uint32s
	permaYIncrement, permaYDecrement         simd.Uint32s
	oneCoordinates, zeroCoordinates          simd.Uint32s
	maximumSizeScore, maximumStepZ           simd.Uint32s
	sizeScoreLanes                           [BitSpriteSlopeCodeCount]simd.Uint32s
	collisionLookups                         [len(rockAmountScales)][BitSpriteSlopeCodeCount]simd.Uint32s
	hoverRadiusLookups                       [len(rockAmountScales)][BitSpriteSlopeCodeCount]simd.Uint32s
	mouseRadii                               [len(rockAmountScales)]simd.Uint32s
	slopeZFrameCount                         simd.Uint32s
	maximumSlopeZ                            simd.Uint32s
	screenWidth, screenHeight                simd.Uint32s
	maximumOverlap                           simd.Uint32s
	zeroSlope, oneSlope                      simd.Int32s
	minimumSlope, maximumSlope               simd.Int32s
	slopeOffset, slopeCycle, halfSlopeCycle  simd.Int32s
)

func init() {
	// Go 1.27 documents BUG(global initialization): "SIMD-dependent var
	// initializers don't work." Keep this call out of the var initializer.
	// https://github.com/golang/go/commit/c83d55f69519f8cc633215287142f394d3b66087#diff-81ad425e4481409eeb696055ddac6751598ac9be712c00ae47324cd0aa3db38d
	RocksPerPackedVector = simd.Uint32s{}.Len()

	// Position: [ VX:4 ][ VY:4 ][ X:12 ][ Y:12 ]
	velocityXMask = simd.BroadcastUint32s(packedVelocityXMask)
	velocityYMask = simd.BroadcastUint32s(packedVelocityYMask)
	positionXMask = simd.BroadcastUint32s(packedPositionXMask)
	positionYMask = simd.BroadcastUint32s(packedPositionYMask)

	// Sprite: [ SlopeX:4 ][ SlopeY:4 ][ SlopeZ:4 ][ SizeScore:4 ][ StepY:3 ][ StepX:3 ][ StepZ:4 ][ StepTick:4 ][ PermaSpin:1 ][ SpinAgain:1 ]
	slopeXMask = simd.BroadcastUint32s(packedSlopeXMask)
	slopeYMask = simd.BroadcastUint32s(packedSlopeYMask)
	spriteSlopeZMask = simd.BroadcastUint32s(spriteSlopeZMask32)
	spriteSizeScoreMask = simd.BroadcastUint32s(spriteSizeScoreMask32)
	stepXMask = simd.BroadcastUint32s(stepXMask32)
	stepYMask = simd.BroadcastUint32s(stepYMask32)
	stepZMask = simd.BroadcastUint32s(stepZMask32)
	stepTickMask = simd.BroadcastUint32s(stepTickMask32)
	permaSpinMask = simd.BroadcastUint32s(permaSpinMask32)
	spinAgainMask = simd.BroadcastUint32s(spinAgainMask32)
	permaXDirectionMask = simd.BroadcastUint32s(permaXDirectionMask32)
	permaYDirectionMask = simd.BroadcastUint32s(permaYDirectionMask32)
	permaXIncrement = simd.BroadcastUint32s(permaXIncrement32)
	permaXDecrement = simd.BroadcastUint32s(permaXDecrement32)
	permaYIncrement = simd.BroadcastUint32s(permaYIncrement32)
	permaYDecrement = simd.BroadcastUint32s(permaYDecrement32)

	zeroCoordinates = simd.BroadcastUint32s(0)
	oneCoordinates = simd.BroadcastUint32s(1)
	maximumSizeScore = simd.BroadcastUint32s(BitSpriteSlopeCodeCount - 1)
	maximumStepZ = simd.BroadcastUint32s(0xF)
	for sizeScore := range sizeScoreLanes {
		sizeScoreLanes[sizeScore] = simd.BroadcastUint32s(uint32(sizeScore))
	}
	slopeZFrameCount = simd.BroadcastUint32s(AtlasSlopeZFrames)
	maximumSlopeZ = simd.BroadcastUint32s(AtlasSlopeZFrames - 1)
	maximumOverlap = simd.BroadcastUint32s(0xFFFF_FFFF)
	zeroSlope = simd.BroadcastInt32s(0)
	oneSlope = simd.BroadcastInt32s(1)
	minimumSlope = simd.BroadcastInt32s(-7)
	maximumSlope = simd.BroadcastInt32s(7)
	slopeOffset = simd.BroadcastInt32s(8)
	slopeCycle = simd.BroadcastInt32s(atlasSlopeStates)
	halfSlopeCycle = simd.BroadcastInt32s(7)
	initByteFieldSIMD()
}

// Init precomputes screen-dependent SIMD values after settings initializes the
// monitor resolution.
func Init(screen settings.ScreenSettings) {
	screenWidth = simd.BroadcastUint32s(uint32(screen.ResolutionX))
	screenHeight = simd.BroadcastUint32s(uint32(screen.ResolutionY))
}

// Velocity and slope codes use bias 8: codes 1..15 represent -7..7, while
// code 0 is reserved and decodes to zero for empty SIMD lanes.
