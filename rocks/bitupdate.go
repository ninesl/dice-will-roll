package rocks

import (
	"simd"

	"github.com/ninesl/dice-will-roll/controls"
)

const UpdateStride = 2

func UpdateRocks(rocks *Rocks, amountScale, phase int, mouse controls.MouseInfo) {
	prepareRockLayerDirtyState(rocks)
	input := prepareRockUpdateInput16(rocks, amountScale, mouse)
	groups := (rocks.Len() + RocksPerPackedVector - 1) / RocksPerPackedVector
	if cap(rocks.drawGroups) < groups {
		rocks.drawGroups = append(rocks.drawGroups, make([]bool, groups-len(rocks.drawGroups))...)
	} else {
		rocks.drawGroups = rocks.drawGroups[:groups]
	}
	start := min(rocks.Len(), groups*phase/UpdateStride*RocksPerPackedVector)
	end := min(rocks.Len(), groups*(phase+1)/UpdateStride*RocksPerPackedVector)
	runRockUpdateRoutine16(rocks, start, end, &input)
}

func prepareRockUpdateInput16(
	rocks *Rocks,
	amountScale int,
	mouse controls.MouseInfo,
) rockUpdateInput16 {
	state := &rocks.updateState
	positionX, positionY := int16(mouse.Position.X), int16(mouse.Position.Y)
	if !state.initialized || state.amountScale != amountScale {
		state.amountScale = amountScale
		rocks.redraw = true
	}
	if !state.initialized || state.rockCount != rocks.Len() {
		state.rockCount = rocks.Len()
		rocks.redraw = true
	}
	if !state.initialized || state.positionX != positionX || state.positionY != positionY {
		state.positionX, state.positionY = positionX, positionY
	}
	if !state.initialized || state.mouseActive != mouse.Active ||
		state.leftDown != mouse.Down || state.rightDown != mouse.RightDown {
		state.mouseActive = mouse.Active
		state.leftDown = mouse.Down
		state.rightDown = mouse.RightDown
		switch {
		case mouse.Down && mouse.RightDown:
			state.routine = rockUpdateButton
		case mouse.Down:
			state.routine = rockUpdateButton
		case mouse.RightDown:
			state.routine = rockUpdateButton
		case !mouse.Active && positionX == 0 && positionY == 0:
			state.routine = rockUpdateDisabled
		default:
			state.routine = rockUpdateHover
		}
	}
	state.initialized = true
	all := oneUint16.Equal(oneUint16)
	none := zeroUint16.NotEqual(zeroUint16)
	farForce, attract := none, none
	if state.leftDown && !state.rightDown {
		farForce = all
	} else if state.leftDown && state.rightDown {
		attract = all
	}
	return rockUpdateInput16{
		constants: &rockUpdateConstants[state.amountScale],
		routine:   state.routine,
		mouseX:    simd.BroadcastInt16s(state.positionX),
		mouseY:    simd.BroadcastInt16s(state.positionY),
		farForce:  farForce,
		attract:   attract,
	}
}

func runRockUpdateRoutine16(
	rocks *Rocks,
	start, end int,
	state *rockUpdateInput16,
) {
	fullEnd := end - (end-start)%RocksPerPackedVector
	for i := start; i < fullEnd; i += RocksPerPackedVector {
		positionX := simd.LoadUint16s(rocks.PosX[i:])
		positionY := simd.LoadUint16s(rocks.PosY[i:])
		stepping := simd.LoadUint16s(rocks.Stepping[i:])
		group := i / RocksPerPackedVector
		rocks.drawGroups[group] = rockGroupNeedsUpdate16(positionX, positionY, stepping,
			rocks.Stepping[i:i+RocksPerPackedVector], state)
		if !rocks.drawGroups[group] {
			continue
		}
		slope := simd.LoadUint16s(rocks.Slope[i:])
		oldPositionX, oldPositionY, oldSlope := positionX, positionY, slope
		positionX, positionY, slope, stepping = updateRockGroup16(
			positionX, positionY, slope, stepping, state)
		if !rocks.redraw {
			visualChange := positionX.And(coordinateMask16).NotEqual(oldPositionX.And(coordinateMask16)).
				Or(positionY.And(coordinateMask16).NotEqual(oldPositionY.And(coordinateMask16))).
				Or(slope.NotEqual(oldSlope))
			visualChange.ToInt16s().ToBits().Store(rocks.Stepping[i:])
			for _, changed := range rocks.Stepping[i : i+RocksPerPackedVector] {
				if changed != 0 {
					rocks.markGroupLayerDirty(group)
					break
				}
			}
		}
		positionX.Store(rocks.PosX[i:])
		positionY.Store(rocks.PosY[i:])
		slope.Store(rocks.Slope[i:])
		stepping.Store(rocks.Stepping[i:])
	}
	if fullEnd == end {
		return
	}
	positionX, _ := simd.LoadUint16sPart(rocks.PosX[fullEnd:end])
	positionY, _ := simd.LoadUint16sPart(rocks.PosY[fullEnd:end])
	stepping, _ := simd.LoadUint16sPart(rocks.Stepping[fullEnd:end])
	group := fullEnd / RocksPerPackedVector
	rocks.drawGroups[group] = rockGroupNeedsUpdate16(
		positionX, positionY, stepping, rocks.Stepping[fullEnd:end], state)
	if !rocks.drawGroups[group] {
		return
	}
	slope, _ := simd.LoadUint16sPart(rocks.Slope[fullEnd:end])
	oldPositionX, oldPositionY, oldSlope := positionX, positionY, slope
	positionX, positionY, slope, stepping = updateRockGroup16(
		positionX, positionY, slope, stepping, state)
	if !rocks.redraw {
		visualChange := positionX.And(coordinateMask16).NotEqual(oldPositionX.And(coordinateMask16)).
			Or(positionY.And(coordinateMask16).NotEqual(oldPositionY.And(coordinateMask16))).
			Or(slope.NotEqual(oldSlope))
		visualChange.ToInt16s().ToBits().StorePart(rocks.Stepping[fullEnd:end])
		for _, changed := range rocks.Stepping[fullEnd:end] {
			if changed != 0 {
				rocks.markGroupLayerDirty(group)
				break
			}
		}
	}
	positionX.StorePart(rocks.PosX[fullEnd:end])
	positionY.StorePart(rocks.PosY[fullEnd:end])
	slope.StorePart(rocks.Slope[fullEnd:end])
	stepping.StorePart(rocks.Stepping[fullEnd:end])
}

func rockGroupNeedsUpdate16(
	packedX, packedY, packedStepping simd.Uint16s,
	reduction []uint16,
	input *rockUpdateInput16,
) bool {
	activity := packedX.Or(packedY).And(velocityMask16).Or(packedStepping)
	if input.routine != rockUpdateDisabled {
		positionX := packedX.And(coordinateMask16).ConvertToInt16()
		positionY := packedY.And(coordinateMask16).ConvertToInt16()
		absX := positionX.Sub(input.mouseX).Abs().ToBits()
		absY := positionY.Sub(input.mouseY).Abs().ToBits()
		broadHit := absX.LessEqual(input.constants.mouseRadius).
			And(absY.LessEqual(input.constants.mouseRadius))
		activity = activity.Or(broadHit.ToInt16s().ToBits())
	}

	// The simd package has no portable horizontal Any operation. Use the
	// already-loaded Stepping stream as temporary reduction storage; an idle
	// group necessarily writes the same all-zero state, while an active group
	// is overwritten by the full update immediately afterward.
	activity.StorePart(reduction)
	var active uint16
	for _, lane := range reduction {
		active |= lane
	}
	return active != 0
}

func updateRockGroup16(
	packedX, packedY, packedSlope, packedAnimate simd.Uint16s,
	state *rockUpdateInput16,
) (simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s) {
	positionX := packedX.And(coordinateMask16).ConvertToInt16()
	positionY := packedY.And(coordinateMask16).ConvertToInt16()
	velocityX := packedX.BitsToInt16().ShiftAllRight(12)
	velocityY := packedY.BitsToInt16().ShiftAllRight(12)
	previousX, previousY := velocityX, velocityY

	size := packedSlope.And(nibbleMask16)
	falseMask := zeroUint16.NotEqual(zeroUint16)
	mouseHit := falseMask
	if state.routine == rockUpdateHover {
		velocityX, velocityY, mouseHit = setHoverVelocity16(
			positionX, positionY, velocityX, velocityY, size,
			state.constants, state.mouseX, state.mouseY)
	} else if state.routine == rockUpdateButton {
		velocityX, velocityY, mouseHit = setButtonVelocity16(
			positionX, positionY, velocityX, velocityY,
			state.constants, state.mouseX, state.mouseY, state.farForce, state.attract)
	}

	bounce := oneUint16.Equal(oneUint16)
	if state.routine == rockUpdateButton {
		// A rock actively pushed into a wall keeps the mouse-assigned direction.
		// The wall still clamps it and drives the queued stepping animation.
		bounce = mouseHit.ToInt16s().Equal(zeroInt16)
	}
	velocityX, velocityY, wallHitX, wallHitY := collideWalls16(
		positionX, positionY, velocityX, velocityY, bounce)
	wallHit := wallHitX.Or(wallHitY)
	impact := mouseHit.Or(wallHit)
	collisionVelocityX, collisionVelocityY := velocityX, velocityY
	forceStepping := packedAnimate.ShiftAllRight(1).And(oneUint16)

	// Collision velocities move once at full strength. Every other finite
	// velocity damps exactly once before this update's movement.
	damp := forceStepping.Equal(zeroUint16).And(impact.ToInt16s().Equal(zeroInt16))
	velocityX = dampVelocity16(velocityX).IfElse(damp, velocityX)
	velocityY = dampVelocity16(velocityY).IfElse(damp, velocityY)
	positionX = positionX.IfElse(wallHitX.And(bounce), positionX.Add(velocityX))
	positionY = positionY.IfElse(wallHitY.And(bounce), positionY.Add(velocityY))
	positionX = positionX.Max(oneInt16).Min(screenWidth16)
	positionY = positionY.Max(oneInt16).Min(screenHeight16)

	slopeZ := packedSlope.ShiftAllRight(4).And(nibbleMask16)
	slopeX := packedSlope.BitsToInt16().ShiftAllRight(12)
	slopeY := packedSlope.ShiftAllLeft(4).BitsToInt16().ShiftAllRight(12)
	stepY := packedAnimate.ShiftAllRight(13).And(nibbleValues16[7])
	stepX := packedAnimate.ShiftAllRight(10).And(nibbleValues16[7])
	stepZ := packedAnimate.ShiftAllRight(6).And(nibbleMask16)
	stepTick := packedAnimate.ShiftAllRight(2).And(nibbleMask16)
	stepping := packedAnimate.And(oneUint16)

	// Existing animation advances after movement and collision handling. New
	// collision state is scheduled afterward, so it starts on the next update.
	size, slopeZ, stepX, stepY, stepZ, stepTick, forceStepping, stepping, slopeX, slopeY =
		advanceStepping16(size, slopeZ, stepX, stepY, stepZ, stepTick,
			forceStepping, stepping, slopeX, slopeY, previousX, previousY)
	stopped := velocityX.Equal(zeroInt16).And(velocityY.Equal(zeroInt16)).
		And(forceStepping.Equal(zeroUint16)).And(stepZ.Equal(zeroUint16)).And(stepping.Equal(zeroUint16))
	stepX = zeroUint16.IfElse(stopped, stepX)
	stepY = zeroUint16.IfElse(stopped, stepY)
	stepTick = zeroUint16.IfElse(stopped, stepTick)
	stepX, stepY, stepZ, stepping, slopeX, slopeY, _ = scheduleCollision16(
		stepX, stepY, stepZ, forceStepping, stepping,
		slopeX, slopeY, collisionVelocityX, collisionVelocityY, previousX, previousY,
		impact, wallHit, mouseHit)

	return packPositionAxis16(positionX, velocityX), packPositionAxis16(positionY, velocityY),
		packSlope16(size, slopeZ, slopeX, slopeY),
		packAnimate16(stepX, stepY, stepZ, stepTick, forceStepping, stepping)
}

func advanceStepping16(
	size, slopeZ, stepX, stepY, stepZ, tick, forceStepping, stepping simd.Uint16s,
	slopeX, slopeY, velocityX, velocityY simd.Int16s,
) (simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Uint16s, simd.Int16s, simd.Int16s) {
	inactive := forceStepping.Equal(zeroUint16)
	active := forceStepping.NotEqual(zeroUint16)
	stepZ = canonicalForceStepZ16(stepZ).IfElse(active, stepZ)
	stepping = stepping.Masked(inactive)
	hasStepping := stepX.Or(stepY).Or(stepZ).Or(stepping).Or(forceStepping).NotEqual(zeroUint16)
	enabled := size.NotEqual(zeroUint16).And(hasStepping)
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
	startStep := stepZ.Equal(zeroUint16).And(stepping.NotEqual(zeroUint16)).And(due).And(inactive)
	slopeZ = updateSlopeZ16(slopeZ, velocityX, velocityY).
		IfElse(finiteZ.Or(startStep).Or(active.And(due)), slopeZ)
	stepZ = stepZ.Sub(oneUint16.Masked(finiteZ))
	rolloverStep := stepZ.Equal(zeroUint16).And(stepping.NotEqual(zeroUint16)).And(due).And(inactive)
	stepZ = maximumStepZ16.IfElse(rolloverStep, stepZ)
	stepping = zeroUint16.IfElse(rolloverStep, stepping)
	return size, slopeZ, stepX, stepY, stepZ, tick, forceStepping, stepping, slopeX, slopeY
}

func canonicalForceStepZ16(value simd.Uint16s) simd.Uint16s {
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

func packAnimate16(stepX, stepY, stepZ, tick, forceStepping, stepping simd.Uint16s) simd.Uint16s {
	return stepY.ShiftAllLeft(13).Or(stepX.ShiftAllLeft(10)).Or(stepZ.ShiftAllLeft(6)).
		Or(tick.ShiftAllLeft(2)).Or(forceStepping.ShiftAllLeft(1)).Or(stepping)
}
