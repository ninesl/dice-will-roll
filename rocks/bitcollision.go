package rocks

import "simd"

func collideWallAxis(
	position simd.Uint32s,
	slope simd.Int32s,
	maximum simd.Uint32s,
) (simd.Uint32s, simd.Int32s, simd.Mask32s) {
	maximum = maximum.Sub(oneCoordinates)
	pastMaximum := position.Greater(maximum)

	hitMinimum := slope.Less(zeroSlope).
		And(position.Less(oneCoordinates).Or(pastMaximum))
	hitMaximum := slope.Greater(zeroSlope).And(pastMaximum)
	hitWall := hitMinimum.Or(hitMaximum)

	position = oneCoordinates.IfElse(hitMinimum, position)
	position = maximum.IfElse(hitMaximum, position)
	slope = slope.Neg().IfElse(hitWall, slope)

	return position, slope, hitWall
}

func collideWalls(
	positionX, positionY simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) (
	simd.Uint32s, simd.Uint32s,
	simd.Int32s, simd.Int32s,
	simd.Mask32s, simd.Mask32s,
) {
	positionX, slopeX, hitX := collideWallAxis(positionX, slopeX, maximumPixelX)
	positionY, slopeY, hitY := collideWallAxis(positionY, slopeY, maximumPixelY)
	return positionX, positionY, slopeX, slopeY, hitX, hitY
}
