package rocks

const (
	mortonRadixBits = 12
	mortonRadixSize = 1 << mortonRadixBits
	mortonRadixMask = mortonRadixSize - 1
)

// SortByMortonPosition establishes a stable, quadrant-local painter order.
func (rocks *Rocks) SortByMortonPosition() {
	count := rocks.Len()
	if len(rocks.PosY) != count || len(rocks.Slope) != count || len(rocks.Stepping) != count {
		panic("rocks: packed streams have different lengths")
	}
	if count < 2 {
		rocks.InvalidateLayers()
		return
	}

	keys := make([]uint32, count)
	temporaryKeys := make([]uint32, count)
	for i := range keys {
		keys[i] = mortonPositionKey(rocks.PosX[i], rocks.PosY[i])
	}
	temporary := Rocks{
		PosX:     make([]uint16, count),
		PosY:     make([]uint16, count),
		Slope:    make([]uint16, count),
		Stepping: make([]uint16, count),
	}

	radixSortRockPass(rocks, &temporary, keys, temporaryKeys, 0)
	radixSortRockPass(&temporary, rocks, temporaryKeys, keys, mortonRadixBits)
	rocks.drawGroups = nil
	rocks.InvalidateLayers()
}

func radixSortRockPass(source, target *Rocks, sourceKeys, targetKeys []uint32, shift uint) {
	var offsets [mortonRadixSize]int
	for _, key := range sourceKeys {
		offsets[int(key>>shift)&mortonRadixMask]++
	}
	total := 0
	for bucket, size := range offsets {
		offsets[bucket], total = total, total+size
	}
	for sourceIndex, key := range sourceKeys {
		bucket := int(key>>shift) & mortonRadixMask
		targetIndex := offsets[bucket]
		offsets[bucket]++
		targetKeys[targetIndex] = key
		target.PosX[targetIndex] = source.PosX[sourceIndex]
		target.PosY[targetIndex] = source.PosY[sourceIndex]
		target.Slope[targetIndex] = source.Slope[sourceIndex]
		target.Stepping[targetIndex] = source.Stepping[sourceIndex]
	}
}

func mortonPositionKey(packedX, packedY uint16) uint32 {
	return spreadMortonCoordinate(packedX&packedCoordinateMask) |
		spreadMortonCoordinate(packedY&packedCoordinateMask)<<1
}

func spreadMortonCoordinate(coordinate uint16) uint32 {
	value := uint32(coordinate)
	value = (value | value<<8) & 0x00FF00FF
	value = (value | value<<4) & 0x0F0F0F0F
	value = (value | value<<2) & 0x33333333
	return (value | value<<1) & 0x55555555
}
