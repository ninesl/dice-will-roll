package rocks

import (
	"simd"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/controls"
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
	LiveSprites RockSprites
	Atlas       *RockSpriteAtlas
	DrawOptions *ebiten.DrawImageOptions
	AmountScale int

	scheduledPositions RockPositions
	scheduledSprites   RockSprites
	scheduledIndices   []int
}

var (
	SIMDVectorSize int

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
	hoverRadiusLookups                       [len(rockAmountScales)][BitSpriteSlopeCodeCount]simd.Float32s
	mouseRadius                              simd.Uint32s
	slopeZFrameCount                         simd.Uint32s
	maximumSlopeZ                            simd.Uint32s
	screenWidth, screenHeight                simd.Uint32s
	maximumPositionCoordinate                simd.Uint32s
	maximumOverlap                           simd.Uint32s
	zeroSlope, oneSlope                      simd.Int32s
	minimumSlope, maximumSlope               simd.Int32s
	slopeOffset, slopeCycle, halfSlopeCycle  simd.Int32s
	zeroFloat                                simd.Float32s
	mouseVelocityScale, mouseVelocityFalloff simd.Float32s
)

func init() {
	// Go 1.27 documents BUG(global initialization): "SIMD-dependent var
	// initializers don't work." Keep this call out of the var initializer.
	// https://github.com/golang/go/commit/c83d55f69519f8cc633215287142f394d3b66087#diff-81ad425e4481409eeb696055ddac6751598ac9be712c00ae47324cd0aa3db38d
	SIMDVectorSize = simd.Uint32s{}.Len()

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
	maximumPositionCoordinate = simd.BroadcastUint32s(packedPositionYMask)
	zeroSlope = simd.BroadcastInt32s(0)
	oneSlope = simd.BroadcastInt32s(1)
	minimumSlope = simd.BroadcastInt32s(-7)
	maximumSlope = simd.BroadcastInt32s(7)
	slopeOffset = simd.BroadcastInt32s(8)
	slopeCycle = simd.BroadcastInt32s(atlasSlopeStates)
	halfSlopeCycle = simd.BroadcastInt32s(7)
	zeroFloat = simd.BroadcastFloat32s(0)
	mouseVelocityScale = simd.BroadcastFloat32s(float32(BitSpriteSlopeCodeCount/2 - 1))
	mouseVelocityFalloff = simd.BroadcastFloat32s(float32(BitSpriteSlopeCodeCount/2 - 2))
}

// Init precomputes screen-dependent SIMD values after settings initializes the
// monitor resolution.
func Init(screen settings.ScreenSettings) {
	screenWidth = simd.BroadcastUint32s(uint32(screen.ResolutionX))
	screenHeight = simd.BroadcastUint32s(uint32(screen.ResolutionY))
}

// Velocity and slope codes use bias 8: codes 1..15 represent -7..7, while
// code 0 is reserved and decodes to zero for empty SIMD lanes.
func unpackVelocitiesSIMD(packed simd.Uint32s) (velocityX, velocityY simd.Int32s) {
	velocityXCode := packed.And(velocityXMask).ShiftAllRight(28)
	velocityYCode := packed.And(velocityYMask).ShiftAllRight(24)
	velocityX = velocityXCode.ConvertToInt32().Sub(slopeOffset).
		Masked(velocityXCode.NotEqual(zeroCoordinates))
	velocityY = velocityYCode.ConvertToInt32().Sub(slopeOffset).
		Masked(velocityYCode.NotEqual(zeroCoordinates))
	return velocityX, velocityY
}

func unpackSlopesSIMD(packed simd.Uint32s) (slopeX, slopeY simd.Int32s) {
	slopeXCode := packed.And(slopeXMask).ShiftAllRight(28)
	slopeYCode := packed.And(slopeYMask).ShiftAllRight(24)
	slopeX = slopeXCode.ConvertToInt32().Sub(slopeOffset).
		Masked(slopeXCode.NotEqual(zeroCoordinates))
	slopeY = slopeYCode.ConvertToInt32().Sub(slopeOffset).
		Masked(slopeYCode.NotEqual(zeroCoordinates))
	return slopeX, slopeY
}

func unpackPositionsSIMD(packedPositions simd.Uint32s) (
	positionX, positionY simd.Uint32s,
	velocityX, velocityY simd.Int32s,
) {
	// [ VX:4 ][ VY:4 ][ X:12 ][ Y:12 ]
	//    31..28   27..24   23..12   11..0
	velocityX, velocityY = unpackVelocitiesSIMD(packedPositions)
	positionX = packedPositions.And(positionXMask).ShiftAllRight(12)
	positionY = packedPositions.And(positionYMask)
	return positionX, positionY, velocityX, velocityY
}

func packVelocitiesSIMD(velocityX, velocityY simd.Int32s) simd.Uint32s {
	velocityXBits := velocityX.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(28).And(velocityXMask)
	velocityYBits := velocityY.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(24).And(velocityYMask)
	return velocityXBits.Or(velocityYBits)
}

func packSlopesSIMD(slopeX, slopeY simd.Int32s) simd.Uint32s {
	slopeXBits := slopeX.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(28).And(slopeXMask)
	slopeYBits := slopeY.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(24).And(slopeYMask)
	return slopeXBits.Or(slopeYBits)
}

func packPositionsSIMD(positionX, positionY simd.Uint32s, velocityX, velocityY simd.Int32s) simd.Uint32s {
	positionXBits := positionX.ShiftAllLeft(12).And(positionXMask)
	positionYBits := positionY.And(positionYMask)
	return packVelocitiesSIMD(velocityX, velocityY).Or(positionXBits).Or(positionYBits)
}

// UnpackPosition decodes one packed position.
// velocityX: bits 31..28. Codes 0001..1111 encode -7..7; velocity 0 is
// encoded as 1000. The packer never emits reserved code 0000, which unpacking
// treats as zero for empty SIMD lanes.
// velocityY: bits 27..24. Uses the same encoding and reserved 0000 behavior.
// positionX: bits 23..12. Unsigned center coordinate 0..4095.
// positionY: bits 11..0. Unsigned center coordinate 0..4095.
func UnpackPosition(pos uint32) (positionX, positionY, velocityX, velocityY int) {
	velocityX, velocityY = unpackVelocity(pos)
	positionX = int((pos & packedPositionXMask) >> 12)
	positionY = int(pos & packedPositionYMask)
	return positionX, positionY, velocityX, velocityY
}

func unpackVelocity(packed uint32) (velocityX, velocityY int) {
	velocityXCode := (packed & packedVelocityXMask) >> 28
	velocityYCode := (packed & packedVelocityYMask) >> 24
	if velocityXCode != 0 {
		velocityX = int(velocityXCode) - 8
	}
	if velocityYCode != 0 {
		velocityY = int(velocityYCode) - 8
	}
	return velocityX, velocityY
}

func unpackSlope(packed uint32) (slopeX, slopeY int) {
	slopeXCode := (packed & packedSlopeXMask) >> 28
	slopeYCode := (packed & packedSlopeYMask) >> 24
	if slopeXCode != 0 {
		slopeX = int(slopeXCode) - 8
	}
	if slopeYCode != 0 {
		slopeY = int(slopeYCode) - 8
	}
	return slopeX, slopeY
}

func PackPosition(positionX, positionY uint32, velocityX, velocityY int32) uint32 {
	return packVelocity(velocityX, velocityY) |
		(positionX<<12)&packedPositionXMask |
		positionY&packedPositionYMask
}

func packVelocity(velocityX, velocityY int32) uint32 {
	velocityXCode := uint32(velocityX + 0b1000)
	velocityYCode := uint32(velocityY + 0b1000)
	return (velocityXCode<<28)&packedVelocityXMask |
		(velocityYCode<<24)&packedVelocityYMask
}

func packSlope(slopeX, slopeY int32) uint32 {
	slopeXCode := uint32(slopeX + 0b1000)
	slopeYCode := uint32(slopeY + 0b1000)
	return (slopeXCode<<28)&packedSlopeXMask |
		(slopeYCode<<24)&packedSlopeYMask
}

// [ SlopeX:4 ][ SlopeY:4 ][ SlopeZ:4 ][ SizeScore:4 ][ StepY:3 ][ StepX:3 ][ StepZ:4 ][ StepTick:4 ][ PermaSpin:1 ][ SpinAgain:1 ]
//
//	31..28          27..24          23..20             19..16        15..13      12..10      9..6        5..2          1             0
func unpackSpritesSIMD(sprite simd.Uint32s) (
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) {
	sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY = unpackCollisionSpritesSIMD(sprite)
	stepZ = canonicalPermaStepZ(stepZ).IfElse(permaSpin.NotEqual(zeroCoordinates), stepZ)
	spinAgain = spinAgain.Masked(permaSpin.Equal(zeroCoordinates))
	return sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY
}

func unpackCollisionSpritesSIMD(sprite simd.Uint32s) (
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) {
	slopeX, slopeY = unpackSlopesSIMD(sprite)
	slopeZ = sprite.And(spriteSlopeZMask).ShiftAllRight(20)
	sizeScore = sprite.And(spriteSizeScoreMask).ShiftAllRight(16)
	stepY = sprite.And(stepYMask).ShiftAllRight(13)
	stepX = sprite.And(stepXMask).ShiftAllRight(10)
	stepZ = sprite.And(stepZMask).ShiftAllRight(6)
	stepTick = sprite.And(stepTickMask).ShiftAllRight(2)
	permaSpin = sprite.And(permaSpinMask).ShiftAllRight(1)
	spinAgain = sprite.And(spinAgainMask)
	return sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY
}

func packSpritesSIMD(
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) simd.Uint32s {
	stepZ = canonicalPermaStepZ(stepZ).IfElse(permaSpin.NotEqual(zeroCoordinates), stepZ)
	spinAgain = spinAgain.Masked(permaSpin.Equal(zeroCoordinates))
	return packCollisionSpritesSIMD(
		sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY)
}

func packCollisionSpritesSIMD(
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) simd.Uint32s {
	slopeZBits := slopeZ.ShiftAllLeft(20).And(spriteSlopeZMask)
	sizeScoreBits := sizeScore.ShiftAllLeft(16).And(spriteSizeScoreMask)
	stepYBits := stepY.ShiftAllLeft(13).And(stepYMask)
	stepXBits := stepX.ShiftAllLeft(10).And(stepXMask)
	stepZBits := stepZ.ShiftAllLeft(6).And(stepZMask)
	stepTickBits := stepTick.ShiftAllLeft(2).And(stepTickMask)
	permaSpinBit := permaSpin.ShiftAllLeft(1).And(permaSpinMask)
	spinAgainBit := spinAgain.And(spinAgainMask)

	return packSlopesSIMD(slopeX, slopeY).
		Or(slopeZBits).
		Or(sizeScoreBits).
		Or(stepYBits).
		Or(stepXBits).
		Or(stepZBits).
		Or(stepTickBits).
		Or(permaSpinBit).
		Or(spinAgainBit)
}

// UnpackSprite decodes one packed sprite.
// slopeX: bits 31..28. Codes 0001..1111 encode visual slopes -7..7; slope 0
// is encoded as 1000. The packer never emits reserved code 0000, which
// unpacking treats as zero for empty SIMD lanes.
// slopeY: bits 27..24. Uses the same 0001..1111 encoding as slopeX;
// reserved code 0000 has the same empty-lane behavior.
// slopeZ: bits 23..20. All 16 values 0..15 are atlas frames spaced evenly
// around 360 degrees; updates wrap between 0 and 15.
// sizeScore: bits 19..16. Values 1..15 select size and set the StepTick
// interval; 0000 is reserved and does not advance StepTick.
// stepY/stepX: bits 15..10. Independent 0..7 shortest-path catch-up counts.
// stepZ: bits 9..6. Without PermaSpin, values hold finite Z updates and 15 is
// loaded after SpinAgain supplies the first full-spin frame.
// With PermaSpin, bits 1..0 encode X direction and bits 3..2 encode Y direction:
// 00 is inactive, 01 increments, 10 decrements, and 11 is reserved. Only the
// latest bounced direction remains active; simultaneous X/Y wall overlaps
// select the greater penetration, with X winning an exact tie.
// stepTick: bits 5..2. Counts from 0 toward sizeScore while any step counter or
// PermaSpin is active, then resets before packing; 1111 is never persisted.
// permaSpin: bit 1. A value of 1 advances slopeZ on every due StepTick and
// ignores finite Z state. Canonical permanent state is 10.
// spinAgain: bit 0. A direction change sets it to 1. After stepZ reaches 0,
// the next due update consumes it and starts one full 16-frame Z spin.
// Flag state 11 is normalized to 10 because SpinAgain is redundant when
// PermaSpin is active; 00 and 01 retain their distinct finite-spin meanings.
//
// velocityX/Y live in the packed position and move the rock. When either
// velocity changes, stepX/Y receive the shortest circular distance from the
// current visual slope to that velocity. Finite spin sets stepZ to the sum of
// the unsigned three-bit X/Y animation distances. Permanent spin stores the selected
// bounced X or Y direction and the inverse rotation direction in stepZ.
//
// While any step counter or PermaSpin is active, stepTick counts down from
// max(1, floor(sizeScore/2))-1. Zero performs the animation update. Each due
// update moves slopeX/Y one shortest-path step toward velocityX/Y and
// decrements StepX/Y. Finite Z updates decrement StepZ to 0. SpinAgain then
// provides one Z update, reloads StepZ to 15, and clears itself, producing one
// complete 16-frame revolution. PermaSpin clears SpinAgain, keeps advancing
// slopeZ, and overrides the selected X/Y update mask with the direction stored
// in StepZ so that X or Y continues wrapping after StepX/Y reaches zero.
// Before movement, state 11 normalizes to 10. Finite states 00 and 01 move each
// velocity one integer toward zero after movement; permanent state 10 skips
// damping. Once both
// damped velocities reach zero, all step counters and spin flags are cleared;
// the current visual orientation is retained without a settling animation.
//
// State audit: the X/Y code 0000 and SizeScore 0000 are reserved; StepTick 1111
// is transient and reset before packing. Finite StepZ and SlopeZ use all 16
// states, and StepX/Y use all eight. PermaSpin StepZ canonicalizes to 0, 1, 2,
// 4, or 8; reserved direction code 11 becomes inactive and simultaneous X/Y
// direction codes keep X only. Flag 11 is canonicalized to 10. Packed position
// coordinates use their entire 0..4095 ranges; coordinate 0 is not a sentinel.
func UnpackSprite(packedSprite uint32) (
	sizeScore, slopeZ,
	stepX, stepY,
	stepZ, stepTick,
	permaSpin,
	spinAgain,
	slopeX, slopeY int,
) {
	slopeX, slopeY = unpackSlope(packedSprite)
	sizeScore = int((packedSprite & spriteSizeScoreMask32) >> 16)
	slopeZ = int((packedSprite & spriteSlopeZMask32) >> 20)
	stepY = int((packedSprite & stepYMask32) >> 13)
	stepX = int((packedSprite & stepXMask32) >> 10)
	stepZ = int((packedSprite & stepZMask32) >> 6)
	stepTick = int((packedSprite & stepTickMask32) >> 2)
	permaSpin = int((packedSprite & permaSpinMask32) >> 1)
	spinAgain = int(packedSprite & spinAgainMask32)
	if permaSpin != 0x0 {
		stepZ = int(canonicalPermaStepZScalar(uint32(stepZ)))
		spinAgain = 0x0
	}
	return sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY
}

func canonicalPermaStepZScalar(stepZ uint32) uint32 {
	if stepZ == 0xF {
		return stepZ
	}
	xDirection := stepZ & permaXDirectionMask32
	yDirection := stepZ & permaYDirectionMask32
	if xDirection == permaXDirectionMask32 {
		xDirection = 0x0
	}
	if yDirection == permaYDirectionMask32 || xDirection != 0x0 {
		yDirection = 0x0
	}
	return xDirection | yDirection
}

func PackSprite(input InputSprite) uint32 {
	stepZ := input.StepZ & 0xF
	spinAgain := input.SpinAgain & 0x1
	if input.PermaSpin&0x1 != 0x0 {
		stepZ = canonicalPermaStepZScalar(stepZ)
		spinAgain = 0x0
	}
	return packSlope(input.SlopeX, input.SlopeY) |
		(input.SlopeZ<<0x14)&spriteSlopeZMask32 |
		(input.SizeScore<<0x10)&spriteSizeScoreMask32 |
		(input.StepY<<0xD)&stepYMask32 |
		(input.StepX<<0xA)&stepXMask32 |
		(stepZ<<0x6)&stepZMask32 |
		(input.StepTick<<0x2)&stepTickMask32 |
		(input.PermaSpin<<0x1)&permaSpinMask32 |
		spinAgain&spinAgainMask32
}

func incrementSlope(slope simd.Int32s) simd.Int32s {
	incrementedSlope := slope.Add(oneSlope)
	return minimumSlope.IfElse(incrementedSlope.Greater(maximumSlope), incrementedSlope)
}

func decrementSlope(slope simd.Int32s) simd.Int32s {
	decrementedSlope := slope.Sub(oneSlope)
	return maximumSlope.IfElse(decrementedSlope.Less(minimumSlope), decrementedSlope)
}

func canonicalPermaStepZ(stepZ simd.Uint32s) simd.Uint32s {
	fullSpin := stepZ.Equal(maximumStepZ)
	xDirection := stepZ.And(permaXDirectionMask)
	yDirection := stepZ.And(permaYDirectionMask)
	xDirection = zeroCoordinates.IfElse(xDirection.Equal(permaXDirectionMask), xDirection)
	yDirection = zeroCoordinates.IfElse(yDirection.Equal(permaYDirectionMask), yDirection)
	yDirection = zeroCoordinates.IfElse(xDirection.NotEqual(zeroCoordinates), yDirection)
	return maximumStepZ.IfElse(fullSpin, xDirection.Or(yDirection))
}

func slopeDirection(slope, velocity simd.Int32s) (simd.Mask32s, simd.Mask32s) {
	forwardDistance := velocity.Sub(slope)
	forwardDistance = forwardDistance.Add(slopeCycle).IfElse(
		forwardDistance.Less(zeroSlope), forwardDistance)
	increment := forwardDistance.Greater(zeroSlope).
		And(forwardDistance.LessEqual(halfSlopeCycle))
	decrement := forwardDistance.Greater(halfSlopeCycle)
	return increment, decrement
}

// updateSlopeZ advances the finite Z spin in the dominant velocity direction.
// Positive velocity decrements the atlas frame; negative velocity increments it.
func updateSlopeZ(slopeZ simd.Uint32s, velocityX, velocityY simd.Int32s) simd.Uint32s {
	absX := velocityX.Abs()
	absY := velocityY.Abs()
	useXDirection := absX.GreaterEqual(absY)
	dominantVelocity := velocityX.IfElse(useXDirection, velocityY)
	incrementSlopeZ := dominantVelocity.Less(zeroSlope)
	decrementSlopeZ := dominantVelocity.Greater(zeroSlope)

	decremented := slopeZ.Sub(oneCoordinates)
	decremented = maximumSlopeZ.IfElse(slopeZ.Equal(zeroCoordinates), decremented)

	incremented := incrementSlopeZFrame(slopeZ)

	slopeZ = incremented.IfElse(incrementSlopeZ, slopeZ)
	return decremented.IfElse(decrementSlopeZ, slopeZ)
}

func incrementSlopeZFrame(slopeZ simd.Uint32s) simd.Uint32s {
	incremented := slopeZ.Add(oneCoordinates)
	return zeroCoordinates.IfElse(incremented.GreaterEqual(slopeZFrameCount), incremented)
}

func updateStepTick(
	stepTick, sizeScore simd.Uint32s,
	hasSteps simd.Mask32s,
) (simd.Uint32s, simd.Mask32s) {
	active := sizeScore.NotEqual(zeroCoordinates).And(hasSteps)
	stepDue := stepTick.Equal(zeroCoordinates).And(active)
	waiting := stepTick.NotEqual(zeroCoordinates).And(active)
	stepTick = stepTick.Sub(oneCoordinates.Masked(waiting))
	interval := sizeScore.ShiftAllRight(1)
	interval = oneCoordinates.IfElse(interval.Less(oneCoordinates), interval)
	stepTick = interval.Sub(oneCoordinates).IfElse(stepDue, stepTick)
	return stepTick, stepDue
}

func slopeDistance(slope, velocity simd.Int32s) simd.Uint32s {
	difference := velocity.Sub(slope).Abs()
	wrappedDistance := slopeCycle.Sub(difference)
	return wrappedDistance.IfElse(
		difference.Greater(halfSlopeCycle), difference,
	).ConvertToUint32()
}

func moveSlopeTowardVelocity(
	slope, velocity simd.Int32s,
	step simd.Mask32s,
) simd.Int32s {
	increment, decrement := slopeDirection(slope, velocity)
	increment = increment.And(step)
	decrement = decrement.And(step)

	slope = incrementSlope(slope).IfElse(increment, slope)
	return decrementSlope(slope).IfElse(decrement, slope)
}

func updateSteps(
	slopeX, slopeY, velocityX, velocityY simd.Int32s,
	stepX, stepY simd.Uint32s,
	stepDue simd.Mask32s,
) (simd.Int32s, simd.Int32s, simd.Uint32s, simd.Uint32s) {
	advanceX := stepX.NotEqual(zeroCoordinates).And(stepDue)
	advanceY := stepY.NotEqual(zeroCoordinates).And(stepDue)
	slopeX = moveSlopeTowardVelocity(slopeX, velocityX, advanceX)
	slopeY = moveSlopeTowardVelocity(slopeY, velocityY, advanceY)
	stepX = stepX.Sub(oneCoordinates.Masked(advanceX))
	stepY = stepY.Sub(oneCoordinates.Masked(advanceY))
	return slopeX, slopeY, stepX, stepY
}

func dampVelocity(velocity simd.Int32s) simd.Int32s {
	positive := velocity.Greater(zeroSlope)
	negative := velocity.Less(zeroSlope)
	velocity = velocity.Sub(oneSlope.Masked(positive))
	return velocity.Add(oneSlope.Masked(negative))
}

func updateDirectionChange(
	slopeX, slopeY, velocityX, velocityY simd.Int32s,
	stepX, stepY, stepZ, permaSpin, spinAgain simd.Uint32s,
	changedX, changedY simd.Mask32s,
) (simd.Uint32s, simd.Uint32s, simd.Uint32s, simd.Uint32s) {
	directionChanged := changedX.Or(changedY)
	permaSpinActive := permaSpin.NotEqual(zeroCoordinates)
	stepX = slopeDistance(slopeX, velocityX).IfElse(changedX, stepX)
	stepY = slopeDistance(slopeY, velocityY).IfElse(changedY, stepY)
	stepX = zeroCoordinates.IfElse(permaSpinActive.And(changedY), stepX)
	stepY = zeroCoordinates.IfElse(permaSpinActive.And(changedX), stepY)

	incrementX, decrementX := slopeDirection(slopeX, velocityX)
	incrementY, decrementY := slopeDirection(slopeY, velocityY)
	permaXCode := permaXIncrement.Masked(decrementX).
		Or(permaXDecrement.Masked(incrementX))
	permaYCode := permaYIncrement.Masked(decrementY).
		Or(permaYDecrement.Masked(incrementY))
	stepZ = permaXCode.IfElse(permaSpinActive.And(changedX), stepZ)
	stepZ = permaYCode.IfElse(permaSpinActive.And(changedY), stepZ)
	spinAgain = oneCoordinates.IfElse(directionChanged, spinAgain)
	return stepX, stepY, stepZ, spinAgain
}

// UpdateRocks first advances one strided movement phase, then checks collisions
// for every rock.
func UpdateRocks(rocks *Rocks, tickSize, tick int, mouse *controls.MouseInfo) {
	if tickSize < 1 {
		panic("tick size must be greater than zero")
	}
	if tick < 0 {
		panic("tick must not be negative")
	}
	if tickSize == 1 {
		updateMovementContiguous(rocks)
		collideRocks(rocks, mouse)
		return
	}

	if cap(rocks.scheduledPositions) < SIMDVectorSize {
		rocks.scheduledPositions = make(RockPositions, SIMDVectorSize)
		rocks.scheduledSprites = make(RockSprites, SIMDVectorSize)
		rocks.scheduledIndices = make([]int, SIMDVectorSize)
	}
	positions := rocks.scheduledPositions[:SIMDVectorSize]
	sprites := rocks.scheduledSprites[:SIMDVectorSize]
	indices := rocks.scheduledIndices[:SIMDVectorSize]
	nextIndex := tick % tickSize

	for nextIndex < len(rocks.Positions) {
		loadedRockCount := 0
		for loadedRockCount < SIMDVectorSize && nextIndex < len(rocks.Positions) {
			indices[loadedRockCount] = nextIndex
			positions[loadedRockCount] = rocks.Positions[nextIndex]
			sprites[loadedRockCount] = rocks.Sprites[nextIndex]
			loadedRockCount++
			nextIndex += tickSize
		}

		batch := Rocks{
			Positions:   positions[:loadedRockCount],
			Sprites:     sprites[:loadedRockCount],
			Atlas:       rocks.Atlas,
			AmountScale: rocks.AmountScale,
		}
		updateMovementContiguous(&batch)
		for lane := range loadedRockCount {
			index := indices[lane]
			rocks.Positions[index] = positions[lane]
			rocks.Sprites[index] = sprites[lane]
		}
	}
	collideRocks(rocks, mouse)
}

func updateMovementContiguous(rocks *Rocks) {
	for firstRockIndex := 0; firstRockIndex < len(rocks.Positions); firstRockIndex += SIMDVectorSize {
		packedPositionLanes, loadedRockCount := simd.LoadUint32sPart(rocks.Positions[firstRockIndex:])
		packedSpriteLanes, _ := simd.LoadUint32sPart(rocks.Sprites[firstRockIndex:])

		sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY := unpackSpritesSIMD(packedSpriteLanes)
		positionX, positionY, velocityX, velocityY := unpackPositionsSIMD(packedPositionLanes)
		positionX = positionX.ConvertToInt32().Add(velocityX).ConvertToUint32()
		positionY = positionY.ConvertToInt32().Add(velocityY).ConvertToUint32()

		permaSpinActive := permaSpin.NotEqual(zeroCoordinates)

		hasSteps := stepX.NotEqual(zeroCoordinates).
			Or(stepY.NotEqual(zeroCoordinates)).
			Or(stepZ.NotEqual(zeroCoordinates)).
			Or(spinAgain.NotEqual(zeroCoordinates)).
			Or(permaSpinActive)
		stepTick, stepDue := updateStepTick(stepTick, sizeScore, hasSteps)
		finiteStepDue := stepDue.And(permaSpin.Equal(zeroCoordinates))
		slopeX, slopeY, stepX, stepY = updateSteps(
			slopeX, slopeY,
			velocityX, velocityY,
			stepX, stepY,
			finiteStepDue)
		permaIncrementX := stepZ.And(permaXDirectionMask).Equal(permaXIncrement).
			And(permaSpinActive).And(stepDue)
		permaDecrementX := stepZ.And(permaXDirectionMask).Equal(permaXDecrement).
			And(permaSpinActive).And(stepDue)
		permaIncrementY := stepZ.And(permaYDirectionMask).Equal(permaYIncrement).
			And(permaSpinActive).And(stepDue)
		permaDecrementY := stepZ.And(permaYDirectionMask).Equal(permaYDecrement).
			And(permaSpinActive).And(stepDue)
		slopeX = incrementSlope(slopeX).IfElse(permaIncrementX, slopeX)
		slopeX = decrementSlope(slopeX).IfElse(permaDecrementX, slopeX)
		slopeY = incrementSlope(slopeY).IfElse(permaIncrementY, slopeY)
		slopeY = decrementSlope(slopeY).IfElse(permaDecrementY, slopeY)
		permaStepX := permaIncrementX.Or(permaDecrementX).
			And(stepX.NotEqual(zeroCoordinates))
		permaStepY := permaIncrementY.Or(permaDecrementY).
			And(stepY.NotEqual(zeroCoordinates))
		stepX = stepX.Sub(oneCoordinates.Masked(permaStepX))
		stepY = stepY.Sub(oneCoordinates.Masked(permaStepY))

		// A collision that leaves StepZ at zero sets SpinAgain. On the next due
		// tick, SlopeZ advances once, StepZ is loaded with 15, and SpinAgain is
		// cleared. The next 15 due ticks advance SlopeZ 15 more times while StepZ
		// counts down, making 16 advances total and returning to the starting frame.
		startFullSpin := stepZ.Equal(zeroCoordinates).
			And(spinAgain.NotEqual(zeroCoordinates)).
			And(stepDue)
		finiteStepZ := stepZ.NotEqual(zeroCoordinates).
			And(stepDue).
			And(permaSpin.Equal(zeroCoordinates))
		stepSlopeZ := finiteStepZ.
			Or(startFullSpin).
			Or(permaSpinActive.And(stepDue))
		slopeZ = updateSlopeZ(slopeZ, velocityX, velocityY).IfElse(stepSlopeZ, slopeZ)
		stepZ = stepZ.Sub(oneCoordinates.Masked(finiteStepZ))
		stepZ = maximumStepZ.IfElse(startFullSpin, stepZ)
		spinAgain = zeroCoordinates.IfElse(startFullSpin, spinAgain)

		dampingActive := permaSpin.Equal(zeroCoordinates)
		velocityX = dampVelocity(velocityX).IfElse(dampingActive, velocityX)
		velocityY = dampVelocity(velocityY).IfElse(dampingActive, velocityY)
		stopped := velocityX.Equal(zeroSlope).
			And(velocityY.Equal(zeroSlope)).
			And(dampingActive).
			And(stepZ.Equal(zeroCoordinates)).
			And(spinAgain.Equal(zeroCoordinates))
		stepX = zeroCoordinates.IfElse(stopped, stepX)
		stepY = zeroCoordinates.IfElse(stopped, stepY)
		stepZ = zeroCoordinates.IfElse(stopped, stepZ)
		stepTick = zeroCoordinates.IfElse(stopped, stepTick)
		permaSpin = zeroCoordinates.IfElse(stopped, permaSpin)
		spinAgain = zeroCoordinates.IfElse(stopped, spinAgain)

		updatedPackedPositions := packPositionsSIMD(
			positionX, positionY,
			velocityX, velocityY)

		updatedPackedSprites := packSpritesSIMD(
			sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY)
		updatedPackedPositions.StorePart(rocks.Positions[firstRockIndex : firstRockIndex+loadedRockCount])
		updatedPackedSprites.StorePart(rocks.Sprites[firstRockIndex : firstRockIndex+loadedRockCount])
	}
}

func collideRocks(rocks *Rocks, mouse *controls.MouseInfo) {
	mouseX, mouseY := zeroSlope, zeroSlope
	mouseActive := mouse != nil
	mouseDown := false
	if mouseActive {
		mouseX = simd.BroadcastInt32s(int32(mouse.Position.X))
		mouseY = simd.BroadcastInt32s(int32(mouse.Position.Y))
		mouseDown = mouse.Down
	}

	for firstRockIndex := 0; firstRockIndex < len(rocks.Positions); firstRockIndex += SIMDVectorSize {
		packedPositionLanes, loadedRockCount := simd.LoadUint32sPart(rocks.Positions[firstRockIndex:])
		packedSpriteLanes, _ := simd.LoadUint32sPart(rocks.Sprites[firstRockIndex:])
		sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY := unpackCollisionSpritesSIMD(packedSpriteLanes)
		positionX, positionY, velocityX, velocityY := unpackPositionsSIMD(packedPositionLanes)
		previousVelocityX := velocityX
		previousVelocityY := velocityY
		mouseHit := zeroSlope.ToMask()
		if mouseActive {
			velocityX, velocityY, mouseHit = setMouseVelocity(
				positionX, positionY,
				velocityX, velocityY,
				hoverRadius(sizeScore, rocks.AmountScale),
				mouseDown,
				mouseX, mouseY)
		}

		var wallHitX, wallHitY simd.Mask32s
		positionX, positionY, velocityX, velocityY, wallHitX, wallHitY = collideWalls(
			positionX, positionY,
			velocityX, velocityY,
			collisionRadius(sizeScore, rocks.AmountScale))
		impact := mouseHit.Or(wallHitX).Or(wallHitY)
		changedX := velocityX.NotEqual(previousVelocityX)
		changedY := velocityY.NotEqual(previousVelocityY)
		stepX, stepY, stepZ, spinAgain = updateDirectionChange(
			slopeX, slopeY, velocityX, velocityY,
			stepX, stepY, stepZ, permaSpin, spinAgain,
			changedX, changedY)
		finiteImpact := impact.And(permaSpin.Equal(zeroCoordinates))
		collisionStepZ := stepX.Add(stepY)
		stepZ = collisionStepZ.IfElse(finiteImpact, stepZ)
		slopeX = velocityX.IfElse(impact, slopeX)
		slopeY = velocityY.IfElse(impact, slopeY)
		spinAgain = oneCoordinates.IfElse(
			finiteImpact.And(stepZ.Equal(zeroCoordinates)), spinAgain)
		spinAgain = spinAgain.Masked(permaSpin.Equal(zeroCoordinates))

		packPositionsSIMD(positionX, positionY, velocityX, velocityY).
			StorePart(rocks.Positions[firstRockIndex : firstRockIndex+loadedRockCount])
		packCollisionSpritesSIMD(
			sizeScore, slopeZ,
			stepX, stepY,
			stepZ, stepTick,
			permaSpin,
			spinAgain,
			slopeX, slopeY).
			StorePart(rocks.Sprites[firstRockIndex : firstRockIndex+loadedRockCount])
	}
}
