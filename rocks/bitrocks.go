package rocks

import (
	"fmt"
	"simd"

	"github.com/ninesl/dice-will-roll/settings"
)

const (
	packedCoordinateMask  uint16 = 0x0FFF
	packedVelocityMask    uint16 = 0xF000
	packedSlopeXMask      uint16 = 0xF000
	packedSlopeYMask      uint16 = 0x0F00
	spriteSlopeZMask16    uint16 = 0x00F0
	spriteSizeScoreMask16 uint16 = 0x000F
	stepYMask16           uint16 = 0xE000
	stepXMask16           uint16 = 0x1C00
	stepZMask16           uint16 = 0x03C0
	stepTickMask16        uint16 = 0x003C
	permaSpinMask16       uint16 = 0x0002
	spinAgainMask16       uint16 = 0x0001
	permaXDirectionMask32 uint32 = 0x0000_0003
	permaYDirectionMask32 uint32 = 0x0000_000C
)

type Rocks struct {
	PosX    []uint16
	PosY    []uint16
	Slope   []uint16
	Animate []uint16
}

func (rocks Rocks) Len() int { return len(rocks.PosX) }

var (
	RocksPerPackedVector int

	zeroUint16, oneUint16, maximumSizeUint16, coordinateMask16, nibbleMask16 simd.Uint16s
	maximumStepZ16, maximumSlopeU16, slopeCycleU16, halfSlopeCycleU16        simd.Uint16s

	zeroInt16, oneInt16, minimumSlopeI16, maximumSlopeI16 simd.Int16s
	slopeCycleI16, halfSlopeCycleI16                      simd.Int16s
	nibbleValues16                                        [BitSpriteSlopeCodeCount + 1]simd.Uint16s

	screenWidth16, screenHeight16 simd.Int16s

	collisionLookups   [len(rockAmountScales)][BitSpriteSlopeCodeCount]simd.Uint16s
	hoverRadiusLookups [len(rockAmountScales)][BitSpriteSlopeCodeCount]simd.Uint16s
	mouseRadii         [len(rockAmountScales)]simd.Uint16s
)

func init() {
	RocksPerPackedVector = simd.Uint16s{}.Len()
	zeroUint16 = simd.BroadcastUint16s(0)
	oneUint16 = simd.BroadcastUint16s(1)
	maximumSizeUint16 = simd.BroadcastUint16s(BitSpriteSlopeCodeCount - 1)
	coordinateMask16 = simd.BroadcastUint16s(packedCoordinateMask)
	nibbleMask16 = simd.BroadcastUint16s(0xF)
	maximumStepZ16 = simd.BroadcastUint16s(0xF)
	maximumSlopeU16 = simd.BroadcastUint16s(7)
	slopeCycleU16 = simd.BroadcastUint16s(atlasSlopeStates)
	halfSlopeCycleU16 = simd.BroadcastUint16s(7)

	zeroInt16 = simd.BroadcastInt16s(0)
	oneInt16 = simd.BroadcastInt16s(1)
	minimumSlopeI16 = simd.BroadcastInt16s(-7)
	maximumSlopeI16 = simd.BroadcastInt16s(7)
	slopeCycleI16 = simd.BroadcastInt16s(atlasSlopeStates)
	halfSlopeCycleI16 = simd.BroadcastInt16s(7)
	for value := range nibbleValues16 {
		nibbleValues16[value] = simd.BroadcastUint16s(uint16(value))
	}
}

func Init(screen settings.ScreenSettings) {
	if screen.ResolutionX > int(packedCoordinateMask) || screen.ResolutionY > int(packedCoordinateMask) {
		panic(fmt.Sprintf("rock screen resolution exceeds 12-bit coordinates: %dx%d",
			screen.ResolutionX, screen.ResolutionY))
	}
	screenWidth16 = simd.BroadcastInt16s(int16(screen.ResolutionX))
	screenHeight16 = simd.BroadcastInt16s(int16(screen.ResolutionY))
}
