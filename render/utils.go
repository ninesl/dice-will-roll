package render

// RandomXORRockJitter generates 2 psuedo-random numbers in range [-range, +range] using XOR-shift algorithm
// xSeed and ySeed are typically rock position.X and position.Y
// range specifies the jitter range (e.g., range=1 gives [-1,0,1], range=2 gives [-2,-1,0,1,2])
func RandomXORRockJitter(xSeed, ySeed float32, jitterRange int8) (int8, int8) {
	// Convert position to uint32 seed

	var s uint32
	if xSeed > ySeed {
		s = uint32(xSeed) - uint32(ySeed)
	} else {
		s = uint32(xSeed) + uint32(ySeed)
	}

	s = s ^ (s << 13)
	s = s ^ (s >> 7)
	s = s ^ (s << 17)

	// Calculate modulo value: range * 2 + 1 (e.g., range=1 -> 3 values, range=2 -> 5 values)
	modValue := uint32(jitterRange*2 + 1)

	// Extract X jitter from lower bits
	jitterX := int8((s % modValue) - uint32(jitterRange))

	// Extract Y jitter from upper bits
	jitterY := int8(((s >> 8) % modValue) - uint32(jitterRange))

	return jitterX, jitterY
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
