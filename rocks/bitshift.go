package rocks

import (
	"simd"
)

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
	velocityXBits := velocityX.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(28)
	velocityYBits := velocityY.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(24)
	return velocityXBits.Or(velocityYBits)
}

func packSlopesSIMD(slopeX, slopeY simd.Int32s) simd.Uint32s {
	slopeXBits := slopeX.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(28)
	slopeYBits := slopeY.Add(slopeOffset).ConvertToUint32().ShiftAllLeft(24)
	return slopeXBits.Or(slopeYBits)
}

func packPositionsSIMD(positionX, positionY simd.Uint32s, velocityX, velocityY simd.Int32s) simd.Uint32s {
	positionXBits := positionX.ShiftAllLeft(12)
	positionYBits := positionY
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
//
