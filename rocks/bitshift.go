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

// RockSprites stores packed uint32 values with layout [SX:4][SY:4][Rotation:4][Size:4][Unused:16].
type RockSprites []uint32

const (
	packedSlopeXMask     uint32 = 0xF000_0000
	packedSlopeYMask     uint32 = 0x0F00_0000
	packedPositionXMask  uint32 = 0x00FF_F000
	packedPositionYMask  uint32 = 0x0000_0FFF
	spriteRotationMask32 uint32 = 0x00F0_0000
	spriteRockSizeMask32 uint32 = 0x000F_0000
)

type Rocks struct {
	Positions RockPositions
	Sprites   RockSprites

	Atlas *RockSpriteAtlas
}

var (
	SIMDVectorSize int

	slopeXMask, slopeYMask                 simd.Uint32s
	positionXMask, positionYMask           simd.Uint32s
	spriteRockSizeMask, spriteRotationMask simd.Uint32s
	oneCoordinates, zeroCoordinates        simd.Uint32s
	maximumPixelX, maximumPixelY           simd.Uint32s
	zeroSlope, oneSlope                    simd.Int32s
	minimumSlope, maximumSlope             simd.Int32s
	slopeOffset, halfSlopeCycle            simd.Int32s
	negativeHalfSlopeCycle                 simd.Int32s
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

	// Sprite: [ SX:4 ][ SY:4 ][ Rotation:4 ][ Size:4 ][ Unused:16 ]
	spriteRotationMask = simd.BroadcastUint32s(spriteRotationMask32)
	spriteRockSizeMask = simd.BroadcastUint32s(spriteRockSizeMask32)

	zeroCoordinates = simd.BroadcastUint32s(0)
	oneCoordinates = simd.BroadcastUint32s(1)
	zeroSlope = simd.BroadcastInt32s(0)
	oneSlope = simd.BroadcastInt32s(1)
	minimumSlope = simd.BroadcastInt32s(-7)
	maximumSlope = simd.BroadcastInt32s(7)
	slopeOffset = simd.BroadcastInt32s(8)
	halfSlopeCycle = simd.BroadcastInt32s(7)
	negativeHalfSlopeCycle = simd.BroadcastInt32s(-7)
	debugRotationFrameCount = simd.BroadcastUint32s(BitSpriteRotationFrames)
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
func UnpackPosition(packedPosition uint32) (positionX, positionY uint32, slopeX, slopeY int32) {
	slopeX, slopeY = UnpackSlope(packedPosition)
	positionX = (packedPosition & packedPositionXMask) >> 12
	positionY = packedPosition & packedPositionYMask
	return positionX, positionY, slopeX, slopeY
}

// UnpackSlope decodes the shared slope fields in one packed value.
func UnpackSlope(packed uint32) (slopeX, slopeY int32) {
	slopeXCode := (packed & packedSlopeXMask) >> 28
	slopeYCode := (packed & packedSlopeYMask) >> 24
	if slopeXCode != 0 {
		slopeX = int32(slopeXCode) - 8
	}
	if slopeYCode != 0 {
		slopeY = int32(slopeYCode) - 8
	}
	return slopeX, slopeY
}

// PackPosition encodes one packed position.
func PackPosition(positionX, positionY uint32, slopeX, slopeY int32) uint32 {
	return PackSlope(slopeX, slopeY) |
		(positionX<<12)&packedPositionXMask |
		positionY&packedPositionYMask
}

// PackSlope encodes scalar slopes into their shared packed fields.
func PackSlope(slopeX, slopeY int32) uint32 {
	slopeXCode := uint32(slopeX + 8)
	slopeYCode := uint32(slopeY + 8)
	return (slopeXCode<<28)&packedSlopeXMask |
		(slopeYCode<<24)&packedSlopeYMask
}

// [ SX:4 ][ SY:4 ][ Rotation:4 ][ rockSize:4 ][ Unused:16 ]
//
//	31..28   27..24      23..20       19..16       15..0
//
// TODO: add metadata in 16 unused bits, explosion mask, etc.
// unpackSprites returns unsigned size/rotation and signed slopes.
func unpackSprites(sprite simd.Uint32s) (
	rockSize, rotation simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) {
	slopeX, slopeY = unpackSlopes(sprite)
	rotation = sprite.And(spriteRotationMask).ShiftAllRight(20)
	//  FIXME: exploding, we don't need it here we do it a different way?

	rockSize = sprite.And(spriteRockSizeMask).ShiftAllRight(16)

	return rockSize, rotation, slopeX, slopeY
}

// packSprites packs unsigned size/rotation with signed X/Y slopes.
func packSprites(rockSize, rotation simd.Uint32s, slopeX, slopeY simd.Int32s) simd.Uint32s {
	rotationBits := rotation.ShiftAllLeft(20).And(spriteRotationMask)
	rockSizeBits := rockSize.ShiftAllLeft(16).And(spriteRockSizeMask)

	return packSlopes(slopeX, slopeY).Or(rotationBits).Or(rockSizeBits)
}

// UnpackSprite decodes one packed sprite.
func UnpackSprite(packedSprite uint32) (rockSize, rotation uint32, slopeX, slopeY int32) {
	slopeX, slopeY = UnpackSlope(packedSprite)
	rockSize = (packedSprite & spriteRockSizeMask32) >> 16
	rotation = (packedSprite & spriteRotationMask32) >> 20
	return rockSize, rotation, slopeX, slopeY
}

// PackSprite encodes one packed sprite.
func PackSprite(rockSize, rotation uint32, slopeX, slopeY int32) uint32 {
	return PackSlope(slopeX, slopeY) |
		(rotation<<20)&spriteRotationMask32 |
		(rockSize<<16)&spriteRockSizeMask32
}

func incrementSpriteSlope(spriteSlope simd.Int32s) simd.Int32s {
	incrementedSlope := spriteSlope.Add(oneSlope)
	return minimumSlope.IfElse(incrementedSlope.Greater(maximumSlope), incrementedSlope)
}

func decrementSpriteSlope(spriteSlope simd.Int32s) simd.Int32s {
	decrementedSlope := spriteSlope.Sub(oneSlope)
	return maximumSlope.IfElse(decrementedSlope.Less(minimumSlope), decrementedSlope)
}

// trailSpriteSlope moves one state toward the position slope with direct -7/+7 rollover.
func trailSpriteSlope(spriteSlope, positionSlope simd.Int32s) simd.Int32s {
	difference := positionSlope.Sub(spriteSlope)
	moveForward := difference.Greater(zeroSlope).
		And(difference.LessEqual(halfSlopeCycle)).
		Or(difference.Less(negativeHalfSlopeCycle))
	trailedSlope := incrementSpriteSlope(spriteSlope).IfElse(
		moveForward,
		decrementSpriteSlope(spriteSlope),
	)
	return spriteSlope.IfElse(spriteSlope.Equal(positionSlope), trailedSlope)
}

// update loads matching batches from the two global rock slices, unpacks their
// fields into SIMD registers, updates them, repacks them, and writes them back.
func update(positions RockPositions, sprites RockSprites) {
	if len(positions) != len(sprites) {
		panic("positions and sprites must have equal lengths")
	}

	for firstRockIndex := 0; firstRockIndex < len(positions); firstRockIndex += SIMDVectorSize {
		packedPositionLanes, loadedRockCount := simd.LoadUint32sPart(positions[firstRockIndex:])
		packedSpriteLanes, _ := simd.LoadUint32sPart(sprites[firstRockIndex:])

		rockSize, spriteRotation, spriteSlopeX, spriteSlopeY := unpackSprites(packedSpriteLanes)
		positionX, positionY, slopeX, slopeY := unpackPositions(packedPositionLanes)

		// Zero out everything if positionX or positionY is 0.
		// 0 means dead rock
		// TODO: show 0001_000_000 example
		rockWasVisible := positionX.NotEqual(zeroCoordinates).
			And(positionY.NotEqual(zeroCoordinates))

		positionX = positionX.ConvertToInt32().Add(slopeX).ConvertToUint32()
		positionY = positionY.ConvertToInt32().Add(slopeY).ConvertToUint32()

		spriteSlopeX = trailSpriteSlope(spriteSlopeX, slopeX)
		spriteSlopeY = trailSpriteSlope(spriteSlopeY, slopeY)

		// FIXME: clamp, will need to think about bc of cetnerin
		// if positionX is beyond maximumPixelX, set it to maximumPixelX or 1,
		//	set xSlope to .Neg.
		// same for positionY, maximumPixelY, and ySlope

		updatedPackedPositions := packPositions(
			positionX,
			positionY,
			slopeX,
			slopeY,
		)
		// updatedPackedPositions = updatedPackedPositions.Masked(rockWasVisible)

		updatedPackedSprites := packSprites(
			rockSize,
			spriteRotation,
			spriteSlopeX,
			spriteSlopeY,
		)
		updatedPackedSprites = updatedPackedSprites.Masked(rockWasVisible)

		// StorePart writes only the lanes loaded for this batch. For a short
		// final batch, padded zero lanes are not written past the slice end.
		updatedPackedPositions.StorePart(
			positions[firstRockIndex : firstRockIndex+loadedRockCount],
		)
		updatedPackedSprites.StorePart(
			sprites[firstRockIndex : firstRockIndex+loadedRockCount],
		)
	}
}
