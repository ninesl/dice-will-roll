package rocks

import (
	"simd"
	"testing"

	"github.com/ninesl/dice-will-roll/settings"
)

func TestCollideWall16UsesScreenEdges(t *testing.T) {
	tests := []struct {
		name        string
		position    int16
		velocity    int16
		extent      int16
		wantOverlap int16
		wantHit     bool
	}{
		{"inside minimum edge", 2, -1, 100, 0, false},
		{"at minimum edge", 2, -2, 100, 0, true},
		{"past minimum edge", 2, -3, 100, 1, true},
		{"inside maximum edge", 98, 1, 100, 0, false},
		{"at maximum edge", 98, 2, 100, 0, true},
		{"past maximum edge", 98, 3, 100, 1, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			overlap, hit := collideWall16(
				simd.BroadcastInt16s(test.position),
				simd.BroadcastInt16s(test.velocity),
				simd.BroadcastInt16s(test.extent),
			)
			overlaps := make([]int16, RocksPerPackedVector)
			hits := make([]int16, RocksPerPackedVector)
			overlap.Store(overlaps)
			hit.ToInt16s().Store(hits)
			if overlaps[0] != test.wantOverlap {
				t.Errorf("overlap = %d; want %d", overlaps[0], test.wantOverlap)
			}
			if got := hits[0] != 0; got != test.wantHit {
				t.Errorf("hit = %t; want %t", got, test.wantHit)
			}
		})
	}
}

func TestUpdateRockGroup16BouncePreservesPositionOnHitAxis(t *testing.T) {
	Init(settings.ScreenSettings{ResolutionX: 100, ResolutionY: 100})
	packedX, packedY := PackPosition(2, 50, -2, 3)
	packedSlope, packedAnimate := PackSprite(0, 0, 0, 1, 0, 0, 0, 0, 0, 0)
	state := rockUpdateInput16{
		constants: &rockUpdateConstants[0],
		routine:   rockUpdateDisabled,
	}

	gotX, gotY, _, _ := updateRockGroup16(
		simd.BroadcastUint16s(packedX),
		simd.BroadcastUint16s(packedY),
		simd.BroadcastUint16s(packedSlope),
		simd.BroadcastUint16s(packedAnimate),
		&state,
	)
	xValues := make([]uint16, RocksPerPackedVector)
	yValues := make([]uint16, RocksPerPackedVector)
	gotX.Store(xValues)
	gotY.Store(yValues)
	positionX, positionY, velocityX, velocityY := UnpackPosition(xValues[0], yValues[0])
	if positionX != 2 || positionY != 53 {
		t.Errorf("position = (%d, %d); want (2, 53)", positionX, positionY)
	}
	if velocityX != 2 || velocityY != 3 {
		t.Errorf("velocity = (%d, %d); want (2, 3)", velocityX, velocityY)
	}
}
