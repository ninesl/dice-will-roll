package rocks

import "testing"

func TestCalculateRockLayerLayoutUsesSIMDGroupDecades(t *testing.T) {
	tests := []struct {
		name           string
		groupCount     int
		groupsPerLayer int
		layerCount     int
	}{
		{"one group", 1, 10, 1},
		{"ten groups", 10, 10, 1},
		{"eleven groups", 11, 10, 2},
		{"one hundred groups", 100, 10, 10},
		{"one hundred one groups", 101, 100, 2},
		{"one thousand groups", 1000, 100, 10},
		{"one thousand one groups", 1001, 1000, 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			layout := calculateRockLayerLayout(test.groupCount * RocksPerPackedVector)
			if layout.groupCount != test.groupCount {
				t.Errorf("group count = %d; want %d", layout.groupCount, test.groupCount)
			}
			if layout.groupsPerLayer != test.groupsPerLayer {
				t.Errorf("groups per layer = %d; want %d", layout.groupsPerLayer, test.groupsPerLayer)
			}
			if layout.layerCount != test.layerCount {
				t.Errorf("layer count = %d; want %d", layout.layerCount, test.layerCount)
			}
		})
	}
}

func TestCalculateRockLayerLayoutIncludesPartialGroup(t *testing.T) {
	layout := calculateRockLayerLayout(10*RocksPerPackedVector + 1)
	if layout.groupCount != 11 || layout.layerCount != 2 {
		t.Fatalf("layout = %+v; want 11 groups across 2 layers", layout)
	}
}

func TestMarkGroupLayerDirtyMapsWholeGroupsToOneLayer(t *testing.T) {
	rocks := Rocks{
		PosX: make([]uint16, 101*RocksPerPackedVector),
	}
	layout := prepareRockLayerDirtyState(&rocks)
	rocks.markGroupLayerDirty(100)

	if layout.groupsPerLayer != 100 || len(rocks.dirtyLayers) != 2 {
		t.Fatalf("layout = %+v; want 100 groups per layer and 2 layers", layout)
	}
	if rocks.dirtyLayers[0] || !rocks.dirtyLayers[1] {
		t.Fatalf("dirty layers = %v; want only layer 1 dirty", rocks.dirtyLayers)
	}
	if !rocks.anyDirtyLayer {
		t.Fatal("aggregate dirty flag is false; want true")
	}
}
