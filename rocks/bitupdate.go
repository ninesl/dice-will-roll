package rocks

import (
	"simd"

	"github.com/ninesl/dice-will-roll/controls"
)

const UpdateStride = 2

func UpdateRocks(rocks *Rocks, amountScale, phase int, mouse controls.MouseInfo) {
	mode := mouseModeHover
	if mouse.Down && mouse.RightDown {
		mode = mouseModeBothDown
	} else if mouse.Down {
		mode = mouseModeLeftDown
	} else if mouse.RightDown {
		mode = mouseModeRightDown
	}
	if !mouse.Active && mouse.Position.X == 0 && mouse.Position.Y == 0 && !mouse.Down && !mouse.RightDown {
		mode = mouseModeDisabled
	}
	groups := (rocks.Len() + RocksPerPackedVector - 1) / RocksPerPackedVector
	start := min(rocks.Len(), groups*phase/UpdateStride*RocksPerPackedVector)
	end := min(rocks.Len(), groups*(phase+1)/UpdateStride*RocksPerPackedVector)
	updateMovement16(rocks, start, end, amountScale, mode,
		simd.BroadcastInt16s(int16(mouse.Position.X)), simd.BroadcastInt16s(int16(mouse.Position.Y)))
}

func updateMovement16(
	rocks *Rocks,
	start, end, amountScale int,
	mode uint8,
	mouseX, mouseY simd.Int16s,
) {
	fullEnd := end - (end-start)%RocksPerPackedVector
	for i := start; i < fullEnd; i += RocksPerPackedVector {
		positionX := simd.LoadUint16s(rocks.PosX[i:])
		positionY := simd.LoadUint16s(rocks.PosY[i:])
		slope := simd.LoadUint16s(rocks.Slope[i:])
		animate := simd.LoadUint16s(rocks.Animate[i:])
		positionX, positionY, slope, animate = updateRockGroup16(
			positionX, positionY, slope, animate, amountScale, mode, mouseX, mouseY)
		positionX.Store(rocks.PosX[i:])
		positionY.Store(rocks.PosY[i:])
		slope.Store(rocks.Slope[i:])
		animate.Store(rocks.Animate[i:])
	}
	if fullEnd == end {
		return
	}
	positionX, _ := simd.LoadUint16sPart(rocks.PosX[fullEnd:end])
	positionY, _ := simd.LoadUint16sPart(rocks.PosY[fullEnd:end])
	slope, _ := simd.LoadUint16sPart(rocks.Slope[fullEnd:end])
	animate, _ := simd.LoadUint16sPart(rocks.Animate[fullEnd:end])
	positionX, positionY, slope, animate = updateRockGroup16(
		positionX, positionY, slope, animate, amountScale, mode, mouseX, mouseY)
	positionX.StorePart(rocks.PosX[fullEnd:end])
	positionY.StorePart(rocks.PosY[fullEnd:end])
	slope.StorePart(rocks.Slope[fullEnd:end])
	animate.StorePart(rocks.Animate[fullEnd:end])
}

func updateRockGroup16(
	packedX, packedY, packedSlope, packedAnimate simd.Uint16s,
	amountScale int,
	mode uint8,
	mouseX, mouseY simd.Int16s,
) (simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s) {
	positionX := packedX.And(coordinateMask16).ConvertToInt16()
	positionY := packedY.And(coordinateMask16).ConvertToInt16()
	velocityX := packedX.BitsToInt16().ShiftAllRight(12)
	velocityY := packedY.BitsToInt16().ShiftAllRight(12)
	previousX, previousY := velocityX, velocityY

	size := packedSlope.And(nibbleMask16)
	slopeZ := packedSlope.ShiftAllRight(4).And(nibbleMask16)
	slopeX := packedSlope.BitsToInt16().ShiftAllRight(12)
	slopeY := packedSlope.ShiftAllLeft(4).BitsToInt16().ShiftAllRight(12)
	stepY := packedAnimate.ShiftAllRight(13).And(nibbleValues16[7])
	stepX := packedAnimate.ShiftAllRight(10).And(nibbleValues16[7])
	stepZ := packedAnimate.ShiftAllRight(6).And(nibbleMask16)
	stepTick := packedAnimate.ShiftAllRight(2).And(nibbleMask16)
	permaSpin := packedAnimate.ShiftAllRight(1).And(oneUint16)
	spinAgain := packedAnimate.And(oneUint16)

	falseMask := zeroUint16.NotEqual(zeroUint16)
	mouseHit := falseMask
	if mode != mouseModeDisabled {
		velocityX, velocityY, mouseHit = setMouseVelocity16(
			positionX, positionY, velocityX, velocityY, size,
			amountScale, mode, mouseX, mouseY)
	}

	radius := collisionRadius(size, amountScale).BitsToInt16()
	bounce := oneUint16.Equal(oneUint16)
	buttonDown := mode == mouseModeLeftDown || mode == mouseModeRightDown || mode == mouseModeBothDown
	if buttonDown {
		// A rock actively pushed into a wall keeps the mouse-assigned direction.
		// The wall still clamps it and drives the queued spin animation.
		bounce = mouseHit.ToInt16s().Equal(zeroInt16)
	}
	velocityX, velocityY, wallHit := collideWalls16(
		positionX, positionY, velocityX, velocityY, radius, bounce)
	impact := mouseHit.Or(wallHit)
	collisionVelocityX, collisionVelocityY := velocityX, velocityY

	// Collision velocities move once at full strength. Every other finite
	// velocity damps exactly once before this update's movement.
	damp := permaSpin.Equal(zeroUint16).And(impact.ToInt16s().Equal(zeroInt16))
	velocityX = dampVelocity16(velocityX).IfElse(damp, velocityX)
	velocityY = dampVelocity16(velocityY).IfElse(damp, velocityY)
	positionX = positionX.Add(velocityX)
	positionY = positionY.Add(velocityY)
	positionX = positionX.Max(radius).Min(screenWidth16.Sub(radius))
	positionY = positionY.Max(radius).Min(screenHeight16.Sub(radius))

	// Existing animation advances after movement and collision handling. New
	// collision state is scheduled afterward, so it starts on the next update.
	size, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain, slopeX, slopeY =
		advanceAnimation16(size, slopeZ, stepX, stepY, stepZ, stepTick,
			permaSpin, spinAgain, slopeX, slopeY, previousX, previousY)
	stopped := velocityX.Equal(zeroInt16).And(velocityY.Equal(zeroInt16)).
		And(permaSpin.Equal(zeroUint16)).And(stepZ.Equal(zeroUint16)).And(spinAgain.Equal(zeroUint16))
	stepX = zeroUint16.IfElse(stopped, stepX)
	stepY = zeroUint16.IfElse(stopped, stepY)
	stepTick = zeroUint16.IfElse(stopped, stepTick)
	stepX, stepY, stepZ, spinAgain, slopeX, slopeY, _ = scheduleCollision16(
		stepX, stepY, stepZ, permaSpin, spinAgain,
		slopeX, slopeY, collisionVelocityX, collisionVelocityY, previousX, previousY,
		impact, wallHit, mouseHit)

	return packPositionAxis16(positionX, velocityX), packPositionAxis16(positionY, velocityY),
		packSlope16(size, slopeZ, slopeX, slopeY),
		packAnimate16(stepX, stepY, stepZ, stepTick, permaSpin, spinAgain)
}

func advanceAnimation16(
	size, slopeZ, stepX, stepY, stepZ, tick, permaSpin, spinAgain simd.Uint16s,
	slopeX, slopeY, velocityX, velocityY simd.Int16s,
) (simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Int16s, simd.Int16s) {
	inactive := permaSpin.Equal(zeroUint16)
	active := permaSpin.NotEqual(zeroUint16)
	stepZ = canonicalPermaStepZ16(stepZ).IfElse(active, stepZ)
	spinAgain = spinAgain.Masked(inactive)
	hasAnimation := stepX.Or(stepY).Or(stepZ).Or(spinAgain).Or(permaSpin).NotEqual(zeroUint16)
	enabled := size.NotEqual(zeroUint16).And(hasAnimation)
	due := tick.Equal(zeroUint16).And(enabled)
	waiting := tick.NotEqual(zeroUint16).And(enabled)
	tick = tick.Sub(oneUint16.Masked(waiting))
	interval := size.AndNot(oneUint16).Average(zeroUint16).Max(oneUint16)
	tick = interval.Sub(oneUint16).IfElse(due, tick)

	finite := due.And(inactive)
	advanceX := stepX.NotEqual(zeroUint16).And(finite)
	advanceY := stepY.NotEqual(zeroUint16).And(finite)
	slopeX = moveSlope16(slopeX, velocityX, advanceX)
	slopeY = moveSlope16(slopeY, velocityY, advanceY)
	stepX = stepX.Sub(oneUint16.Masked(advanceX))
	stepY = stepY.Sub(oneUint16.Masked(advanceY))

	xCode := stepZ.And(nibbleValues16[3])
	yCode := stepZ.And(nibbleValues16[12])
	incrementX := xCode.Equal(oneUint16).And(active).And(due)
	decrementX := xCode.Equal(nibbleValues16[2]).And(active).And(due)
	incrementY := yCode.Equal(nibbleValues16[4]).And(active).And(due)
	decrementY := yCode.Equal(nibbleValues16[8]).And(active).And(due)
	slopeX = incrementSlope16(slopeX).IfElse(incrementX, slopeX)
	slopeX = decrementSlope16(slopeX).IfElse(decrementX, slopeX)
	slopeY = incrementSlope16(slopeY).IfElse(incrementY, slopeY)
	slopeY = decrementSlope16(slopeY).IfElse(decrementY, slopeY)
	stepX = stepX.Sub(oneUint16.Masked(incrementX.Or(decrementX).And(stepX.NotEqual(zeroUint16))))
	stepY = stepY.Sub(oneUint16.Masked(incrementY.Or(decrementY).And(stepY.NotEqual(zeroUint16))))

	finiteZ := stepZ.NotEqual(zeroUint16).And(due).And(inactive)
	startSpin := stepZ.Equal(zeroUint16).And(spinAgain.NotEqual(zeroUint16)).And(due).And(inactive)
	slopeZ = updateSlopeZ16(slopeZ, velocityX, velocityY).
		IfElse(finiteZ.Or(startSpin).Or(active.And(due)), slopeZ)
	stepZ = stepZ.Sub(oneUint16.Masked(finiteZ))
	rolloverSpin := stepZ.Equal(zeroUint16).And(spinAgain.NotEqual(zeroUint16)).And(due).And(inactive)
	stepZ = maximumStepZ16.IfElse(rolloverSpin, stepZ)
	spinAgain = zeroUint16.IfElse(rolloverSpin, spinAgain)
	return size, slopeZ, stepX, stepY, stepZ, tick, permaSpin, spinAgain, slopeX, slopeY
}

func canonicalPermaStepZ16(value simd.Uint16s) simd.Uint16s {
	full := value.Equal(maximumStepZ16)
	xDirection := value.And(nibbleValues16[3])
	yDirection := value.And(nibbleValues16[12])
	xDirection = zeroUint16.IfElse(xDirection.Equal(nibbleValues16[3]), xDirection)
	yDirection = zeroUint16.IfElse(yDirection.Equal(nibbleValues16[12]), yDirection)
	yDirection = zeroUint16.IfElse(xDirection.NotEqual(zeroUint16), yDirection)
	return maximumStepZ16.IfElse(full, xDirection.Or(yDirection))
}

func moveSlope16(slope, velocity simd.Int16s, mask simd.Mask16s) simd.Int16s {
	forward := velocity.Sub(slope)
	forward = forward.Add(slopeCycleI16).IfElse(forward.Less(zeroInt16), forward)
	increment := forward.Greater(zeroInt16).And(forward.LessEqual(halfSlopeCycleI16))
	slope = incrementSlope16(slope).IfElse(increment.And(mask), slope)
	return decrementSlope16(slope).IfElse(forward.Greater(halfSlopeCycleI16).And(mask), slope)
}

func incrementSlope16(value simd.Int16s) simd.Int16s {
	next := value.Add(oneInt16)
	return minimumSlopeI16.IfElse(next.Greater(maximumSlopeI16), next)
}

func decrementSlope16(value simd.Int16s) simd.Int16s {
	next := value.Sub(oneInt16)
	return maximumSlopeI16.IfElse(next.Less(minimumSlopeI16), next)
}

func updateSlopeZ16(slopeZ simd.Uint16s, velocityX, velocityY simd.Int16s) simd.Uint16s {
	direction := velocityX.IfElse(velocityX.Abs().GreaterEqual(velocityY.Abs()), velocityY)
	increment := slopeZ.Add(oneUint16)
	increment = zeroUint16.IfElse(increment.Equal(nibbleValues16[16]), increment)
	decrement := slopeZ.Sub(oneUint16)
	decrement = maximumStepZ16.IfElse(slopeZ.Equal(zeroUint16), decrement)
	slopeZ = increment.IfElse(direction.Less(zeroInt16), slopeZ)
	return decrement.IfElse(direction.Greater(zeroInt16), slopeZ)
}

func dampVelocity16(velocity simd.Int16s) simd.Int16s {
	velocity = velocity.Sub(oneInt16.Masked(velocity.Greater(zeroInt16)))
	return velocity.Add(oneInt16.Masked(velocity.Less(zeroInt16)))
}

func packPositionAxis16(position simd.Int16s, velocity simd.Int16s) simd.Uint16s {
	return position.ToBits().And(coordinateMask16).
		Or(velocity.ToBits().And(nibbleMask16).ShiftAllLeft(12))
}

func packSlope16(size, slopeZ simd.Uint16s, slopeX, slopeY simd.Int16s) simd.Uint16s {
	return slopeX.ToBits().And(nibbleMask16).ShiftAllLeft(12).
		Or(slopeY.ToBits().And(nibbleMask16).ShiftAllLeft(8)).
		Or(slopeZ.ShiftAllLeft(4)).Or(size)
}

func packAnimate16(stepX, stepY, stepZ, tick, permaSpin, spinAgain simd.Uint16s) simd.Uint16s {
	return stepY.ShiftAllLeft(13).Or(stepX.ShiftAllLeft(10)).Or(stepZ.ShiftAllLeft(6)).
		Or(tick.ShiftAllLeft(2)).Or(permaSpin.ShiftAllLeft(1)).Or(spinAgain)
}
