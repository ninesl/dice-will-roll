package rocks

// getTransitionStepsX returns the remaining transition steps for X axis
func (r *SimpleRock) getTransitionStepsX() uint8 {
	return r.transitionSteps & 0x0F
}

// getTransitionStepsY returns the remaining transition steps for Y axis
func (r *SimpleRock) getTransitionStepsY() uint8 {
	return (r.transitionSteps >> 4) & 0x0F
}

// setTransitionSteps sets both X and Y transition steps (each must be 0-15)
func (r *SimpleRock) setTransitionSteps(stepsX, stepsY uint8) {
	r.transitionSteps = (stepsY << 4) | (stepsX & 0x0F)
}

// decrementTransitionStepX decrements X transition counter if > 0
func (r *SimpleRock) decrementTransitionStepX() {
	if r.getTransitionStepsX() > 0 {
		r.transitionSteps--
	}
}

// decrementTransitionStepY decrements Y transition counter if > 0
func (r *SimpleRock) decrementTransitionStepY() {
	if r.getTransitionStepsY() > 0 {
		r.transitionSteps -= 0x10 // Subtract from upper 4 bits
	}
}

// calculateShortestPath calculates the shortest rotational distance between two sprite slope values
func calculateShortestPath(current, target int8) (distance uint8) {
	diff := target - current

	if diff > DIRECTIONS_TO_SNAP/2 {
		// Going backwards is shorter
		distance = uint8(DIRECTIONS_TO_SNAP - diff)
	} else if diff < -DIRECTIONS_TO_SNAP/2 {
		// Going forwards is shorter
		distance = uint8(DIRECTIONS_TO_SNAP + diff)
	} else if diff > 0 {
		// Normal forward
		distance = uint8(diff)
	} else if diff < 0 {
		// Normal backward
		distance = uint8(-diff)
	} else {
		// Already at target
		distance = 0
	}

	return distance
}

// UpdateTransition handles the smooth sprite rotation during direction changes
// Now includes full rotation on each bounce
func (r *SimpleRock) UpdateTransition(frameCounter int) {
	// Update SpriteIndex based on rock SIZE (smaller rocks rotate faster)
	if frameCounter%r.Score.GetScore() == 0 {
		// Increment or decrement sprite index based on horizontal direction
		// Moving right (positive SlopeX): increment (rotate clockwise)
		// Moving left (negative SlopeX): decrement (rotate counter-clockwise)
		if r.SlopeX != 0 && r.SlopeY != 0 {
			if r.SlopeX >= 0 {
				if r.SpriteIndex == 0 {
					r.SpriteIndex = ROTATION_FRAMES - 1
				} else {
					r.SpriteIndex--
				}
			} else {
				r.SpriteIndex++
				if r.SpriteIndex >= ROTATION_FRAMES {
					r.SpriteIndex = 0
				}
			}
		}
		//  else {
		// Stationary rocks: cycle through transition slopes for visual variety
		// r.SpriteSlopeX++
		// if r.SpriteSlopeX >= DIRECTIONS_TO_SNAP {
		// r.SpriteSlopeX = 0
		// r.SpriteSlopeY++
		// if r.SpriteSlopeY >= DIRECTIONS_TO_SNAP {
		// r.SpriteSlopeY = 0
		// }
		// }
		// }
	}

	if r.SlopeX == 0 && r.SlopeY == 0 {
		return
	}

	// Update SpriteSlopeX/Y based on SPEED (for smooth transitions during bounces)
	// Derive animation rate on-the-fly from current slopes (faster than storing/loading from memory)
	// Fast approximation: max(|x|, |y|) + min(|x|, |y|)/2
	absX := r.SlopeX
	if absX < 0 {
		absX = -absX
	}
	absY := r.SlopeY
	if absY < 0 {
		absY = -absY
	}

	var speed int8
	if absX > absY {
		speed = absX + absY/2
	} else {
		speed = absY + absX/2
	}

	n := int(baseN) - int(speed)*3 // Simplified from speed*speedFactor (3.5 → 3 for int math)
	if n < 2 {
		n = 2
	}

	if frameCounter%n != 0 {
		return
	}

	// If rock is completely stopped, don't update sprite slopes
	// Keep SpriteSlopeX/Y frozen at current values
	if r.SlopeX == 0 && r.SlopeY == 0 {
		return
	}

	// Always update sprite slopes by +1 (transitions always increment)
	// X component
	if r.getTransitionStepsX() > 0 {
		r.SpriteSlopeX++
		if r.SpriteSlopeX >= DIRECTIONS_TO_SNAP {
			r.SpriteSlopeX = 0
		}
		r.decrementTransitionStepX()
	}

	// Y component
	if r.getTransitionStepsY() > 0 {
		r.SpriteSlopeY++
		if r.SpriteSlopeY >= DIRECTIONS_TO_SNAP {
			r.SpriteSlopeY = 0
		}
		r.decrementTransitionStepY()
	}
}
