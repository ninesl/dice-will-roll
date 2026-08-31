package rocks

import (
	"sort"
	"testing"
)

func TestSortByMortonPositionKeepsStreamsTogetherAndTiesStable(t *testing.T) {
	positions := [][2]uint32{{2, 0}, {1, 0}, {0, 1}, {0, 0}, {1, 0}}
	identities := []uint16{50, 20, 30, 10, 21}
	rocks := Rocks{}
	for i, position := range positions {
		packedX, packedY := PackPosition(position[0], position[1], int32(i+1), -int32(i+1))
		rocks.PosX = append(rocks.PosX, packedX)
		rocks.PosY = append(rocks.PosY, packedY)
		rocks.Slope = append(rocks.Slope, identities[i])
		rocks.Stepping = append(rocks.Stepping, identities[i]+100)
	}

	rocks.SortByMortonPosition()

	want := []uint16{10, 20, 21, 30, 50}
	for i, identity := range want {
		if rocks.Slope[i] != identity || rocks.Stepping[i] != identity+100 {
			t.Fatalf("rock %d streams = (%d, %d); want identity %d", i,
				rocks.Slope[i], rocks.Stepping[i], identity)
		}
		if i > 0 && mortonPositionKey(rocks.PosX[i-1], rocks.PosY[i-1]) >
			mortonPositionKey(rocks.PosX[i], rocks.PosY[i]) {
			t.Fatalf("Morton keys are not sorted at index %d", i)
		}
	}
	if !rocks.redraw {
		t.Fatal("sort did not invalidate cached layers")
	}
}

func TestMortonPositionKeyIgnoresPackedVelocity(t *testing.T) {
	firstX, firstY := PackPosition(123, 456, -7, 7)
	secondX, secondY := PackPosition(123, 456, 5, -4)
	if mortonPositionKey(firstX, firstY) != mortonPositionKey(secondX, secondY) {
		t.Fatal("Morton key depends on packed velocity")
	}
}

func TestSortByMortonPositionMatchesStableReference(t *testing.T) {
	const count = 10_000
	type referenceRock struct {
		key      uint32
		identity uint16
	}
	reference := make([]referenceRock, count)
	rocks := Rocks{
		PosX:     make([]uint16, count),
		PosY:     make([]uint16, count),
		Slope:    make([]uint16, count),
		Stepping: make([]uint16, count),
	}
	for i := range count {
		x := uint32(i*4051+i*i*17) & uint32(packedCoordinateMask)
		y := uint32(i*3163+i*i*29) & uint32(packedCoordinateMask)
		velocityX := int32(i%15 - 7)
		velocityY := int32((i*7)%15 - 7)
		rocks.PosX[i], rocks.PosY[i] = PackPosition(x, y, velocityX, velocityY)
		rocks.Slope[i] = uint16(i)
		rocks.Stepping[i] = ^uint16(i)
		reference[i] = referenceRock{
			key:      mortonPositionKey(rocks.PosX[i], rocks.PosY[i]),
			identity: uint16(i),
		}
	}
	sort.SliceStable(reference, func(i, j int) bool {
		return reference[i].key < reference[j].key
	})

	rocks.SortByMortonPosition()

	for i, want := range reference {
		if key := mortonPositionKey(rocks.PosX[i], rocks.PosY[i]); key != want.key {
			t.Fatalf("rock %d key = %d; want %d", i, key, want.key)
		}
		if rocks.Slope[i] != want.identity || rocks.Stepping[i] != ^want.identity {
			t.Fatalf("rock %d streams do not match identity %d", i, want.identity)
		}
	}
}

func BenchmarkSortByMortonPositionMillion(b *testing.B) {
	const count = 1_000_000
	rocks := Rocks{
		PosX:     make([]uint16, count),
		PosY:     make([]uint16, count),
		Slope:    make([]uint16, count),
		Stepping: make([]uint16, count),
	}
	for i := range count {
		rocks.PosX[i] = uint16(i*4051) & packedCoordinateMask
		rocks.PosY[i] = uint16(i*3163) & packedCoordinateMask
		rocks.Slope[i] = uint16(i)
		rocks.Stepping[i] = uint16(i >> 16)
	}

	b.ResetTimer()
	for range b.N {
		rocks.SortByMortonPosition()
	}
}
