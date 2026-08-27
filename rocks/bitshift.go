package rocks

import (
	"simd"

	"github.com/ninesl/dice-will-roll/settings"
)

/*
SIMD REFERENCE

Each simd.Uint32s operation runs independently on every uint32 lane.

    a = [a0 a1 a2 a3]
    b = [b0 b1 b2 b3]

    a.And(b) = [a0 & b0, a1 & b1, a2 & b2, a3 & b3]


BITWISE OPERATIONS

AND    a.And(b)    a & b

      11001010
    & 10101100
    = 10001000

    Keeps bits that are 1 in both inputs. Commonly used to isolate a field.

OR     a.Or(b)     a | b

      11000000
    | 00101010
    = 11101010

    Keeps bits that are 1 in either input. Used to combine fields that occupy
    different bit ranges. OR is not addition:

      0101 | 0011 = 0111
      0101 + 0011 = 1000

XOR    a.Xor(b)    a ^ b

      11001010
    ^ 10101100
    = 01100110

    Keeps bits that differ and clears bits that are equal.

AND NOT    a.AndNot(b)    a &^ b

       11001010
    &^ 00101100
     = 11000010

    Keeps bits from a except where b contains a 1. The bits in b select which
    bits to clear.

NOT    a.Not()    ^a

    ^ 11001010
    = 00110101

    Flips every 0 to 1 and every 1 to 0.


BIT MOVEMENT

SHIFT LEFT    a.ShiftAllLeft(2)    a << 2

      00101101 << 2
    = 10110100

    Moves bits left, drops bits past the left edge, and adds zeros on the right.

SHIFT RIGHT    a.ShiftAllRight(2)    a >> 2

      10110100 >> 2
    = 00101101

    Moves bits right, drops bits past the right edge, and adds zeros on the left.

ROTATE LEFT    a.RotateAllLeft(2)    bits.RotateLeft32(a, 2)

      10110001 rotate left 2
    = 11000110

    Moves bits left and wraps the shifted-out bits around to the right edge.

ROTATE RIGHT    a.RotateAllRight(2)    bits.RotateLeft32(a, -2)

      10110001 rotate right 2
    = 01101100

    Moves bits right and wraps the shifted-out bits around to the left edge.


ARITHMETIC

ADD    a.Add(b)    a + b

      00000101  (5)
    + 00000011  (3)
    = 00001000  (8)

SUBTRACT    a.Sub(b)    a - b

      00000101  (5)
    - 00000011  (3)
    = 00000010  (2)

MULTIPLY    a.Mul(b)    a * b

      00000101  (5)
    * 00000011  (3)
    = 00001111  (15)

Each operation performs its arithmetic independently in every lane.


LANE COMPARISONS AND SELECTION

COMPARE    currentValues.Equal(expectedValues)

    currentValues  = [0011  0101  0111  1001]
    expectedValues = [0011  0010  0111  0001]
    matchingLanes  = [true  false true  false]

    matchingLanes := currentValues.Equal(expectedValues)

    Produces one boolean per lane, not integer zero or one. Less, LessEqual,
    Greater, GreaterEqual, and NotEqual produce masks in the same way.

SELECT    updatedValues.IfElse(updateLanes, originalValues)

    updatedValues  = [0001  0010  0011  0100]
    originalValues = [1001  1010  1011  1100]
    updateLanes    = [true  false true  false]
    selectedValues = [0001  1010  0011  1100]

    selectedValues := updatedValues.IfElse(
        updateLanes,
        originalValues,
    )

    SIMD examines each lane independently. A true update lane copies from
    updatedValues. A false update lane copies from originalValues.

CLEAR DISABLED LANES    inputValues.Masked(enabledLanes)

    inputValues    = [0001  0010  0011  0100]
    enabledLanes   = [true  false true  false]
    filteredValues = [0001  0000  0011  0000]

    filteredValues := inputValues.Masked(enabledLanes)

    A true enabled lane retains its complete input value. A false enabled lane
    replaces its complete input value with zero.
*/

// RockPositions stores packed uint32 values with layout [SX:4][SY:4][X:12][Y:12].
type RockPositions []uint32

// RockSprites stores packed sprite orientation, size, and animation state.
type RockSprites []uint32

const (
	packedSlopeXMask     uint32 = 0xF000_0000
	packedSlopeYMask     uint32 = 0x0F00_0000
	packedPositionXMask  uint32 = 0x00FF_F000
	packedPositionYMask  uint32 = 0x0000_0FFF
	spriteRotationMask32 uint32 = 0x00F0_0000
	spriteRockSizeMask32 uint32 = 0x000F_0000
	rotationStepsXMask32 uint32 = 0x0000_0F00
	rotationStepsYMask32 uint32 = 0x0000_F000
	animationTickMask32  uint32 = 0x0000_00F0
)

type Rocks struct {
	Positions RockPositions
	Sprites   RockSprites
	Atlas     *RockSpriteAtlas
}

var (
	SIMDVectorSize int

	slopeXMask, slopeYMask                  simd.Uint32s
	positionXMask, positionYMask            simd.Uint32s
	spriteRockSizeMask, spriteRotationMask  simd.Uint32s
	rotationStepsXMask, rotationStepsYMask  simd.Uint32s
	animationTickMask                       simd.Uint32s
	oneCoordinates, zeroCoordinates         simd.Uint32s
	maximumRockSize, rotationFrameCount     simd.Uint32s
	maximumRotationFrame                    simd.Uint32s
	maximumPixelX, maximumPixelY            simd.Uint32s
	zeroSlope, oneSlope                     simd.Int32s
	minimumSlope, maximumSlope              simd.Int32s
	slopeOffset, slopeCycle, halfSlopeCycle simd.Int32s
)

func init() {
	// Go 1.27 documents BUG(global initialization): "SIMD-dependent var
	// initializers don't work." Keep this call out of the var initializer.
	// https://github.com/golang/go/commit/c83d55f69519f8cc633215287142f394d3b66087#diff-81ad425e4481409eeb696055ddac6751598ac9be712c00ae47324cd0aa3db38d
	SIMDVectorSize = simd.Uint32s{}.Len()

	// Position: [ SX:4 ][ SY:4 ][ X:12 ][ Y:12 ]
	slopeXMask = simd.BroadcastUint32s(packedSlopeXMask)
	slopeYMask = simd.BroadcastUint32s(packedSlopeYMask)
	positionXMask = simd.BroadcastUint32s(packedPositionXMask)
	positionYMask = simd.BroadcastUint32s(packedPositionYMask)

	// Sprite: [ SpriteSlopes:8 ][ RotationFrame:4 ][ SizeCode:4 ][ RotationStepsY:4 ][ RotationStepsX:4 ][ AnimationTick:4 ][ Unused:4 ]
	spriteRotationMask = simd.BroadcastUint32s(spriteRotationMask32)
	spriteRockSizeMask = simd.BroadcastUint32s(spriteRockSizeMask32)
	rotationStepsXMask = simd.BroadcastUint32s(rotationStepsXMask32)
	rotationStepsYMask = simd.BroadcastUint32s(rotationStepsYMask32)
	animationTickMask = simd.BroadcastUint32s(animationTickMask32)

	zeroCoordinates = simd.BroadcastUint32s(0)
	oneCoordinates = simd.BroadcastUint32s(1)
	maximumRockSize = simd.BroadcastUint32s(BitSpriteSlopeCodeCount - 1)
	rotationFrameCount = simd.BroadcastUint32s(AtlasRotationFrames)
	maximumRotationFrame = simd.BroadcastUint32s(AtlasRotationFrames - 1)
	zeroSlope = simd.BroadcastInt32s(0)
	oneSlope = simd.BroadcastInt32s(1)
	minimumSlope = simd.BroadcastInt32s(-7)
	maximumSlope = simd.BroadcastInt32s(7)
	slopeOffset = simd.BroadcastInt32s(8)
	slopeCycle = simd.BroadcastInt32s(atlasSlopeStates)
	halfSlopeCycle = simd.BroadcastInt32s(7)
}

// Init precomputes screen-dependent SIMD values after settings initializes the
// monitor resolution. Resolution dimensions are counts, so the final valid
// pixel coordinate is one less than the corresponding dimension.
func Init(screen settings.ScreenSettings) {
	// TODO: max resolution for 4096?

	// can we force windowed mode and a maximum size?
	// can only have up to 12 bits of pixel, 4095

	// 4k is 3840×2160 16:9

	// 21:9 ultrawide
	// 3440 x 1440, 2560x1080, 3840x1600
	// FIXME: this will be an issue eventually. need to normalize x,y
	// based on scale factor NOT the literal screen coordinates
	// should only really matter for drawing? The calculations shouldn't
	// have to change if we implment true scaling later...

	// NOTE: will also need to do colorblind and other things
	// https://howtomarketagame.com/2021/09/28/implement-these-features-to-avoid-bad-reviews/

	maximumPixelX = simd.BroadcastUint32s(uint32(screen.ResolutionX - 1))
	maximumPixelY = simd.BroadcastUint32s(uint32(screen.ResolutionY - 1))
}

// unpackSlopes decodes shared biased codes 1..15 to signed slopes -7..7; code 0 stays unused.
func unpackSlopes(packed simd.Uint32s) (slopeX, slopeY simd.Int32s) {
	/*
		Position and sprite slopes use the same biased four-bit representation:
		0000 = unused
		0001 = -7
		0111 = -1
		1000 =  0
		1001 = +1
		1111 = +7
		SX and SY occupy the same high eight bits in packed positions and sprites:

		    [ SX:4 ][ SY:4 ][ remaining 24 bits ]
	*/
	slopeXCode := packed.And(slopeXMask).ShiftAllRight(28)
	slopeYCode := packed.And(slopeYMask).ShiftAllRight(24)
	slopeX = slopeXCode.ConvertToInt32().Sub(slopeOffset).
		Masked(slopeXCode.NotEqual(zeroCoordinates))
	slopeY = slopeYCode.ConvertToInt32().Sub(slopeOffset).
		Masked(slopeYCode.NotEqual(zeroCoordinates))

	return slopeX, slopeY
}

// unpackPositions returns unsigned X/Y positions and signed X/Y slopes.
func unpackPositions(packedPositions simd.Uint32s) (
	positionX, positionY simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) {
	// [ SX:4 ][ SY:4 ][ X:12 ][ Y:12 ]
	//    31..28   27..24   23..12   11..0
	slopeX, slopeY = unpackSlopes(packedPositions)
	positionX = packedPositions.And(positionXMask).ShiftAllRight(12)
	positionY = packedPositions.And(positionYMask)
	return positionX, positionY, slopeX, slopeY
}

// packSlopes encodes signed slopes -7..7 as shared biased codes 1..15.
func packSlopes(slopeX, slopeY simd.Int32s) simd.Uint32s {
	slopeXBits := slopeX.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(28).And(slopeXMask)
	slopeYBits := slopeY.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(24).And(slopeYMask)
	return slopeXBits.Or(slopeYBits)
}

// packPositions packs unsigned X/Y positions with signed X/Y slopes.
func packPositions(positionX, positionY simd.Uint32s, slopeX, slopeY simd.Int32s) simd.Uint32s {
	positionXBits := positionX.ShiftAllLeft(12).And(positionXMask)
	positionYBits := positionY.And(positionYMask)
	return packSlopes(slopeX, slopeY).Or(positionXBits).Or(positionYBits)
}

// UnpackPosition decodes one packed position.
func UnpackPosition(pos uint32) (positionX, positionY, slopeX, slopeY int) {
	slopeX, slopeY = UnpackSlope(pos)
	positionX = int((pos & packedPositionXMask) >> 12)
	positionY = int(pos & packedPositionYMask)
	return positionX, positionY, slopeX, slopeY
}

// UnpackSlope decodes the shared slope fields in one packed value.
func UnpackSlope(slope uint32) (slopeX, slopeY int) {
	slopeXCode := (slope & packedSlopeXMask) >> 28
	slopeYCode := (slope & packedSlopeYMask) >> 24
	if slopeXCode != 0 {
		slopeX = int(slopeXCode) - 8
	}
	if slopeYCode != 0 {
		slopeY = int(slopeYCode) - 8
	}
	return slopeX, slopeY
}

// PackPosition encodes one packed position.
func PackPosition(positionX, positionY uint32, slopeX, slopeY int32) uint32 {
	return packSlope(slopeX, slopeY) |
		(positionX<<12)&packedPositionXMask |
		positionY&packedPositionYMask
}

// packSlope encodes scalar slopes into their shared packed fields.
func packSlope(slopeX, slopeY int32) uint32 {
	slopeXCode := uint32(slopeX + 0b1000)
	slopeYCode := uint32(slopeY + 0b1000)
	return (slopeXCode<<28)&packedSlopeXMask |
		(slopeYCode<<24)&packedSlopeYMask
}

// [ SpriteSlopeX:4 ][ SpriteSlopeY:4 ][ RotationFrame:4 ][ SizeCode:4 ][ RotationStepsY:4 ][ RotationStepsX:4 ][ AnimationTick:4 ][ Unused:4 ]
//
//	31..28          27..24          23..20             19..16        15..12              11..8              7..4              3..0
//
// TODO: add metadata in 16 unused bits, explosion mask, etc.
// unpackSprites returns unsigned size/rotation and signed slopes.
func unpackSprites(sprite simd.Uint32s) (
	rockSize, rotationFrame simd.Uint32s,
	rotationStepsX, rotationStepsY, animationTick simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) {
	slopeX, slopeY = unpackSlopes(sprite)
	rotationFrame = sprite.And(spriteRotationMask).ShiftAllRight(20)
	rockSize = sprite.And(spriteRockSizeMask).ShiftAllRight(16)
	rotationStepsY = sprite.And(rotationStepsYMask).ShiftAllRight(12)
	rotationStepsX = sprite.And(rotationStepsXMask).ShiftAllRight(8)
	animationTick = sprite.And(animationTickMask).ShiftAllRight(4)

	return rockSize, rotationFrame,
		rotationStepsX, rotationStepsY, animationTick,
		slopeX, slopeY
}

// packSprites packs visual orientation, size, and animation state.
func packSprites(
	rockSize, rotationFrame simd.Uint32s,
	rotationStepsX, rotationStepsY, animationTick simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) simd.Uint32s {
	rotationBits := rotationFrame.ShiftAllLeft(20).And(spriteRotationMask)
	rockSizeBits := rockSize.ShiftAllLeft(16).And(spriteRockSizeMask)
	rotationStepsYBits := rotationStepsY.ShiftAllLeft(12).And(rotationStepsYMask)
	rotationStepsXBits := rotationStepsX.ShiftAllLeft(8).And(rotationStepsXMask)
	animationTickBits := animationTick.ShiftAllLeft(4).And(animationTickMask)

	return packSlopes(slopeX, slopeY).
		Or(rotationBits).
		Or(rockSizeBits).
		Or(rotationStepsYBits).
		Or(rotationStepsXBits).
		Or(animationTickBits)
}

// UnpackSprite decodes one packed sprite.
func UnpackSprite(packedSprite uint32) (
	rockSize, rotationFrame,
	rotationStepsX, rotationStepsY, animationTick,
	slopeX, slopeY int,
) {
	slopeX, slopeY = UnpackSlope(packedSprite)
	rockSize = int((packedSprite & spriteRockSizeMask32) >> 16)
	rotationFrame = int((packedSprite & spriteRotationMask32) >> 20)
	rotationStepsY = int((packedSprite & rotationStepsYMask32) >> 12)
	rotationStepsX = int((packedSprite & rotationStepsXMask32) >> 8)
	animationTick = int((packedSprite & animationTickMask32) >> 4)
	return rockSize, rotationFrame,
		rotationStepsX, rotationStepsY, animationTick,
		slopeX, slopeY
}

// PackSprite encodes one packed sprite.
func PackSprite(
	rockSize, rotationFrame,
	rotationStepsX, rotationStepsY, animationTick uint32,
	slopeX, slopeY int32,
) uint32 {
	return packSlope(slopeX, slopeY) |
		(rotationFrame<<20)&spriteRotationMask32 |
		(rockSize<<16)&spriteRockSizeMask32 |
		(rotationStepsY<<12)&rotationStepsYMask32 |
		(rotationStepsX<<8)&rotationStepsXMask32 |
		(animationTick<<4)&animationTickMask32
}

func incrementSpriteSlope(spriteSlope simd.Int32s) simd.Int32s {
	incrementedSlope := spriteSlope.Add(oneSlope)
	return minimumSlope.IfElse(incrementedSlope.Greater(maximumSlope), incrementedSlope)
}

// updateRotationFrame matches SimpleRock.UpdateAnimation's Z rotation direction.
func updateRotationFrame(rotationFrame simd.Uint32s, slopeX, slopeY simd.Int32s) simd.Uint32s {
	isMoving := slopeX.NotEqual(zeroSlope).And(slopeY.NotEqual(zeroSlope))
	incrementFrame := isMoving.And(slopeX.GreaterEqual(zeroSlope))
	decrementFrame := isMoving.And(slopeX.Less(zeroSlope))

	decremented := rotationFrame.Sub(oneCoordinates)
	decremented = maximumRotationFrame.IfElse(rotationFrame.Equal(zeroCoordinates), decremented)

	incremented := rotationFrame.Add(oneCoordinates)
	incremented = zeroCoordinates.IfElse(incremented.GreaterEqual(rotationFrameCount), incremented)

	rotationFrame = incremented.IfElse(incrementFrame, rotationFrame)
	return decremented.IfElse(decrementFrame, rotationFrame)
}

func updateAnimationTick(
	animationTick, rockSize simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) (simd.Uint32s, simd.Mask32s) {
	isMoving := slopeX.NotEqual(zeroSlope).Or(slopeY.NotEqual(zeroSlope))
	animationTick = animationTick.Add(oneCoordinates.Masked(isMoving))
	rotationDue := animationTick.GreaterEqual(rockSize).
		And(rockSize.NotEqual(zeroCoordinates)).
		And(isMoving)
	animationTick = zeroCoordinates.IfElse(rotationDue, animationTick)
	return animationTick, rotationDue
}

func spriteSlopeDistance(spriteSlope, positionSlope simd.Int32s) simd.Uint32s {
	difference := positionSlope.Sub(spriteSlope).Abs()
	wrappedDistance := slopeCycle.Sub(difference)
	return wrappedDistance.IfElse(
		difference.Greater(halfSlopeCycle), difference,
	).ConvertToUint32()
}

func updateRotationSteps(
	spriteSlopeX, spriteSlopeY, slopeX, slopeY simd.Int32s,
	rotationStepsX, rotationStepsY simd.Uint32s,
	hitX, hitY, animationDue simd.Mask32s,
) (simd.Int32s, simd.Int32s, simd.Uint32s, simd.Uint32s) {
	tumbleStepsX := slopeX.Abs().ConvertToUint32()
	tumbleStepsY := slopeY.Abs().ConvertToUint32()
	rotationStepsX = spriteSlopeDistance(spriteSlopeX, slopeX).IfElse(hitX, rotationStepsX)
	rotationStepsY = tumbleStepsY.Add(tumbleStepsY).IfElse(hitX, rotationStepsY)
	rotationStepsX = tumbleStepsX.Add(tumbleStepsX).IfElse(hitY, rotationStepsX)
	rotationStepsY = spriteSlopeDistance(spriteSlopeY, slopeY).IfElse(hitY, rotationStepsY)

	rotateX := rotationStepsX.NotEqual(zeroCoordinates).And(animationDue)
	rotateY := rotationStepsY.NotEqual(zeroCoordinates).And(animationDue)
	spriteSlopeX = incrementSpriteSlope(spriteSlopeX).IfElse(rotateX, spriteSlopeX)
	spriteSlopeY = incrementSpriteSlope(spriteSlopeY).IfElse(rotateY, spriteSlopeY)
	rotationStepsX = rotationStepsX.Sub(oneCoordinates.Masked(rotateX))
	rotationStepsY = rotationStepsY.Sub(oneCoordinates.Masked(rotateY))
	return spriteSlopeX, spriteSlopeY, rotationStepsX, rotationStepsY
}

// update loads matching batches from the two global rock slices, unpacks their
// fields into SIMD registers, updates them, repacks them, and writes them back.
func UpdateRocks(rocks *Rocks) {
	if len(rocks.Positions) != len(rocks.Sprites) {
		panic("positions and sprites must have equal lengths")
	}

	for firstRockIndex := 0; firstRockIndex < len(rocks.Positions); firstRockIndex += SIMDVectorSize {
		packedPositionLanes, loadedRockCount := simd.LoadUint32sPart(rocks.Positions[firstRockIndex:])
		packedSpriteLanes, _ := simd.LoadUint32sPart(rocks.Sprites[firstRockIndex:])

		rockSize, rotationFrame,
			rotationStepsX, rotationStepsY, animationTick,
			spriteSlopeX, spriteSlopeY := unpackSprites(packedSpriteLanes)
		positionX, positionY, slopeX, slopeY := unpackPositions(packedPositionLanes)

		// Zero out everything if positionX or positionY is 0.
		// 0 means dead rock
		// TODO: show 0001_000_000 example
		//rockWasVisible := positionX.NotEqual(zeroCoordinates).
		//	And(positionY.NotEqual(zeroCoordinates))

		positionX = positionX.ConvertToInt32().Add(slopeX).ConvertToUint32()
		positionY = positionY.ConvertToInt32().Add(slopeY).ConvertToUint32()

		positionX, positionY, slopeX, slopeY, hitX, hitY := collideWalls(
			positionX, positionY,
			slopeX, slopeY)

		animationTick, animationDue := updateAnimationTick(animationTick, rockSize, slopeX, slopeY)
		nextRotationFrame := updateRotationFrame(rotationFrame, slopeX, slopeY)
		rotationFrame = nextRotationFrame.IfElse(animationDue, rotationFrame)
		spriteSlopeX, spriteSlopeY,
			rotationStepsX, rotationStepsY = updateRotationSteps(
			spriteSlopeX, spriteSlopeY,
			slopeX, slopeY,
			rotationStepsX, rotationStepsY,
			hitX, hitY, animationDue)

		updatedPackedPositions := packPositions(
			positionX, positionY,
			slopeX, slopeY)
		// updatedPackedPositions = updatedPackedPositions.Masked(rockWasVisible)

		updatedPackedSprites := packSprites(
			rockSize, rotationFrame,
			rotationStepsX, rotationStepsY, animationTick,
			spriteSlopeX, spriteSlopeY)
		//updatedPackedSprites = updatedPackedSprites.Masked(rockWasVisible)

		// StorePart writes only the lanes loaded for this batch. For a short
		// final batch, padded zero lanes are not written past the slice end.
		updatedPackedPositions.StorePart(rocks.Positions[firstRockIndex : firstRockIndex+loadedRockCount])
		updatedPackedSprites.StorePart(rocks.Sprites[firstRockIndex : firstRockIndex+loadedRockCount])
	}
}
