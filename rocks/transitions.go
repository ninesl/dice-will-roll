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
