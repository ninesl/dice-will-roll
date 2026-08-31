package rocks

import (
	"simd"
	"testing"
)

func TestRockGroupNeedsUpdate16(t *testing.T) {
	positionsX := make([]uint16, RocksPerPackedVector)
	positionsY := make([]uint16, RocksPerPackedVector)
	stepping := make([]uint16, RocksPerPackedVector)
	for lane := range positionsX {
		positionsX[lane], positionsY[lane] = PackPosition(500, 500, 0, 0)
	}
	input := rockUpdateInput16{
		constants: &rockUpdateConstants[0],
		routine:   rockUpdateHover,
		mouseX:    simd.BroadcastInt16s(2000),
		mouseY:    simd.BroadcastInt16s(2000),
	}

	needsUpdate := func() bool {
		return rockGroupNeedsUpdate16(
			simd.LoadUint16s(positionsX),
			simd.LoadUint16s(positionsY),
			simd.LoadUint16s(stepping),
			stepping,
			&input,
		)
	}

	if needsUpdate() {
		t.Fatal("stationary group outside mouse broad phase needs update")
	}

	positionsX[0], positionsY[0] = PackPosition(500, 500, 1, 0)
	if !needsUpdate() {
		t.Fatal("group with nonzero velocity was skipped")
	}

	positionsX[0], positionsY[0] = PackPosition(500, 500, 0, 0)
	stepping[0] = 1
	if !needsUpdate() {
		t.Fatal("group with nonzero Stepping was skipped")
	}

	stepping[0] = 0
	positionsX[0], positionsY[0] = PackPosition(2000, 2000, 0, 0)
	if !needsUpdate() {
		t.Fatal("group inside mouse broad phase was skipped")
	}

	stepping[0] = 0
	input.routine = rockUpdateDisabled
	if needsUpdate() {
		t.Fatal("stationary group needs update with mouse disabled")
	}
}
