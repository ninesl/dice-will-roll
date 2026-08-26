package rocks

import "simd"

var debugRotationFrameCount simd.Uint32s

// UpdateDEBUG advances packed rocks through every generated sprite frame.
//
// it does not update positions, but requires them for behavior
func UpdateDEBUG(positions RockPositions, sprites RockSprites) {
	if len(positions) != len(sprites) {
		panic("positions and sprites must have equal lengths")
	}

	for startIndex := 0; startIndex < len(positions); startIndex += SIMDVectorSize {
		packedSprite, loadedRockCount := simd.LoadUint32sPart(sprites[startIndex:])
		rockSize, rotation, spriteSlopeX, spriteSlopeY := unpackSprites(packedSprite)

		rotation = rotation.Add(oneCoordinates)
		rotationWrapped := rotation.GreaterEqual(debugRotationFrameCount)
		rotation = zeroCoordinates.IfElse(rotationWrapped, rotation)

		xWrapped := rotationWrapped.And(spriteSlopeX.Equal(maximumSlope))
		spriteSlopeX = incrementSpriteSlope(spriteSlopeX).IfElse(rotationWrapped, spriteSlopeX)
		spriteSlopeY = incrementSpriteSlope(spriteSlopeY).IfElse(xWrapped, spriteSlopeY)

		packSprites(
			rockSize, rotation,
			spriteSlopeX, spriteSlopeY,
		).StorePart(sprites[startIndex : startIndex+loadedRockCount])
	}
}
