package rocks

import (
	"simd"
	"testing"

	"github.com/ninesl/dice-will-roll/controls"
	"github.com/ninesl/dice-will-roll/settings"
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

func TestUpdateRocksTracksActiveDrawGroups(t *testing.T) {
	Init(settings.ScreenSettings{ResolutionX: 2000, ResolutionY: 2000})
	rockCount := RocksPerPackedVector*2 + 1
	rocks := Rocks{
		PosX:     make([]uint16, rockCount),
		PosY:     make([]uint16, rockCount),
		Slope:    make([]uint16, rockCount),
		Stepping: make([]uint16, rockCount),
	}
	for i := range rockCount {
		rocks.PosX[i], rocks.PosY[i] = PackPosition(500, 500, 0, 0)
		rocks.Slope[i], rocks.Stepping[i] = PackSprite(0, 0, 0, 1, 0, 0, 0, 0, 0, 0)
	}
	rocks.PosX[0], rocks.PosY[0] = PackPosition(500, 500, 2, 0)
	rocks.Stepping[RocksPerPackedVector] = 1

	UpdateRocks(&rocks, 0, 0, controls.MouseInfo{})
	assertDrawGroups(t, rocks.drawGroups, []bool{true, false, false})

	UpdateRocks(&rocks, 0, 1, controls.MouseInfo{})
	assertDrawGroups(t, rocks.drawGroups, []bool{true, true, false})

	positionX, positionY, _, _ := UnpackPosition(rocks.PosX[0], rocks.PosY[0])
	rocks.PosX[0], rocks.PosY[0] = PackPosition(uint32(positionX), uint32(positionY), 0, 0)
	rocks.Stepping[0] = 0
	UpdateRocks(&rocks, 0, 0, controls.MouseInfo{})
	assertDrawGroups(t, rocks.drawGroups, []bool{false, true, false})

	last := rockCount - 1
	rocks.PosX[last], rocks.PosY[last] = PackPosition(500, 500, 2, 0)
	UpdateRocks(&rocks, 0, 1, controls.MouseInfo{})
	assertDrawGroups(t, rocks.drawGroups, []bool{false, true, true})
}

func TestUpdateRocksTracksVisualChanges(t *testing.T) {
	Init(settings.ScreenSettings{ResolutionX: 2000, ResolutionY: 2000})
	packedX, packedY := PackPosition(500, 500, 0, 0)
	packedSlope, packedStepping := PackSprite(0, 0, 0, 1, 0, 0, 0, 0, 0, 0)
	rocks := Rocks{
		PosX:     []uint16{packedX},
		PosY:     []uint16{packedY},
		Slope:    []uint16{packedSlope},
		Stepping: []uint16{packedStepping},
	}

	UpdateRocks(&rocks, 0, 1, controls.MouseInfo{})
	if !rocks.redraw {
		t.Fatal("initial update did not request a redraw")
	}

	rocks.redraw = false
	UpdateRocks(&rocks, 0, 1, controls.MouseInfo{})
	if rocks.redraw {
		t.Fatal("stationary update requested a redraw")
	}

	rocks.PosX[0], rocks.PosY[0] = PackPosition(500, 500, 2, 0)
	UpdateRocks(&rocks, 0, 1, controls.MouseInfo{})
	if rocks.redraw {
		t.Fatal("visible SIMD update requested a complete redraw")
	}
	if len(rocks.dirtyLayers) != 1 || !rocks.dirtyLayers[0] {
		t.Fatal("visible SIMD update did not dirty its layer")
	}
}

func assertDrawGroups(t *testing.T, got, want []bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("draw group count = %d; want %d", len(got), len(want))
	}
	for group := range want {
		if got[group] != want[group] {
			t.Errorf("draw group %d active = %t; want %t", group, got[group], want[group])
		}
	}
}
