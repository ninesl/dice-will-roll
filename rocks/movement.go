package rocks

import (
	"math"
)

// Update updates the rock based on the current target transitions.
// Will update it's state based on other params every Tick/time this is called
// NOTE: Damping is NOT applied here - it happens after all collisions in ApplyDamping()
func (r *SimpleRock) Update(frameCounter int) {
	r.Position.Y += BaseVelocity * float32(r.SlopeY)
	r.Position.X += BaseVelocity * float32(r.SlopeX)

	r.UpdateTransition(frameCounter)
}

// ApplyDamping gradually reduces rock velocity over time
// This is called AFTER all collision handling to ensure bounces get full velocity
// Damping rate varies by rock size: smaller rocks slow down faster
func (r *SimpleRock) ApplyDamping(frameCounter int) {
	// Skip if rock is already stopped
	if r.SlopeX == 0 && r.SlopeY == 0 {
		return
	}

	// Damping rate based on rock size (smaller rocks = faster damping)
	// Small rocks: every 14 frames, Medium: 16, Big: 18, Huge: 20
	dampingCycle := ROCK_DAMPING_FRAME_CYCLE - int(r.Score.GetScore())/2
	// if dampingCycle < 10 {
	// 	dampingCycle = 10 // Minimum cycle to prevent too-fast damping
	// }

	if frameCounter%dampingCycle == 0 {
		// Gradually reduce slope toward zero
		if r.SlopeX > 0 {
			r.SlopeX--
		} else if r.SlopeX < 0 {
			r.SlopeX++
		}

		if r.SlopeY > 0 {
			r.SlopeY--
		} else if r.SlopeY < 0 {
			r.SlopeY++
		}
	}
}

// Bounce sets rock direction and speed with transition animation
// if newY or newX is IDENTICAL to the current value in the struct,
// does the go compiler ignore this if the func is called? I want to reuse this in BounceX and BounceY but it seems to be
// a wasted allocation if we were to assign newX newY each time?
func (r *SimpleRock) Bounce(newX int8, newY int8) {
	r.SlopeX = newX
	r.SlopeY = newY

	// Calculate transition steps for X
	targetX := newX + MAX_SLOPE
	if targetX == DIRECTIONS_TO_SNAP {
		targetX = 0
	}
	distX := calculateShortestPath(r.SpriteSlopeX, targetX)

	// Calculate transition steps for Y
	targetY := newY + MAX_SLOPE
	if targetY == DIRECTIONS_TO_SNAP {
		targetY = 0
	}
	distY := calculateShortestPath(r.SpriteSlopeY, targetY)

	// Add full rotation (DIRECTIONS_TO_SNAP) to shortest path
	r.setTransitionSteps(distX+uint8(DIRECTIONS_TO_SNAP), distY+uint8(DIRECTIONS_TO_SNAP))
}

// BounceX flips horizontal direction (bounce off vertical wall)
// Rock spins around Y-axis (sideways tumble) based on Y speed
func (r *SimpleRock) BounceX() {
	r.SlopeX = -r.SlopeX

	// X needs to transition to new target (shortest path, no extra rotation)
	targetX := r.SlopeX + MAX_SLOPE
	if targetX == DIRECTIONS_TO_SNAP {
		targetX = 0
	}
	distX := calculateShortestPath(r.SpriteSlopeX, targetX)

	// When bouncing off vertical wall, tumble around Y-axis based on Y speed
	// Y slope doesn't change, so just tumble
	absY := r.SlopeY
	if absY < 0 {
		absY = -absY
	}
	tumbleY := uint8(absY) * 2 // 0-8 range, scaled by speed

	r.setTransitionSteps(distX, tumbleY)
}

// BounceY flips vertical direction (bounce off horizontal wall)
// Rock spins around X-axis (forward/backward tumble) based on X speed
func (r *SimpleRock) BounceY() {
	r.SlopeY = -r.SlopeY

	// Y needs to transition to new target (shortest path, no extra rotation)
	targetY := r.SlopeY + MAX_SLOPE
	if targetY == DIRECTIONS_TO_SNAP {
		targetY = 0
	}
	distY := calculateShortestPath(r.SpriteSlopeY, targetY)

	// When bouncing off horizontal wall, tumble around X-axis based on X speed
	// X slope doesn't change, so just tumble
	absX := r.SlopeX
	if absX < 0 {
		absX = -absX
	}
	tumbleX := uint8(absX) * 2 // 0-8 range, scaled by speed

	r.setTransitionSteps(tumbleX, distY)
}

// BounceTowardsAngle sets rock direction and speed based on target angle (0-360 degrees)
// Angle system: 0°=right, 90°=down, 180°=left, 270°=up (standard screen coordinates)
// Uses trigonometry to accurately map angles to SlopeX/SlopeY components
func (r *SimpleRock) BounceTowardsAngle(angle int) {
	// Convert angle to radians for trigonometric functions
	angleRad := float64(angle) * math.Pi / 180.0

	// Calculate velocity components using trigonometry
	speedX := math.Cos(angleRad)
	speedY := math.Sin(angleRad)

	// Scale to int8 range (-4 to +4) while maintaining ratio
	const maxVal = float64(MAX_SLOPE) // 4

	// Scale both components so the larger one reaches maxVal
	maxComponent := math.Max(math.Abs(speedX), math.Abs(speedY))
	scale := maxVal / maxComponent
	targetSlopeX := int8(math.Round(speedX * scale))
	targetSlopeY := int8(math.Round(speedY * scale))

	r.Bounce(targetSlopeX, targetSlopeY)
}

// SizeData returns pre-computed size data for this rock
// This is MUCH faster than GetSize() as it's a simple array lookup
func (r *SimpleRock) SizeData() RockSizeData {
	return rockSizeLookup[r.Score]
}

// GetSize returns the pixel size of this rock based on its RockScoreType
// DEPRECATED: Use SizeData().Size for better performance
func (r *SimpleRock) GetSize(baseSpriteSize float32) float32 {
	return baseSpriteSize * r.Score.SizeMultiplier()
}
