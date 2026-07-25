package render

// makes it 0.0 - 1.0 for Kage
func normalize(v int) float32 {
	return float32(v) / 255.0
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
