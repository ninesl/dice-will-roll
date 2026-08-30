package rocks

func PackPosition(positionX, positionY uint32, velocityX, velocityY int32) (uint16, uint16) {
	return packPositionAxis(positionX, velocityX), packPositionAxis(positionY, velocityY)
}

func packPositionAxis(position uint32, velocity int32) uint16 {
	return uint16(position)&packedCoordinateMask | (uint16(velocity)&0xF)<<12
}

func UnpackPosition(posX, posY uint16) (positionX, positionY, velocityX, velocityY int) {
	return int(posX & packedCoordinateMask), int(posY & packedCoordinateMask),
		unpackSignedNibble(posX, 12), unpackSignedNibble(posY, 12)
}

func unpackSignedNibble(packed uint16, shift uint8) int {
	return int(int8(uint8(packed>>shift&0xF)<<4) >> 4)
}

func PackSprite(slopeX, slopeY int32,
	slopeZ, sizeScore, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain uint32,
) (uint16, uint16) {
	stepZ &= 0xF
	spinAgain &= 1
	if permaSpin&1 != 0 {
		stepZ = canonicalPermaStepZScalar(stepZ)
		spinAgain = 0
	}
	return uint16(slopeX)&0xF<<12 | uint16(slopeY)&0xF<<8 |
			uint16(slopeZ&0xF)<<4 | uint16(sizeScore&0xF),
		uint16(stepY&7)<<13 | uint16(stepX&7)<<10 | uint16(stepZ)<<6 |
			uint16(stepTick&0xF)<<2 | uint16(permaSpin&1)<<1 | uint16(spinAgain)
}

func UnpackSprite(slope, animate uint16) (
	sizeScore, slopeZ, stepX, stepY, stepZ, stepTick,
	permaSpin, spinAgain, slopeX, slopeY int,
) {
	sizeScore = int(slope & 0xF)
	slopeZ = int(slope >> 4 & 0xF)
	slopeX = unpackSignedNibble(slope, 12)
	slopeY = unpackSignedNibble(slope, 8)
	stepY = int(animate >> 13 & 7)
	stepX = int(animate >> 10 & 7)
	stepZ = int(animate >> 6 & 0xF)
	stepTick = int(animate >> 2 & 0xF)
	permaSpin = int(animate >> 1 & 1)
	spinAgain = int(animate & 1)
	if permaSpin != 0 {
		stepZ = int(canonicalPermaStepZScalar(uint32(stepZ)))
		spinAgain = 0
	}
	return
}

func canonicalPermaStepZScalar(stepZ uint32) uint32 {
	if stepZ == 0xF {
		return stepZ
	}
	xDirection := stepZ & permaXDirectionMask32
	yDirection := stepZ & permaYDirectionMask32
	if xDirection == permaXDirectionMask32 {
		xDirection = 0
	}
	if yDirection == permaYDirectionMask32 || xDirection != 0 {
		yDirection = 0
	}
	return xDirection | yDirection
}
