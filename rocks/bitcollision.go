package rocks

import "simd"

func collideWall(
	position simd.Uint32s,
	velocity simd.Int32s,
	extent simd.Uint32s,
	radius simd.Uint32s,
) (simd.Uint32s, simd.Uint32s, simd.Mask32s) {
	maximumPosition := extent.Sub(oneCoordinates)
	movingNegative := velocity.Less(zeroSlope)
	movingPositive := velocity.Greater(zeroSlope)
	rolledBelowMinimum := position.Greater(maximumPositionCoordinate)
	belowMinimum := position.Less(oneCoordinates).Or(rolledBelowMinimum)
	pastMaximum := position.GreaterEqual(extent).
		And(position.LessEqual(maximumPositionCoordinate))

	minimumOverflow := movingNegative.And(belowMinimum)
	maximumOverflow := movingPositive.And(pastMaximum)
	hitMinimumRadius := movingNegative.And(position.Less(radius))
	hitMaximumRadius := movingPositive.And(position.Add(radius).GreaterEqual(extent))
	hitWall := minimumOverflow.Or(maximumOverflow).
		Or(hitMinimumRadius).Or(hitMaximumRadius)
	minimumOverlap := radius.Sub(position).Masked(hitMinimumRadius)
	maximumRadiusOverlap := position.Add(radius).Sub(maximumPosition).Masked(hitMaximumRadius)
	overlap := minimumOverlap.Or(maximumRadiusOverlap)
	overlap = maximumOverlap.IfElse(minimumOverflow.Or(maximumOverflow), overlap)

	position = oneCoordinates.IfElse(belowMinimum, position)
	position = maximumPosition.IfElse(pastMaximum, position)

	return position, overlap, hitWall
}

func collideWalls(
	positionX, positionY simd.Uint32s,
	velocityX, velocityY simd.Int32s,
	radius simd.Uint32s,
) (
	simd.Uint32s, simd.Uint32s,
	simd.Int32s, simd.Int32s,
	simd.Mask32s, simd.Mask32s,
) {
	positionX, overlapX, candidateX := collideWall(positionX, velocityX, screenWidth, radius)
	positionY, overlapY, candidateY := collideWall(positionY, velocityY, screenHeight, radius)
	hitX := candidateX.And(overlapX.GreaterEqual(overlapY))
	hitY := candidateY.And(overlapY.Greater(overlapX))
	velocityX = velocityX.Neg().IfElse(hitX, velocityX)
	velocityY = velocityY.Neg().IfElse(hitY, velocityY)
	return positionX, positionY, velocityX, velocityY, hitX, hitY
}

func collisionRadius(
	sizeScore simd.Uint32s,
	amountScale int,
) simd.Uint32s {
	radius := zeroCoordinates
	for score := 1; score < BitSpriteSlopeCodeCount; score++ {
		radius = collisionLookups[amountScale][score].IfElse(
			sizeScore.Equal(sizeScoreLanes[score]), radius)
	}
	return radius
}

func hoverRadius(sizeScore simd.Uint32s, amountScale int) simd.Float32s {
	radius := zeroFloat
	for score := 1; score < BitSpriteSlopeCodeCount; score++ {
		radius = hoverRadiusLookups[amountScale][score].IfElse(
			sizeScore.Equal(sizeScoreLanes[score]), radius)
	}
	return radius
}

func setMouseVelocity(
	positionX, positionY simd.Uint32s,
	velocityX, velocityY simd.Int32s,
	rockRadius simd.Float32s,
	mouseDown bool,
	mouseX, mouseY simd.Int32s,
) (simd.Int32s, simd.Int32s, simd.Mask32s) {
	radius := rockRadius
	if mouseDown {
		radius = mouseRadius.ConvertToInt32().ConvertToFloat32()
	}
	distanceX := positionX.ConvertToInt32().Sub(mouseX)
	distanceY := positionY.ConvertToInt32().Sub(mouseY)
	absDistanceX := distanceX.Abs().ConvertToFloat32()
	absDistanceY := distanceY.Abs().ConvertToFloat32()
	withinBounds := absDistanceX.LessEqual(radius).
		And(absDistanceY.LessEqual(radius))

	distanceXFloat := distanceX.Masked(withinBounds).ConvertToFloat32()
	distanceYFloat := distanceY.Masked(withinBounds).ConvertToFloat32()
	distanceSquared := distanceXFloat.Mul(distanceXFloat).
		Add(distanceYFloat.Mul(distanceYFloat))
	hit := withinBounds.
		And(distanceSquared.LessEqual(radius.Mul(radius)))

	distance := distanceSquared.Sqrt()
	falloff := distance.Div(radius).Mul(mouseVelocityFalloff)
	impulse := maximumSlope.Sub(falloff.ConvertToInt32())
	impulse = oneSlope.IfElse(impulse.Less(oneSlope), impulse)
	impulseX := impulse.IfElse(distanceX.Greater(zeroSlope), impulse.Neg()).
		Masked(distanceX.NotEqual(zeroSlope))
	impulseY := impulse.IfElse(distanceY.Greater(zeroSlope), impulse.Neg()).
		Masked(distanceY.NotEqual(zeroSlope))
	velocityX = impulseX.IfElse(hit, velocityX)
	velocityY = impulseY.IfElse(hit, velocityY)
	return velocityX, velocityY, hit
}
