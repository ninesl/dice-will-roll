package rocks

import (
	"simd"

	"github.com/ninesl/dice-will-roll/controls"
)

func unpackSpritesSIMD(sprite simd.Uint32s) (
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) {
	sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY = unpackCollisionSpritesSIMD(sprite)
	stepZ = canonicalPermaStepZ(stepZ).IfElse(permaSpin.NotEqual(zeroCoordinates), stepZ)
	spinAgain = spinAgain.Masked(permaSpin.Equal(zeroCoordinates))
	return sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY
}

func unpackCollisionSpritesSIMD(sprite simd.Uint32s) (
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) {
	slopeX, slopeY = unpackSlopesSIMD(sprite)
	slopeZ = sprite.And(spriteSlopeZMask).ShiftAllRight(20)
	sizeScore = sprite.And(spriteSizeScoreMask).ShiftAllRight(16)
	stepY = sprite.And(stepYMask).ShiftAllRight(13)
	stepX = sprite.And(stepXMask).ShiftAllRight(10)
	stepZ = sprite.And(stepZMask).ShiftAllRight(6)
	stepTick = sprite.And(stepTickMask).ShiftAllRight(2)
	permaSpin = sprite.And(permaSpinMask).ShiftAllRight(1)
	spinAgain = sprite.And(spinAgainMask)
	return sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY
}

func packSpritesSIMD(
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) simd.Uint32s {
	return packCollisionSpritesSIMD(
		sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY)
}

func packCollisionSpritesSIMD(
	sizeScore, slopeZ simd.Uint32s,
	stepX, stepY simd.Uint32s,
	stepZ, stepTick simd.Uint32s,
	permaSpin simd.Uint32s,
	spinAgain simd.Uint32s,
	slopeX, slopeY simd.Int32s,
) simd.Uint32s {
	return packSlopesSIMD(slopeX, slopeY).
		Or(slopeZ.ShiftAllLeft(20)).
		Or(sizeScore.ShiftAllLeft(16)).
		Or(stepY.ShiftAllLeft(13)).
		Or(stepX.ShiftAllLeft(10)).
		Or(stepZ.ShiftAllLeft(6)).
		Or(stepTick.ShiftAllLeft(2)).
		Or(permaSpin.ShiftAllLeft(1)).
		Or(spinAgain)
}

// UnpackSprite decodes one packed sprite.
// slopeX: bits 31..28. Codes 0001..1111 encode visual slopes -7..7; slope 0
// is encoded as 1000. The packer never emits reserved code 0000, which
// unpacking treats as zero for empty SIMD lanes.
// slopeY: bits 27..24. Uses the same 0001..1111 encoding as slopeX;
// reserved code 0000 has the same empty-lane behavior.
// slopeZ: bits 23..20. All 16 values 0..15 are atlas frames spaced evenly
// around 360 degrees; updates wrap between 0 and 15.
// sizeScore: bits 19..16. Values 1..15 select size and set the StepTick
// interval; 0000 is reserved and does not advance StepTick.
// stepY/stepX: bits 15..10. Independent 0..7 shortest-path catch-up counts.
// stepZ: bits 9..6. Without PermaSpin, values hold finite Z updates and 15 is
// loaded after SpinAgain supplies the first full-spin frame.
// With PermaSpin, bits 1..0 encode X direction and bits 3..2 encode Y direction:
// 00 is inactive, 01 increments, 10 decrements, and 11 is reserved. Only the
// latest bounced direction remains active; simultaneous X/Y wall overlaps
// select the greater penetration, with X winning an exact tie.
// stepTick: bits 5..2. Counts from 0 toward sizeScore while any step counter or
// PermaSpin is active, then resets before packing; 1111 is never persisted.
// permaSpin: bit 1. A value of 1 advances slopeZ on every due StepTick and
// ignores finite Z state. Canonical permanent state is 10.
// spinAgain: bit 0. After stepZ reaches 0, the next due update consumes it and
// starts one full 16-frame Z spin. Finite collisions clear it because their
// biased slope-code distance already determines the complete Z rotation.
// Flag state 11 is normalized to 10 because SpinAgain is redundant when
// PermaSpin is active; 00 and 01 retain their distinct finite-spin meanings.
//
// velocityX/Y live in the packed position and move the rock. When either
// velocity changes, stepX/Y receive the shortest circular distance from the
// current visual slope to that velocity. A finite collision sets stepZ to the
// absolute biased-code distance across changed X/Y slopes, capped at 15.
// Permanent spin stores the selected bounced X or Y direction and the inverse
// rotation direction in stepZ.
//
// While any step counter or PermaSpin is active, stepTick counts down from
// max(1, floor(sizeScore/2))-1. Zero performs the animation update. Each due
// update moves slopeX/Y one shortest-path step toward velocityX/Y and
// decrements StepX/Y. Finite Z updates decrement StepZ to 0. If SpinAgain is
// set, it then provides one Z update, reloads StepZ to 15, and clears itself,
// producing one complete 16-frame revolution. PermaSpin clears SpinAgain, keeps advancing
// slopeZ, and overrides the selected X/Y update mask with the direction stored
// in StepZ so that X or Y continues wrapping after StepX/Y reaches zero.
// Before movement, state 11 normalizes to 10. Finite states 00 and 01 move each
// velocity one integer toward zero after movement; permanent state 10 skips
// damping. Once both
// damped velocities reach zero, all step counters and spin flags are cleared;
// the current visual orientation is retained without a settling animation.
//
// State audit: the X/Y code 0000 and SizeScore 0000 are reserved; StepTick 1111
// is transient and reset before packing. Finite StepZ and SlopeZ use all 16
// states, and StepX/Y use all eight. PermaSpin StepZ canonicalizes to 0, 1, 2,
// 4, or 8; reserved direction code 11 becomes inactive and simultaneous X/Y
// direction codes keep X only. Flag 11 is canonicalized to 10. Packed position
// coordinates use their entire 0..4095 ranges; coordinate 0 is not a sentinel.
func UnpackSprite(packedSprite uint32) (
	sizeScore, slopeZ,
	stepX, stepY,
	stepZ, stepTick,
	permaSpin,
	spinAgain,
	slopeX, slopeY int,
) {
	slopeX, slopeY = unpackSlope(packedSprite)
	sizeScore = int((packedSprite & spriteSizeScoreMask32) >> 16)
	slopeZ = int((packedSprite & spriteSlopeZMask32) >> 20)
	stepY = int((packedSprite & stepYMask32) >> 13)
	stepX = int((packedSprite & stepXMask32) >> 10)
	stepZ = int((packedSprite & stepZMask32) >> 6)
	stepTick = int((packedSprite & stepTickMask32) >> 2)
	permaSpin = int((packedSprite & permaSpinMask32) >> 1)
	spinAgain = int(packedSprite & spinAgainMask32)
	if permaSpin != 0x0 {
		stepZ = int(canonicalPermaStepZScalar(uint32(stepZ)))
		spinAgain = 0x0
	}
	return sizeScore, slopeZ,
		stepX, stepY,
		stepZ, stepTick,
		permaSpin,
		spinAgain,
		slopeX, slopeY
}

func canonicalPermaStepZScalar(stepZ uint32) uint32 {
	if stepZ == 0xF {
		return stepZ
	}
	xDirection := stepZ & permaXDirectionMask32
	yDirection := stepZ & permaYDirectionMask32
	if xDirection == permaXDirectionMask32 {
		xDirection = 0x0
	}
	if yDirection == permaYDirectionMask32 || xDirection != 0x0 {
		yDirection = 0x0
	}
	return xDirection | yDirection
}

func PackSprite(input InputSprite) uint32 {
	stepZ := input.StepZ & 0xF
	spinAgain := input.SpinAgain & 0x1
	if input.PermaSpin&0x1 != 0x0 {
		stepZ = canonicalPermaStepZScalar(stepZ)
		spinAgain = 0x0
	}
	return packSlope(input.SlopeX, input.SlopeY) |
		(input.SlopeZ<<0x14)&spriteSlopeZMask32 |
		(input.SizeScore<<0x10)&spriteSizeScoreMask32 |
		(input.StepY<<0xD)&stepYMask32 |
		(input.StepX<<0xA)&stepXMask32 |
		(stepZ<<0x6)&stepZMask32 |
		(input.StepTick<<0x2)&stepTickMask32 |
		(input.PermaSpin<<0x1)&permaSpinMask32 |
		spinAgain&spinAgainMask32
}

func incrementSlope(slope simd.Int32s) simd.Int32s {
	incrementedSlope := slope.Add(oneSlope)
	return minimumSlope.IfElse(incrementedSlope.Greater(maximumSlope), incrementedSlope)
}

func decrementSlope(slope simd.Int32s) simd.Int32s {
	decrementedSlope := slope.Sub(oneSlope)
	return maximumSlope.IfElse(decrementedSlope.Less(minimumSlope), decrementedSlope)
}

func canonicalPermaStepZ(stepZ simd.Uint32s) simd.Uint32s {
	fullSpin := stepZ.Equal(maximumStepZ)
	xDirection := stepZ.And(permaXDirectionMask)
	yDirection := stepZ.And(permaYDirectionMask)
	xDirection = zeroCoordinates.IfElse(xDirection.Equal(permaXDirectionMask), xDirection)
	yDirection = zeroCoordinates.IfElse(yDirection.Equal(permaYDirectionMask), yDirection)
	yDirection = zeroCoordinates.IfElse(xDirection.NotEqual(zeroCoordinates), yDirection)
	return maximumStepZ.IfElse(fullSpin, xDirection.Or(yDirection))
}

func slopeDirection(slope, velocity simd.Int32s) (simd.Mask32s, simd.Mask32s) {
	forwardDistance := velocity.Sub(slope)
	forwardDistance = forwardDistance.Add(slopeCycle).IfElse(
		forwardDistance.Less(zeroSlope), forwardDistance)
	increment := forwardDistance.Greater(zeroSlope).
		And(forwardDistance.LessEqual(halfSlopeCycle))
	decrement := forwardDistance.Greater(halfSlopeCycle)
	return increment, decrement
}

// updateSlopeZ advances the finite Z spin in the dominant velocity direction.
// Positive velocity decrements the atlas frame; negative velocity increments it.
func slopeDistance(slope, velocity simd.Int32s) simd.Uint32s {
	difference := velocity.Sub(slope).Abs()
	wrappedDistance := slopeCycle.Sub(difference)
	return wrappedDistance.IfElse(
		difference.Greater(halfSlopeCycle), difference,
	).ConvertToUint32()
}

func updateCollisionState(
	stepX, stepY, stepZ, permaSpin, spinAgain simd.Uint32s,
	slopeX, slopeY, velocityX, velocityY simd.Int32s,
	previousVelocityX, previousVelocityY simd.Int32s,
	impact simd.Mask32s,
) (
	simd.Uint32s, simd.Uint32s, simd.Uint32s, simd.Uint32s,
	simd.Int32s, simd.Int32s,
) {
	changedX := velocityX.NotEqual(previousVelocityX)
	changedY := velocityY.NotEqual(previousVelocityY)
	directionChanged := changedX.Or(changedY)
	permaSpinActive := permaSpin.NotEqual(zeroCoordinates)
	stepX = slopeDistance(slopeX, velocityX).IfElse(changedX, stepX)
	stepY = slopeDistance(slopeY, velocityY).IfElse(changedY, stepY)
	stepX = zeroCoordinates.IfElse(permaSpinActive.And(changedY), stepX)
	stepY = zeroCoordinates.IfElse(permaSpinActive.And(changedX), stepY)
	incrementX, decrementX := slopeDirection(slopeX, velocityX)
	incrementY, decrementY := slopeDirection(slopeY, velocityY)
	permaXCode := permaXIncrement.Masked(decrementX).
		Or(permaXDecrement.Masked(incrementX))
	permaYCode := permaYIncrement.Masked(decrementY).
		Or(permaYDecrement.Masked(incrementY))
	stepZ = permaXCode.IfElse(permaSpinActive.And(changedX), stepZ)
	stepZ = permaYCode.IfElse(permaSpinActive.And(changedY), stepZ)
	spinAgain = oneCoordinates.IfElse(directionChanged, spinAgain)
	finiteImpact := impact.And(permaSpin.Equal(zeroCoordinates))
	flipDistanceX := velocityX.Sub(slopeX).Abs().ConvertToUint32().Masked(changedX)
	flipDistanceY := velocityY.Sub(slopeY).Abs().ConvertToUint32().Masked(changedY)
	collisionStepZ := flipDistanceX.Add(flipDistanceY)
	collisionStepZ = maximumStepZ.IfElse(collisionStepZ.Greater(maximumStepZ), collisionStepZ)
	stepZ = collisionStepZ.IfElse(finiteImpact, stepZ)
	slopeX = velocityX.IfElse(impact, slopeX)
	slopeY = velocityY.IfElse(impact, slopeY)
	spinAgain = zeroCoordinates.IfElse(finiteImpact, spinAgain)
	spinAgain = spinAgain.Masked(permaSpin.Equal(zeroCoordinates))
	return stepX, stepY, stepZ, spinAgain, slopeX, slopeY
}

func collideWall(
	position simd.Uint32s,
	velocity simd.Int32s,
	extent simd.Uint32s,
	radius simd.Uint32s,
) (simd.Uint32s, simd.Uint32s, simd.Mask32s) {
	minimumPosition := radius
	maximumPosition := extent.Sub(radius)
	movingNegative := velocity.Less(zeroSlope)
	movingPositive := velocity.Greater(zeroSlope)
	minimumOverflow := movingNegative.And(position.GreaterEqual(extent))
	maximumOverflow := movingPositive.And(position.GreaterEqual(extent))
	hitMinimumRadius := movingNegative.And(position.Less(radius))
	hitMaximumRadius := movingPositive.And(position.Add(radius).GreaterEqual(extent))
	hitWall := minimumOverflow.Or(maximumOverflow).
		Or(hitMinimumRadius).Or(hitMaximumRadius)
	minimumOverlap := radius.Sub(position).Masked(hitMinimumRadius)
	maximumRadiusOverlap := position.Add(radius).Sub(extent).Masked(hitMaximumRadius)
	overlap := minimumOverlap.Or(maximumRadiusOverlap)
	overlap = maximumOverlap.IfElse(minimumOverflow.Or(maximumOverflow), overlap)

	position = minimumPosition.IfElse(minimumOverflow.Or(hitMinimumRadius), position)
	position = maximumPosition.IfElse(maximumOverflow.Or(hitMaximumRadius), position)

	return position, overlap, hitWall
}

func collideWalls(
	positionX, positionY simd.Uint32s,
	velocityX, velocityY simd.Int32s,
	radius simd.Uint32s,
) (
	simd.Uint32s, simd.Uint32s,
	simd.Int32s, simd.Int32s,
	simd.Mask32s,
) {
	positionX, overlapX, candidateX := collideWall(positionX, velocityX, screenWidth, radius)
	positionY, overlapY, candidateY := collideWall(positionY, velocityY, screenHeight, radius)
	hitX := candidateX.And(overlapX.GreaterEqual(overlapY))
	hitY := candidateY.And(overlapY.Greater(overlapX))
	velocityX = velocityX.Neg().IfElse(hitX, velocityX)
	velocityY = velocityY.Neg().IfElse(hitY, velocityY)
	return positionX, positionY, velocityX, velocityY, hitX.Or(hitY)
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

func hoverRadius(sizeScore simd.Uint32s, amountScale int) simd.Uint32s {
	radius := zeroCoordinates
	for score := 1; score < BitSpriteSlopeCodeCount; score++ {
		radius = hoverRadiusLookups[amountScale][score].IfElse(
			sizeScore.Equal(sizeScoreLanes[score]), radius)
	}
	return radius
}

func setMouseVelocity(
	positionX, positionY simd.Uint32s,
	velocityX, velocityY simd.Int32s,
	radius simd.Uint32s,
	mouseX, mouseY simd.Int32s,
	impact simd.Mask32s,
) (simd.Int32s, simd.Int32s, simd.Mask32s) {
	distanceX := positionX.ConvertToInt32().Sub(mouseX)
	distanceY := positionY.ConvertToInt32().Sub(mouseY)
	absDistanceX := distanceX.Abs().ConvertToUint32()
	absDistanceY := distanceY.Abs().ConvertToUint32()
	distanceSquared := absDistanceX.Mul(absDistanceX).Add(absDistanceY.Mul(absDistanceY))
	radiusSquared := radius.Mul(radius)
	hit := distanceSquared.LessEqual(radiusSquared)

	scaledDistanceSquared := distanceSquared.Mul(mouseFalloffScale)
	falloff := zeroCoordinates
	for band := uint32(1); band <= 6; band++ {
		threshold := radiusSquared.Mul(mouseFalloffBandSquared[band])
		falloff = mouseFalloffBands[band].IfElse(
			scaledDistanceSquared.GreaterEqual(threshold), falloff)
	}
	impulse := maximumSlope.Sub(falloff.ConvertToInt32())
	impulseX := impulse.IfElse(distanceX.Greater(zeroSlope), impulse.Neg()).
		Masked(distanceX.NotEqual(zeroSlope))
	impulseY := impulse.IfElse(distanceY.Greater(zeroSlope), impulse.Neg()).
		Masked(distanceY.NotEqual(zeroSlope))
	velocityX = impulseX.IfElse(hit, velocityX)
	velocityY = impulseY.IfElse(hit, velocityY)
	return velocityX, velocityY, impact.Or(hit)
}

const UpdateStride = 2

var (
	RocksPerUpdateBatch     int
	mouseFalloffScale       simd.Uint32s
	mouseFalloffBands       [7]simd.Uint32s
	mouseFalloffBandSquared [7]simd.Uint32s

	nibbleByteMask    simd.Uint32s
	nibbleZeroU8      simd.Uint8s
	nibbleOneU8       simd.Uint8s
	nibbleEightU8     simd.Uint8s
	nibbleSixteenU8   simd.Uint8s
	nibbleZeroI8      simd.Int8s
	nibbleOneI8       simd.Int8s
	nibbleMinI8       simd.Int8s
	nibbleMaxI8       simd.Int8s
	nibbleCycleI8     simd.Int8s
	nibbleHalfI8      simd.Int8s
	nibbleValuesU8    [BitSpriteSlopeCodeCount]simd.Uint8s
	nibbleTickResetU8 [BitSpriteSlopeCodeCount]simd.Uint8s
)

func initByteFieldSIMD() {
	RocksPerUpdateBatch = simd.Uint8s{}.Len()
	mouseFalloffScale = simd.BroadcastUint32s(36)
	for band := uint32(1); band <= 6; band++ {
		mouseFalloffBands[band] = simd.BroadcastUint32s(band)
		mouseFalloffBandSquared[band] = simd.BroadcastUint32s(band * band)
	}
	nibbleByteMask = simd.BroadcastUint32s(0xFF)
	nibbleZeroU8 = simd.BroadcastUint8s(0)
	nibbleOneU8 = simd.BroadcastUint8s(1)
	nibbleEightU8 = simd.BroadcastUint8s(8)
	nibbleSixteenU8 = simd.BroadcastUint8s(16)
	nibbleZeroI8 = simd.BroadcastInt8s(0)
	nibbleOneI8 = simd.BroadcastInt8s(1)
	nibbleMinI8 = simd.BroadcastInt8s(-7)
	nibbleMaxI8 = simd.BroadcastInt8s(7)
	nibbleCycleI8 = simd.BroadcastInt8s(15)
	nibbleHalfI8 = simd.BroadcastInt8s(7)
	for value := range nibbleValuesU8 {
		nibbleValuesU8[value] = simd.BroadcastUint8s(uint8(value))
		nibbleTickResetU8[value] = simd.BroadcastUint8s(max(uint8(1), uint8(value)/2))
	}
}

// UpdateRocks updates one contiguous phase. Packed fields use Uint8s/Int8s;
// coordinates widen to 32 bits for movement and collision.
// UpdateRocks takes Rocks by value, but Positions and Sprites still share their
// backing arrays with the caller, so SIMD stores update the original elements.
// Changes to slice headers or scalar Rocks fields would need to be returned.
func UpdateRocks(rocks Rocks, phase int, mouse controls.MouseInfo) {
	if mouse.Position.X == 0 && mouse.Position.Y == 0 {
		updateMovementPhase(rocks, UpdateStride, phase, mouse, zeroSlope, zeroSlope)
		return
	}
	updateMovementPhase(
		rocks, UpdateStride, phase, mouse,
		simd.BroadcastInt32s(int32(mouse.Position.X)),
		simd.BroadcastInt32s(int32(mouse.Position.Y)))
}

func updateMovementPhase(
	rocks Rocks,
	phaseCount, phase int,
	mouse controls.MouseInfo,
	mouseX, mouseY simd.Int32s,
) {
	groupCount := (len(rocks.Positions) + RocksPerUpdateBatch - 1) / RocksPerUpdateBatch
	start := min(len(rocks.Positions), groupCount*phase/phaseCount*RocksPerUpdateBatch)
	end := min(len(rocks.Positions), groupCount*(phase+1)/phaseCount*RocksPerUpdateBatch)
	updateMovement(
		rocks.Positions[start:end], rocks.Sprites[start:end],
		rocks.AmountScale, mouse, mouseX, mouseY)
}

func updateMovement(
	positions RockPositions,
	sprites RockSprites,
	amountScale int,
	mouse controls.MouseInfo,
	mouseX, mouseY simd.Int32s,
) {
	// Each rock has one packed position uint32 and one packed sprite uint32.
	// A Uint8s vector has four times as many lanes as a Uint32s vector, so one
	// update batch loads four packed-position vectors and four packed-sprite
	// vectors. Their small fields transpose into one Uint8s/Int8s vector, while
	// coordinates remain in four Uint32s vectors. The updated fields transpose
	// back into the same packed position and sprite vectors for one store.
	fullBatchEnd := len(positions) - len(positions)%RocksPerUpdateBatch
	for start := 0; start < fullBatchEnd; start += RocksPerUpdateBatch {
		packedPositions0 := simd.LoadUint32s(positions[start:])
		packedPositions1 := simd.LoadUint32s(positions[start+RocksPerPackedVector:])
		packedPositions2 := simd.LoadUint32s(positions[start+2*RocksPerPackedVector:])
		packedPositions3 := simd.LoadUint32s(positions[start+3*RocksPerPackedVector:])
		packedSprites0 := simd.LoadUint32s(sprites[start:])
		packedSprites1 := simd.LoadUint32s(sprites[start+RocksPerPackedVector:])
		packedSprites2 := simd.LoadUint32s(sprites[start+2*RocksPerPackedVector:])
		packedSprites3 := simd.LoadUint32s(sprites[start+3*RocksPerPackedVector:])
		packedPositions0, packedPositions1, packedPositions2, packedPositions3,
			packedSprites0, packedSprites1, packedSprites2, packedSprites3 = updateRockBatch(
			packedPositions0, packedPositions1, packedPositions2, packedPositions3,
			packedSprites0, packedSprites1, packedSprites2, packedSprites3,
			amountScale, mouse, mouseX, mouseY)
		packedPositions0.Store(positions[start:])
		packedPositions1.Store(positions[start+RocksPerPackedVector:])
		packedPositions2.Store(positions[start+2*RocksPerPackedVector:])
		packedPositions3.Store(positions[start+3*RocksPerPackedVector:])
		packedSprites0.Store(sprites[start:])
		packedSprites1.Store(sprites[start+RocksPerPackedVector:])
		packedSprites2.Store(sprites[start+2*RocksPerPackedVector:])
		packedSprites3.Store(sprites[start+3*RocksPerPackedVector:])
	}
	if fullBatchEnd == len(positions) {
		return
	}

	// One partial batch handles every rock left after the full SIMD batches.
	tail1 := min(fullBatchEnd+RocksPerPackedVector, len(positions))
	tail2 := min(fullBatchEnd+2*RocksPerPackedVector, len(positions))
	tail3 := min(fullBatchEnd+3*RocksPerPackedVector, len(positions))
	packedPositions0, _ := simd.LoadUint32sPart(positions[fullBatchEnd:])
	packedPositions1, _ := simd.LoadUint32sPart(positions[tail1:])
	packedPositions2, _ := simd.LoadUint32sPart(positions[tail2:])
	packedPositions3, _ := simd.LoadUint32sPart(positions[tail3:])
	packedSprites0, _ := simd.LoadUint32sPart(sprites[fullBatchEnd:])
	packedSprites1, _ := simd.LoadUint32sPart(sprites[tail1:])
	packedSprites2, _ := simd.LoadUint32sPart(sprites[tail2:])
	packedSprites3, _ := simd.LoadUint32sPart(sprites[tail3:])
	packedPositions0, packedPositions1, packedPositions2, packedPositions3,
		packedSprites0, packedSprites1, packedSprites2, packedSprites3 = updateRockBatch(
		packedPositions0, packedPositions1, packedPositions2, packedPositions3,
		packedSprites0, packedSprites1, packedSprites2, packedSprites3,
		amountScale, mouse, mouseX, mouseY)
	packedPositions0.StorePart(positions[fullBatchEnd:])
	packedPositions1.StorePart(positions[tail1:])
	packedPositions2.StorePart(positions[tail2:])
	packedPositions3.StorePart(positions[tail3:])
	packedSprites0.StorePart(sprites[fullBatchEnd:])
	packedSprites1.StorePart(sprites[tail1:])
	packedSprites2.StorePart(sprites[tail2:])
	packedSprites3.StorePart(sprites[tail3:])
}

// The four packed position and sprite vectors cover the same rocks as each
// extracted byte-field vector. Coordinates stay split across four Uint32s
// vectors; velocities and sprite fields update together in Uint8s/Int8s.
func updateRockBatch(
	packedPositions0, packedPositions1, packedPositions2, packedPositions3,
	packedSprites0, packedSprites1, packedSprites2, packedSprites3 simd.Uint32s,
	amountScale int,
	mouse controls.MouseInfo,
	mouseX, mouseY simd.Int32s,
) (
	simd.Uint32s, simd.Uint32s, simd.Uint32s, simd.Uint32s,
	simd.Uint32s, simd.Uint32s, simd.Uint32s, simd.Uint32s,
) {
	sizeScore := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, spriteSizeScoreMask, 16)
	slopeZ := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, spriteSlopeZMask, 20)
	stepX := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, stepXMask, 10)
	stepY := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, stepYMask, 13)
	stepZ := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, stepZMask, 6)
	stepTick := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, stepTickMask, 2)
	permaSpin := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, permaSpinMask, 1)
	spinAgain := extractNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, spinAgainMask, 0)
	slopeX := extractSignedNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, slopeXMask, 28)
	slopeY := extractSignedNibbleField(packedSprites0, packedSprites1, packedSprites2, packedSprites3, slopeYMask, 24)
	velocityX := extractSignedNibbleField(packedPositions0, packedPositions1, packedPositions2, packedPositions3, velocityXMask, 28)
	velocityY := extractSignedNibbleField(packedPositions0, packedPositions1, packedPositions2, packedPositions3, velocityYMask, 24)

	positionX0, positionX1, positionX2, positionX3 := unpackNibbleCoordinates(
		packedPositions0, packedPositions1, packedPositions2, packedPositions3, positionXMask, 12)
	positionY0, positionY1, positionY2, positionY3 := unpackNibbleCoordinates(
		packedPositions0, packedPositions1, packedPositions2, packedPositions3, positionYMask, 0)
	velocityX0, velocityX1, velocityX2, velocityX3 := widenInt8Lanes(velocityX)
	velocityY0, velocityY1, velocityY2, velocityY3 := widenInt8Lanes(velocityY)
	positionX0 = positionX0.ConvertToInt32().Add(velocityX0).ConvertToUint32()
	positionX1 = positionX1.ConvertToInt32().Add(velocityX1).ConvertToUint32()
	positionX2 = positionX2.ConvertToInt32().Add(velocityX2).ConvertToUint32()
	positionX3 = positionX3.ConvertToInt32().Add(velocityX3).ConvertToUint32()
	positionY0 = positionY0.ConvertToInt32().Add(velocityY0).ConvertToUint32()
	positionY1 = positionY1.ConvertToInt32().Add(velocityY1).ConvertToUint32()
	positionY2 = positionY2.ConvertToInt32().Add(velocityY2).ConvertToUint32()
	positionY3 = positionY3.ConvertToInt32().Add(velocityY3).ConvertToUint32()

	sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain,
		slopeX, slopeY, velocityX, velocityY = updateByteFields(
		sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain,
		slopeX, slopeY, velocityX, velocityY)
	velocityX0, velocityX1, velocityX2, velocityX3 = widenInt8Lanes(velocityX)
	velocityY0, velocityY1, velocityY2, velocityY3 = widenInt8Lanes(velocityY)

	packedPositions0, packedSprites0 = collideRockGroup(0, positionX0, positionY0, velocityX0, velocityY0, sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain, slopeX, slopeY, amountScale, mouse, mouseX, mouseY)
	packedPositions1, packedSprites1 = collideRockGroup(1, positionX1, positionY1, velocityX1, velocityY1, sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain, slopeX, slopeY, amountScale, mouse, mouseX, mouseY)
	packedPositions2, packedSprites2 = collideRockGroup(2, positionX2, positionY2, velocityX2, velocityY2, sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain, slopeX, slopeY, amountScale, mouse, mouseX, mouseY)
	packedPositions3, packedSprites3 = collideRockGroup(3, positionX3, positionY3, velocityX3, velocityY3, sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain, slopeX, slopeY, amountScale, mouse, mouseX, mouseY)
	return packedPositions0, packedPositions1, packedPositions2, packedPositions3,
		packedSprites0, packedSprites1, packedSprites2, packedSprites3
}

func collideRockGroup(group uint8, positionX, positionY simd.Uint32s, velocityX, velocityY simd.Int32s, sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain simd.Uint8s, slopeX, slopeY simd.Int8s, amountScale int, mouse controls.MouseInfo, mouseX, mouseY simd.Int32s) (simd.Uint32s, simd.Uint32s) {
	previousX, previousY := velocityX, velocityY
	sizeScore32 := uint8LanesToUint32Group(sizeScore, group)
	permaSpin32 := uint8LanesToUint32Group(permaSpin, group)
	positionX, positionY, velocityX, velocityY, impact := collideWalls(
		positionX, positionY, velocityX, velocityY,
		collisionRadius(sizeScore32, amountScale))
	if mouse.Position.X != 0 || mouse.Position.Y != 0 {
		velocityX, velocityY, impact = setMouseVelocity(
			positionX, positionY, velocityX, velocityY,
			mouseCollisionRadius(mouse, sizeScore32, amountScale),
			mouseX, mouseY, impact)
	}
	stepX32, stepY32, stepZ32, spinAgain32, slopeX32, slopeY32 := updateCollisionState(
		uint8LanesToUint32Group(stepX, group),
		uint8LanesToUint32Group(stepY, group),
		uint8LanesToUint32Group(stepZ, group),
		permaSpin32,
		uint8LanesToUint32Group(spinAgain, group),
		int8LanesToInt32Group(slopeX, group),
		int8LanesToInt32Group(slopeY, group),
		velocityX, velocityY,
		previousX, previousY, impact)
	return packPositionsSIMD(positionX, positionY, velocityX, velocityY),
		packCollisionSpritesSIMD(
			sizeScore32,
			uint8LanesToUint32Group(slopeZ, group),
			stepX32, stepY32, stepZ32,
			uint8LanesToUint32Group(stepTick, group),
			permaSpin32,
			spinAgain32, slopeX32, slopeY32)
}

func mouseCollisionRadius(
	mouse controls.MouseInfo,
	sizeScore simd.Uint32s,
	amountScale int,
) simd.Uint32s {
	if mouse.Down {
		return mouseRadii[amountScale]
	}
	return hoverRadius(sizeScore, amountScale)
}

func updateByteFields(
	sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain simd.Uint8s,
	slopeX, slopeY, velocityX, velocityY simd.Int8s,
) (
	simd.Uint8s, simd.Uint8s, simd.Uint8s, simd.Uint8s,
	simd.Uint8s, simd.Uint8s, simd.Uint8s, simd.Uint8s,
	simd.Int8s, simd.Int8s, simd.Int8s, simd.Int8s,
) {
	permaInactive := permaSpin.Equal(nibbleZeroU8)
	permaActive := permaSpin.NotEqual(nibbleZeroU8)
	stepZ = canonicalNibblePermaStepZ(stepZ).IfElse(permaActive, stepZ)
	spinAgain = spinAgain.Masked(permaInactive)
	hasSteps := stepX.Or(stepY).Or(stepZ).Or(spinAgain).Or(permaSpin).NotEqual(nibbleZeroU8)
	active := sizeScore.NotEqual(nibbleZeroU8).And(hasSteps)
	stepDue := stepTick.Equal(nibbleZeroU8).And(active)
	waiting := stepTick.NotEqual(nibbleZeroU8).And(active)
	stepTick = stepTick.Sub(nibbleOneU8.Masked(waiting))
	interval := nibbleOneU8
	for score := 1; score < BitSpriteSlopeCodeCount; score++ {
		interval = nibbleTickResetU8[score].IfElse(sizeScore.Equal(nibbleValuesU8[score]), interval)
	}
	stepTick = interval.Sub(nibbleOneU8).IfElse(stepDue, stepTick)

	finiteDue := stepDue.And(permaInactive)
	advanceX := stepX.NotEqual(nibbleZeroU8).And(finiteDue)
	advanceY := stepY.NotEqual(nibbleZeroU8).And(finiteDue)
	slopeX = moveNibbleSlope(slopeX, velocityX, advanceX)
	slopeY = moveNibbleSlope(slopeY, velocityY, advanceY)
	stepX = stepX.Sub(nibbleOneU8.Masked(advanceX))
	stepY = stepY.Sub(nibbleOneU8.Masked(advanceY))

	xCode := stepZ.And(nibbleValuesU8[3])
	yCode := stepZ.And(nibbleValuesU8[12])
	permaIncrementX := xCode.Equal(nibbleValuesU8[1]).And(permaActive).And(stepDue)
	permaDecrementX := xCode.Equal(nibbleValuesU8[2]).And(permaActive).And(stepDue)
	permaIncrementY := yCode.Equal(nibbleValuesU8[4]).And(permaActive).And(stepDue)
	permaDecrementY := yCode.Equal(nibbleValuesU8[8]).And(permaActive).And(stepDue)
	slopeX = incrementNibbleSlope(slopeX).IfElse(permaIncrementX, slopeX)
	slopeX = decrementNibbleSlope(slopeX).IfElse(permaDecrementX, slopeX)
	slopeY = incrementNibbleSlope(slopeY).IfElse(permaIncrementY, slopeY)
	slopeY = decrementNibbleSlope(slopeY).IfElse(permaDecrementY, slopeY)
	stepX = stepX.Sub(nibbleOneU8.Masked(
		permaIncrementX.Or(permaDecrementX).And(stepX.NotEqual(nibbleZeroU8))))
	stepY = stepY.Sub(nibbleOneU8.Masked(
		permaIncrementY.Or(permaDecrementY).And(stepY.NotEqual(nibbleZeroU8))))

	startFullSpin := stepZ.Equal(nibbleZeroU8).And(spinAgain.NotEqual(nibbleZeroU8)).And(stepDue)
	finiteStepZ := stepZ.NotEqual(nibbleZeroU8).And(stepDue).And(permaInactive)
	stepSlopeZ := finiteStepZ.Or(startFullSpin).Or(permaActive.And(stepDue))
	slopeZ = updateNibbleSlopeZ(slopeZ, velocityX, velocityY).IfElse(stepSlopeZ, slopeZ)
	stepZ = stepZ.Sub(nibbleOneU8.Masked(finiteStepZ))
	stepZ = nibbleValuesU8[15].IfElse(startFullSpin, stepZ)
	spinAgain = nibbleZeroU8.IfElse(startFullSpin, spinAgain)

	velocityX = dampNibbleVelocity(velocityX).IfElse(permaInactive, velocityX)
	velocityY = dampNibbleVelocity(velocityY).IfElse(permaInactive, velocityY)
	stopped := velocityX.Equal(nibbleZeroI8).And(velocityY.Equal(nibbleZeroI8)).
		And(permaInactive).And(stepZ.Equal(nibbleZeroU8)).And(spinAgain.Equal(nibbleZeroU8))
	stepX = nibbleZeroU8.IfElse(stopped, stepX)
	stepY = nibbleZeroU8.IfElse(stopped, stepY)
	stepTick = nibbleZeroU8.IfElse(stopped, stepTick)
	return sizeScore, slopeZ, stepX, stepY, stepZ, stepTick, permaSpin, spinAgain,
		slopeX, slopeY, velocityX, velocityY
}

func extractNibbleField(
	packedRocks0, packedRocks1, packedRocks2, packedRocks3, mask simd.Uint32s,
	shift uint8,
) simd.Uint8s {
	field0 := shiftNibbleRight(packedRocks0.And(mask), shift)
	field1 := shiftNibbleRight(packedRocks1.And(mask), shift).ShiftAllLeft(8)
	field2 := shiftNibbleRight(packedRocks2.And(mask), shift).ShiftAllLeft(16)
	field3 := shiftNibbleRight(packedRocks3.And(mask), shift).ShiftAllLeft(24)
	return field0.Or(field1).Or(field2).Or(field3).ReshapeToUint8s()
}

func extractSignedNibbleField(
	packedRocks0, packedRocks1, packedRocks2, packedRocks3, mask simd.Uint32s,
	shift uint8,
) simd.Int8s {
	code := extractNibbleField(
		packedRocks0, packedRocks1, packedRocks2, packedRocks3, mask, shift)
	return code.Sub(nibbleEightU8).BitsToInt8().Masked(code.NotEqual(nibbleZeroU8))
}

func unpackNibbleCoordinates(
	packedPositions0, packedPositions1, packedPositions2, packedPositions3, mask simd.Uint32s,
	shift uint8,
) (simd.Uint32s, simd.Uint32s, simd.Uint32s, simd.Uint32s) {
	return shiftNibbleRight(packedPositions0.And(mask), shift),
		shiftNibbleRight(packedPositions1.And(mask), shift),
		shiftNibbleRight(packedPositions2.And(mask), shift),
		shiftNibbleRight(packedPositions3.And(mask), shift)
}

func widenInt8Lanes(values simd.Int8s) (simd.Int32s, simd.Int32s, simd.Int32s, simd.Int32s) {
	return int8LanesToInt32Group(values, 0), int8LanesToInt32Group(values, 1),
		int8LanesToInt32Group(values, 2), int8LanesToInt32Group(values, 3)
}

func uint8LanesToUint32Group(values simd.Uint8s, group uint8) simd.Uint32s {
	bytes := values.ReshapeToUint32s()
	switch group {
	case 0:
		return bytes.And(nibbleByteMask)
	case 1:
		return bytes.ShiftAllRight(8).And(nibbleByteMask)
	case 2:
		return bytes.ShiftAllRight(16).And(nibbleByteMask)
	case 3:
		return bytes.ShiftAllRight(24).And(nibbleByteMask)
	}
	return simd.Uint32s{}
}

func int8LanesToInt32Group(values simd.Int8s, group uint8) simd.Int32s {
	return uint8LanesToUint32Group(values.ToBits(), group).
		ShiftAllLeft(24).BitsToInt32().ShiftAllRight(24)
}

func shiftNibbleRight(values simd.Uint32s, shift uint8) simd.Uint32s {
	switch shift {
	case 0:
		return values
	case 1:
		return values.ShiftAllRight(1)
	case 2:
		return values.ShiftAllRight(2)
	case 6:
		return values.ShiftAllRight(6)
	case 10:
		return values.ShiftAllRight(10)
	case 12:
		return values.ShiftAllRight(12)
	case 13:
		return values.ShiftAllRight(13)
	case 16:
		return values.ShiftAllRight(16)
	case 20:
		return values.ShiftAllRight(20)
	case 24:
		return values.ShiftAllRight(24)
	case 28:
		return values.ShiftAllRight(28)
	}
	return values
}

func canonicalNibblePermaStepZ(stepZ simd.Uint8s) simd.Uint8s {
	fullSpin := stepZ.Equal(nibbleValuesU8[15])
	xDirection := stepZ.And(nibbleValuesU8[3])
	yDirection := stepZ.And(nibbleValuesU8[12])
	xDirection = nibbleZeroU8.IfElse(xDirection.Equal(nibbleValuesU8[3]), xDirection)
	yDirection = nibbleZeroU8.IfElse(yDirection.Equal(nibbleValuesU8[12]), yDirection)
	yDirection = nibbleZeroU8.IfElse(xDirection.NotEqual(nibbleZeroU8), yDirection)
	return nibbleValuesU8[15].IfElse(fullSpin, xDirection.Or(yDirection))
}

func nibbleSlopeDirection(slope, velocity simd.Int8s) (simd.Mask8s, simd.Mask8s) {
	forward := velocity.Sub(slope)
	forward = forward.Add(nibbleCycleI8).IfElse(forward.Less(nibbleZeroI8), forward)
	increment := forward.Greater(nibbleZeroI8).And(forward.LessEqual(nibbleHalfI8))
	return increment, forward.Greater(nibbleHalfI8)
}

func moveNibbleSlope(slope, velocity simd.Int8s, step simd.Mask8s) simd.Int8s {
	increment, decrement := nibbleSlopeDirection(slope, velocity)
	slope = incrementNibbleSlope(slope).IfElse(increment.And(step), slope)
	return decrementNibbleSlope(slope).IfElse(decrement.And(step), slope)
}

func incrementNibbleSlope(slope simd.Int8s) simd.Int8s {
	incremented := slope.Add(nibbleOneI8)
	return nibbleMinI8.IfElse(incremented.Greater(nibbleMaxI8), incremented)
}

func decrementNibbleSlope(slope simd.Int8s) simd.Int8s {
	decremented := slope.Sub(nibbleOneI8)
	return nibbleMaxI8.IfElse(decremented.Less(nibbleMinI8), decremented)
}

func updateNibbleSlopeZ(slopeZ simd.Uint8s, velocityX, velocityY simd.Int8s) simd.Uint8s {
	useX := velocityX.Abs().GreaterEqual(velocityY.Abs())
	dominant := velocityX.IfElse(useX, velocityY)
	increment := dominant.Less(nibbleZeroI8)
	decrement := dominant.Greater(nibbleZeroI8)
	incremented := slopeZ.Add(nibbleOneU8)
	incremented = nibbleZeroU8.IfElse(incremented.Equal(nibbleSixteenU8), incremented)
	decremented := slopeZ.Sub(nibbleOneU8)
	decremented = nibbleValuesU8[15].IfElse(slopeZ.Equal(nibbleZeroU8), decremented)
	slopeZ = incremented.IfElse(increment, slopeZ)
	return decremented.IfElse(decrement, slopeZ)
}

func dampNibbleVelocity(velocity simd.Int8s) simd.Int8s {
	velocity = velocity.Sub(nibbleOneI8.Masked(velocity.Greater(nibbleZeroI8)))
	return velocity.Add(nibbleOneI8.Masked(velocity.Less(nibbleZeroI8)))
}
