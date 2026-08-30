package rocks

import "simd"

const (
	mouseModeDisabled uint8 = iota
	mouseModeHover
	mouseModeLeftDown
	mouseModeRightDown
	mouseModeBothDown
)

func collisionRadius(size simd.Uint16s, amountScale int) simd.Uint16s {
	// These affine forms exactly reproduce the atlas's ceil-scaled radii for
	// every size score while avoiding a 15-way vector lookup.
	value := size.Mul(collisionRadiusMultiplier[amountScale]).Add(collisionRadiusBias[amountScale])
	switch amountScale {
	case 0, 3:
		value = value.ShiftAllRight(4)
	case 1, 2, 4:
		value = value.ShiftAllRight(5)
	case 5:
		value = value.ShiftAllRight(2)
	}
	return value.Masked(size.NotEqual(zeroUint16))
}

func hoverRadius(size simd.Uint16s, amountScale int) simd.Uint16s {
	value := size.Mul(hoverRadiusMultiplier[amountScale]).Add(hoverRadiusBias[amountScale])
	switch amountScale {
	case 0, 1, 2:
		value = value.ShiftAllRight(4)
	case 3:
		value = value.ShiftAllRight(5)
	case 4:
		value = value.ShiftAllRight(6)
	case 5:
		value = value.ShiftAllRight(3)
	}
	return value.Masked(size.NotEqual(zeroUint16))
}

func setMouseVelocity16(
	positionX, positionY simd.Int16s,
	velocityX, velocityY simd.Int16s,
	size simd.Uint16s,
	amountScale int,
	mode uint8,
	mouseX, mouseY simd.Int16s,
) (simd.Int16s, simd.Int16s, simd.Mask16s) {
	deltaX := positionX.Sub(mouseX)
	deltaY := positionY.Sub(mouseY)
	absX := deltaX.Abs().ToBits()
	absY := deltaY.Abs().ToBits()
	broadRadius := mouseRadii[amountScale]
	broadHit := absX.LessEqual(broadRadius).And(absY.LessEqual(broadRadius)).
		And(broadRadius.NotEqual(zeroUint16))

	hit := broadHit
	interactionRadius := broadRadius
	if mode == mouseModeHover {
		interactionRadius = hoverRadius(size, amountScale)
		distanceSquared := deltaX.Mul(deltaX).ToBits().Add(deltaY.Mul(deltaY).ToBits())
		hit = hit.And(distanceSquared.LessEqual(interactionRadius.Mul(interactionRadius)))
		hit = hit.And(interactionRadius.NotEqual(zeroUint16))
	}

	magnitudeX := closeMouseMagnitude16(absX, interactionRadius)
	magnitudeY := closeMouseMagnitude16(absY, interactionRadius)
	if mode == mouseModeLeftDown {
		magnitudeX = nibbleValues16[8].Sub(magnitudeX)
		magnitudeY = nibbleValues16[8].Sub(magnitudeY)
	}
	impulseX := signedMouseMagnitude16(magnitudeX, deltaX)
	impulseY := signedMouseMagnitude16(magnitudeY, deltaY)
	if mode == mouseModeBothDown {
		impulseX = impulseX.Neg()
		impulseY = impulseY.Neg()
	}
	return impulseX.IfElse(hit, velocityX), impulseY.IfElse(hit, velocityY), hit
}

func closeMouseMagnitude16(distance, radius simd.Uint16s) simd.Uint16s {
	scaledDistance := distance.Mul(nibbleValues16[7])
	magnitude := nibbleValues16[7]
	for band := 1; band <= 6; band++ {
		threshold := radius.Mul(nibbleValues16[band])
		magnitude = nibbleValues16[7-band].IfElse(scaledDistance.GreaterEqual(threshold), magnitude)
	}
	return magnitude
}

func signedMouseMagnitude16(magnitude simd.Uint16s, delta simd.Int16s) simd.Int16s {
	signed := magnitude.BitsToInt16()
	return signed.IfElse(delta.Greater(zeroInt16), signed.Neg()).Masked(delta.NotEqual(zeroInt16))
}

func collideWall16(position, velocity, extent, radius simd.Int16s) (simd.Int16s, simd.Mask16s) {
	maximumPosition := extent.Sub(radius)
	projected := position.Add(velocity)
	hitMinimum := projected.Less(radius).And(velocity.Less(zeroInt16))
	hitMaximum := projected.Greater(maximumPosition).And(velocity.Greater(zeroInt16))
	overlap := radius.Sub(projected).Masked(hitMinimum).
		Or(projected.Sub(maximumPosition).Masked(hitMaximum))
	return overlap, hitMinimum.Or(hitMaximum)
}

func collideWalls16(
	positionX, positionY, velocityX, velocityY, radius simd.Int16s,
	bounce simd.Mask16s,
) (
	simd.Int16s, simd.Int16s, simd.Mask16s,
) {
	overlapX, candidateX := collideWall16(positionX, velocityX, screenWidth16, radius)
	overlapY, candidateY := collideWall16(positionY, velocityY, screenHeight16, radius)
	hitX := candidateX.And(overlapX.GreaterEqual(overlapY))
	hitY := candidateY.And(overlapY.Greater(overlapX))
	velocityX = velocityX.Neg().IfElse(hitX.And(bounce), velocityX)
	velocityY = velocityY.Neg().IfElse(hitY.And(bounce), velocityY)
	return velocityX, velocityY, hitX.Or(hitY)
}

func slopeDistance16(slope, velocity simd.Int16s) simd.Uint16s {
	distance := velocity.Sub(slope).Abs().ToBits()
	return distance.Min(slopeCycleU16.Sub(distance))
}

func slopeDirection16(slope, velocity simd.Int16s) (simd.Mask16s, simd.Mask16s) {
	forward := velocity.Sub(slope)
	forward = forward.Add(slopeCycleI16).IfElse(forward.Less(zeroInt16), forward)
	return forward.Greater(zeroInt16).And(forward.LessEqual(halfSlopeCycleI16)),
		forward.Greater(halfSlopeCycleI16)
}

func scheduleCollision16(
	stepX, stepY, stepZ, permaSpin, spinAgain simd.Uint16s,
	slopeX, slopeY, velocityX, velocityY, previousX, previousY simd.Int16s,
	impact, wallHit simd.Mask16s,
	mouseHit simd.Mask16s,
) (simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Int16s, simd.Int16s, simd.Mask16s) {
	changedX := velocityX.NotEqual(previousX)
	changedY := velocityY.NotEqual(previousY)
	changed := changedX.Or(changedY)
	inactive := permaSpin.Equal(zeroUint16)
	active := permaSpin.NotEqual(zeroUint16)

	stepX = slopeDistance16(slopeX, velocityX).IfElse(changedX, stepX)
	stepY = slopeDistance16(slopeY, velocityY).IfElse(changedY, stepY)
	stepX = zeroUint16.IfElse(active.And(changedY), stepX)
	stepY = zeroUint16.IfElse(active.And(changedX), stepY)
	incX, decX := slopeDirection16(slopeX, velocityX)
	incY, decY := slopeDirection16(slopeY, velocityY)
	stepZ = oneUint16.Masked(decX).Or(nibbleValues16[2].Masked(incX)).
		IfElse(active.And(changedX), stepZ)
	stepZ = nibbleValues16[4].Masked(decY).Or(nibbleValues16[8].Masked(incY)).
		IfElse(active.And(changedY), stepZ)
	spinAgain = oneUint16.IfElse(changed, spinAgain)

	// Repeated contact may keep requesting another spin, but it must not reset
	// the active finite animation unless the resulting velocity changed.
	changedImpact := impact.And(changed)
	finiteImpact := changedImpact.And(inactive)
	flipX := velocityX.Sub(slopeX).Abs().ToBits().Masked(changedX)
	flipY := velocityY.Sub(slopeY).Abs().ToBits().Masked(changedY)
	collisionStepZ := flipX.Add(flipY)
	collisionStepZ = maximumStepZ16.IfElse(collisionStepZ.Greater(maximumStepZ16), collisionStepZ)
	stepZ = collisionStepZ.IfElse(finiteImpact, stepZ)
	slopeX = velocityX.IfElse(changedImpact, slopeX)
	slopeY = velocityY.IfElse(changedImpact, slopeY)
	spinAgain = zeroUint16.IfElse(finiteImpact, spinAgain).Masked(inactive)

	// Mouse and wall contact independently queue another finite spin. Repeated
	// contact leaves an active stepZ untouched and only keeps this bit queued.
	spinAgain = oneUint16.IfElse(wallHit.And(inactive), spinAgain)
	spinAgain = oneUint16.IfElse(mouseHit.And(inactive), spinAgain)
	return stepX, stepY, stepZ, spinAgain, slopeX, slopeY, changed
}
