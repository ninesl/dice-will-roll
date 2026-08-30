package rocks

import "simd"

const (
	mouseRadiusDisabled uint8 = iota
	mouseRadiusHover
	mouseRadiusDown
)

func collisionRadius(size simd.Uint16s, amountScale int) simd.Uint16s {
	radius := zeroUint16
	for score := 1; score < BitSpriteSlopeCodeCount; score++ {
		radius = collisionLookups[amountScale][score].IfElse(
			size.Equal(nibbleValues16[score]), radius)
	}
	return radius
}

func hoverRadius(size simd.Uint16s, amountScale int) simd.Uint16s {
	radius := zeroUint16
	for score := 1; score < BitSpriteSlopeCodeCount; score++ {
		radius = hoverRadiusLookups[amountScale][score].IfElse(
			size.Equal(nibbleValues16[score]), radius)
	}
	return radius
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

	distanceSquared := deltaX.Mul(deltaX).ToBits().Add(deltaY.Mul(deltaY).ToBits())
	hit := broadHit
	interactionRadius := broadRadius
	if mode == mouseRadiusHover {
		interactionRadius = hoverRadius(size, amountScale)
		hit = hit.And(distanceSquared.LessEqual(interactionRadius.Mul(interactionRadius)))
		hit = hit.And(interactionRadius.NotEqual(zeroUint16))
	}

	falloff := radialFalloff16(distanceSquared, interactionRadius.Mul(interactionRadius))
	impulse := maximumSlopeU16.Sub(falloff).BitsToInt16()
	impulseX := impulse.IfElse(deltaX.Greater(zeroInt16), impulse.Neg()).Masked(deltaX.NotEqual(zeroInt16))
	impulseY := impulse.IfElse(deltaY.Greater(zeroInt16), impulse.Neg()).Masked(deltaY.NotEqual(zeroInt16))
	return impulseX.IfElse(hit, velocityX), impulseY.IfElse(hit, velocityY), hit
}

func radialFalloff16(distanceSquared, radiusSquared simd.Uint16s) simd.Uint16s {
	falloff := oneUint16.IfElse(distanceSquared.GreaterEqual(radiusSquared.ShiftAllRight(5)), zeroUint16)
	falloff = nibbleValues16[2].IfElse(distanceSquared.GreaterEqual(radiusSquared.ShiftAllRight(3)), falloff)
	falloff = nibbleValues16[3].IfElse(distanceSquared.GreaterEqual(radiusSquared.ShiftAllRight(2)), falloff)
	falloff = nibbleValues16[4].IfElse(distanceSquared.GreaterEqual(radiusSquared.ShiftAllRight(1)), falloff)
	threeQuarters := radiusSquared.ShiftAllRight(1).Add(radiusSquared.ShiftAllRight(2))
	falloff = nibbleValues16[5].IfElse(distanceSquared.GreaterEqual(threeQuarters), falloff)
	return nibbleValues16[6].IfElse(distanceSquared.GreaterEqual(radiusSquared), falloff)
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
	mouseDown bool,
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

	// A wall impact always requests another finite spin, but permanent-spin
	// rocks keep their canonical animation state.
	spinAgain = oneUint16.IfElse(wallHit.And(inactive), spinAgain)
	if mouseDown {
		spinAgain = oneUint16.IfElse(mouseHit.And(inactive), spinAgain)
	}
	return stepX, stepY, stepZ, spinAgain, slopeX, slopeY, changed
}
