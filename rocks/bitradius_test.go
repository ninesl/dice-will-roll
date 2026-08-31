package rocks

import (
	"simd"
	"testing"
)

func TestPackedRadiusFormulas(t *testing.T) {
	hover := [...][16]uint16{
		{0, 28, 33, 37, 42, 46, 50, 55, 59, 64, 68, 73, 77, 82, 86, 90},
		{0, 23, 26, 30, 33, 37, 40, 44, 48, 51, 55, 58, 62, 65, 69, 72},
		{0, 23, 26, 30, 33, 37, 40, 44, 48, 51, 55, 58, 62, 65, 69, 72},
		{0, 12, 13, 15, 17, 19, 20, 22, 24, 26, 28, 29, 31, 33, 35, 36},
		{0, 6, 7, 8, 9, 10, 10, 11, 12, 13, 14, 15, 16, 17, 18, 18},
		{0, 3, 3, 3, 4, 4, 4, 5, 5, 6, 6, 6, 7, 7, 7, 8},
	}
	collision := [...][16]uint16{
		{0, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49, 52, 55, 58, 60},
		{0, 15, 18, 20, 22, 25, 27, 30, 32, 34, 37, 39, 41, 44, 46, 48},
		{0, 15, 18, 20, 22, 25, 27, 30, 32, 34, 37, 39, 41, 44, 46, 48},
		{0, 8, 9, 10, 11, 13, 14, 15, 16, 17, 19, 20, 21, 22, 23, 24},
		{0, 4, 5, 5, 6, 7, 7, 8, 8, 9, 10, 10, 11, 11, 12, 12},
		{0, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5},
	}
	for amountScale := range hover {
		for size := range hover[amountScale] {
			lane := simd.BroadcastUint16s(uint16(size))
			constants := &rockUpdateConstants[amountScale]
			if got := firstUint16Lane(hoverRadius(lane, constants)); got != hover[amountScale][size] {
				t.Errorf("hoverRadius(%d, %d) = %d; want %d", size, amountScale, got, hover[amountScale][size])
			}
			if got := firstUint16Lane(collisionRadius(lane, constants)); got != collision[amountScale][size] {
				t.Errorf("collisionRadius(%d, %d) = %d; want %d", size, amountScale, got, collision[amountScale][size])
			}
		}
	}
}

func TestMouseMagnitudeBands(t *testing.T) {
	for amountScale := range rockUpdateConstants {
		constants := &rockUpdateConstants[amountScale]
		radius := firstUint16Lane(constants.mouseRadius)
		for distance := uint16(0); distance <= radius; distance++ {
			want := uint16(mouseForceBands)
			for band := uint16(1); band < mouseForceBands; band++ {
				if distance*mouseForceBands >= radius*band {
					want = mouseForceBands - band
				}
			}
			lane := simd.BroadcastUint16s(distance)
			if got := firstUint16Lane(closeButtonMagnitude16(lane, &constants.buttonThresholds)); got != want {
				t.Fatalf("button magnitude scale %d distance %d = %d; want %d", amountScale, distance, got, want)
			}
			if got := firstUint16Lane(closeHoverMagnitude16(lane, constants.mouseRadius)); got != want {
				t.Fatalf("hover magnitude scale %d distance %d = %d; want %d", amountScale, distance, got, want)
			}
		}
	}
}

func firstUint16Lane(vector simd.Uint16s) uint16 {
	values := make([]uint16, 1)
	vector.StorePart(values)
	return values[0]
}
