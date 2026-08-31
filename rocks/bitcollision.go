package rocks

import "simd"

func collisionRadius(size simd.Uint16s, constants *rockUpdateConstants16) simd.Uint16s {
	// These affine forms exactly reproduce the atlas's ceil-scaled radii for
	// every size score while avoiding a 15-way vector lookup.
	value := size.Mul(constants.collisionMultiplier).Add(constants.collisionBias).
		ShiftAllRight(collisionRadiusShift)
	return value.Masked(size.NotEqual(zeroUint16))
}

func hoverRadius(size simd.Uint16s, constants *rockUpdateConstants16) simd.Uint16s {
	value := size.Mul(constants.hoverMultiplier).Add(constants.hoverBias).
		ShiftAllRight(hoverRadiusShift)
	return value.Masked(size.NotEqual(zeroUint16))
}

func setHoverVelocity16(
	positionX, positionY simd.Int16s,
	velocityX, velocityY simd.Int16s,
	size simd.Uint16s,
	constants *rockUpdateConstants16,
	mouseX, mouseY simd.Int16s,
) (simd.Int16s, simd.Int16s, simd.Mask16s) {
	deltaX := positionX.Sub(mouseX)
	deltaY := positionY.Sub(mouseY)
	absX := deltaX.Abs().ToBits()
	absY := deltaY.Abs().ToBits()
	broadRadius := constants.mouseRadius
	broadHit := absX.LessEqual(broadRadius).And(absY.LessEqual(broadRadius)).
		And(broadRadius.NotEqual(zeroUint16))

	interactionRadius := hoverRadius(size, constants)
	distanceSquared := deltaX.Mul(deltaX).ToBits().Add(deltaY.Mul(deltaY).ToBits())
	hit := broadHit.And(distanceSquared.LessEqual(interactionRadius.Mul(interactionRadius))).
		And(interactionRadius.NotEqual(zeroUint16))
	magnitudeX := closeHoverMagnitude16(absX, interactionRadius)
	magnitudeY := closeHoverMagnitude16(absY, interactionRadius)
	impulseX := signedMouseMagnitude16(magnitudeX, deltaX)
	impulseY := signedMouseMagnitude16(magnitudeY, deltaY)
	return impulseX.IfElse(hit, velocityX), impulseY.IfElse(hit, velocityY), hit
}

func setButtonVelocity16(
	positionX, positionY simd.Int16s,
	velocityX, velocityY simd.Int16s,
	constants *rockUpdateConstants16,
	mouseX, mouseY simd.Int16s,
	farForce, attract simd.Mask16s,
) (simd.Int16s, simd.Int16s, simd.Mask16s) {
	deltaX := positionX.Sub(mouseX)
	deltaY := positionY.Sub(mouseY)
	absX := deltaX.Abs().ToBits()
	absY := deltaY.Abs().ToBits()
	radius := constants.mouseRadius
	hit := absX.LessEqual(radius).And(absY.LessEqual(radius)).
		And(radius.NotEqual(zeroUint16))
	magnitudeX := closeButtonMagnitude16(absX, &constants.buttonThresholds)
	magnitudeY := closeButtonMagnitude16(absY, &constants.buttonThresholds)
	magnitudeX = nibbleValues16[8].Sub(magnitudeX).IfElse(farForce, magnitudeX)
	magnitudeY = nibbleValues16[8].Sub(magnitudeY).IfElse(farForce, magnitudeY)
	impulseX := signedMouseMagnitude16(magnitudeX, deltaX)
	impulseY := signedMouseMagnitude16(magnitudeY, deltaY)
	impulseX = impulseX.Neg().IfElse(attract, impulseX)
	impulseY = impulseY.Neg().IfElse(attract, impulseY)
	return impulseX.IfElse(hit, velocityX), impulseY.IfElse(hit, velocityY), hit
}

func closeHoverMagnitude16(distance, radius simd.Uint16s) simd.Uint16s {
	// Seven force values have six transitions. Keep these explicit: this avoids
	// a hot-loop branch and expensive blends. A true mask converts to 0xffff,
	// so adding it decrements the unsigned magnitude by one.
	scaledDistance := distance.Mul(nibbleValues16[7])
	magnitude := nibbleValues16[7]
	magnitude = magnitude.Add(scaledDistance.GreaterEqual(radius).ToInt16s().ToBits())
	magnitude = magnitude.Add(scaledDistance.GreaterEqual(radius.Mul(nibbleValues16[2])).ToInt16s().ToBits())
	magnitude = magnitude.Add(scaledDistance.GreaterEqual(radius.Mul(nibbleValues16[3])).ToInt16s().ToBits())
	magnitude = magnitude.Add(scaledDistance.GreaterEqual(radius.Mul(nibbleValues16[4])).ToInt16s().ToBits())
	magnitude = magnitude.Add(scaledDistance.GreaterEqual(radius.Mul(nibbleValues16[5])).ToInt16s().ToBits())
	return magnitude.Add(scaledDistance.GreaterEqual(radius.Mul(nibbleValues16[6])).ToInt16s().ToBits())
}

func closeButtonMagnitude16(
	distance simd.Uint16s,
	thresholds *[mouseForceBands - 1]simd.Uint16s,
) simd.Uint16s {
	// Button radii are uniform for a scale tier, so startup converts the same
	// seven force bands into six exact integer distance thresholds.
	magnitude := nibbleValues16[7]
	magnitude = magnitude.Add(distance.GreaterEqual(thresholds[0]).ToInt16s().ToBits())
	magnitude = magnitude.Add(distance.GreaterEqual(thresholds[1]).ToInt16s().ToBits())
	magnitude = magnitude.Add(distance.GreaterEqual(thresholds[2]).ToInt16s().ToBits())
	magnitude = magnitude.Add(distance.GreaterEqual(thresholds[3]).ToInt16s().ToBits())
	magnitude = magnitude.Add(distance.GreaterEqual(thresholds[4]).ToInt16s().ToBits())
	return magnitude.Add(distance.GreaterEqual(thresholds[5]).ToInt16s().ToBits())
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
	stepX, stepY, stepZ, forceStepping, stepping simd.Uint16s,
	slopeX, slopeY, velocityX, velocityY, previousX, previousY simd.Int16s,
	impact, wallHit simd.Mask16s,
	mouseHit simd.Mask16s,
) (simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Int16s, simd.Int16s, simd.Mask16s) {
	changedX := velocityX.NotEqual(previousX)
	changedY := velocityY.NotEqual(previousY)
	changed := changedX.Or(changedY)
	inactive := forceStepping.Equal(zeroUint16)
	active := forceStepping.NotEqual(zeroUint16)

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
	stepping = oneUint16.IfElse(changed, stepping)

	// Repeated contact may keep requesting another step, but it must not reset
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
	stepping = zeroUint16.IfElse(finiteImpact, stepping).Masked(inactive)

	// Mouse and wall contact independently queue another finite step. Repeated
	// contact leaves an active stepZ untouched and only keeps this bit queued.
	stepping = oneUint16.IfElse(wallHit.And(inactive), stepping)
	stepping = oneUint16.IfElse(mouseHit.And(inactive), stepping)
	return stepX, stepY, stepZ, stepping, slopeX, slopeY, changed
}
