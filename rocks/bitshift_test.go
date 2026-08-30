package rocks

import (
	"simd"
	"testing"

	"github.com/ninesl/dice-will-roll/controls"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/settings"
)

const benchmarkRockCount = 1 << 16

func newBenchmarkRocks() Rocks {
	Init(settings.ScreenSettings{ResolutionX: 4000, ResolutionY: 4000})
	rocks := Rocks{
		PosX: make([]uint16, benchmarkRockCount), PosY: make([]uint16, benchmarkRockCount),
		Slope: make([]uint16, benchmarkRockCount), Animate: make([]uint16, benchmarkRockCount),
	}
	for i := range benchmarkRockCount {
		velocityX := int32(i%15 - 7)
		velocityY := int32((i*7)%15 - 7)
		rocks.PosX[i], rocks.PosY[i] = PackPosition(
			uint32(500+i%3000), uint32(500+(i*3)%3000), velocityX, velocityY)
		rocks.Slope[i], rocks.Animate[i] = PackSprite(
			velocityX, velocityY, uint32(i%16), uint32(i%15+1),
			0, 0, 0, 0, 0, 0)
	}
	mouseRadii[0] = simd.BroadcastUint16s(90)
	return rocks
}

func benchmarkUpdateRocks(b *testing.B, mouse controls.MouseInfo) {
	rocks := newBenchmarkRocks()
	b.ReportAllocs()
	// Each phased call reads and writes eight bytes for half the rocks.
	b.SetBytes(benchmarkRockCount * 8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UpdateRocks(&rocks, 0, i%UpdateStride, mouse)
	}
}

func BenchmarkUpdateRocksSoA(b *testing.B) {
	benchmarkUpdateRocks(b, controls.MouseInfo{})
}

func BenchmarkUpdateRocksAnimated(b *testing.B) {
	rocks := newBenchmarkRocks()
	for i := range rocks.Animate {
		rocks.Animate[i] = permaSpinMask16
	}
	b.ReportAllocs()
	b.SetBytes(benchmarkRockCount * 8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UpdateRocks(&rocks, 0, i%UpdateStride, controls.MouseInfo{})
	}
}

func BenchmarkUpdateRocksHover(b *testing.B) {
	benchmarkUpdateRocks(b, controls.MouseInfo{CursorInfo: controls.CursorInfo{
		Position: render.Vec2{X: 2000, Y: 2000},
	}})
}

func BenchmarkUpdateRocksMouseDown(b *testing.B) {
	benchmarkUpdateRocks(b, controls.MouseInfo{
		CursorInfo: controls.CursorInfo{Position: render.Vec2{X: 2000, Y: 2000}},
		Down:       true,
	})
}

func BenchmarkUpdateRocksRightDown(b *testing.B) {
	benchmarkUpdateRocks(b, controls.MouseInfo{
		CursorInfo: controls.CursorInfo{Position: render.Vec2{X: 2000, Y: 2000}},
		RightDown:  true,
	})
}

func BenchmarkUpdateRocksBothDown(b *testing.B) {
	benchmarkUpdateRocks(b, controls.MouseInfo{
		CursorInfo: controls.CursorInfo{Position: render.Vec2{X: 2000, Y: 2000}},
		Down:       true,
		RightDown:  true,
	})
}
