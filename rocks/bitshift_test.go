package rocks

import (
	"simd"
	"testing"
)

func TestPackAndUnpackPosition(t *testing.T) {
	packed := PackPosition(1920, 1080, -3, 5)
	positionX, positionY, slopeX, slopeY := UnpackPosition(packed)
	if positionX != 1920 || positionY != 1080 || slopeX != -3 || slopeY != 5 {
		t.Fatalf("UnpackPosition(PackPosition(...)) = (%d, %d, %d, %d)", positionX, positionY, slopeX, slopeY)
	}
}

func TestUnpackSprite(t *testing.T) {
	size, rotation, slopeX, slopeY := UnpackSprite(PackSprite(8, 10, -3, 4))
	if size != 8 || rotation != 10 || slopeX != -3 || slopeY != 4 {
		t.Fatalf("UnpackSprite() = (%d, %d, %d, %d)", size, rotation, slopeX, slopeY)
	}
}

func TestInitRockScales(t *testing.T) {
	scales := initRockScales()
	if scales[0] != 0 || scales[1] != 0.2 || scales[15] != 1.2 {
		t.Fatalf("scale endpoints = (%f, %f, %f)", scales[0], scales[1], scales[15])
	}
	for sizeCode := 2; sizeCode < len(scales); sizeCode++ {
		if scales[sizeCode] <= scales[sizeCode-1] {
			t.Fatalf("scale %d (%f) must exceed scale %d (%f)", sizeCode, scales[sizeCode], sizeCode-1, scales[sizeCode-1])
		}
	}
}

func TestTrailSpriteSlope(t *testing.T) {
	tests := []struct {
		name          string
		spriteSlope   int32
		positionSlope int32
		want          int32
	}{
		{"positive seven rolls to negative seven", 7, -7, -7},
		{"negative seven rolls to positive seven", -7, 7, 7},
		{"negative seven trails backward to positive six", -7, 6, 7},
		{"positive one trails toward negative one", 1, -1, 0},
		{"stationary trails toward negative one", 0, -1, -1},
		{"negative one trails toward positive one", -1, 1, 0},
		{"stationary trails toward positive one", 0, 1, 1},
		{"matching slope remains unchanged", 3, 3, 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spriteSlopes := make([]int32, SIMDVectorSize)
			positionSlopes := make([]int32, SIMDVectorSize)
			got := make([]int32, SIMDVectorSize)
			for lane := range SIMDVectorSize {
				spriteSlopes[lane] = test.spriteSlope
				positionSlopes[lane] = test.positionSlope
			}

			trailSpriteSlope(
				simd.LoadInt32s(spriteSlopes),
				simd.LoadInt32s(positionSlopes),
			).Store(got)

			for lane, slope := range got {
				if slope != test.want {
					t.Fatalf("lane %d: trailSpriteSlope(%d, %d) = %d; want %d",
						lane, test.spriteSlope, test.positionSlope, slope, test.want)
				}
			}
		})
	}
}
