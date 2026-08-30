package rocks

import (
	"fmt"
	"simd"

	"github.com/ninesl/dice-will-roll/settings"
)

const (
	packedCoordinateMask  uint16 = 0x0FFF
	spriteSizeScoreMask16 uint16 = 0x000F
	permaSpinMask16       uint16 = 0x0002
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
	maximumStepZ16, slopeCycleU16                                            simd.Uint16s

	zeroInt16, oneInt16, minimumSlopeI16, maximumSlopeI16 simd.Int16s
	slopeCycleI16, halfSlopeCycleI16                      simd.Int16s
	nibbleValues16                                        [BitSpriteSlopeCodeCount + 1]simd.Uint16s

	screenWidth16, screenHeight16 simd.Int16s

	hoverRadiusMultiplier, hoverRadiusBias         [len(rockAmountScales)]simd.Uint16s
	collisionRadiusMultiplier, collisionRadiusBias [len(rockAmountScales)]simd.Uint16s
	mouseRadii                                     [len(rockAmountScales)]simd.Uint16s
)

func init() {
	RocksPerPackedVector = simd.Uint16s{}.Len()
	zeroUint16 = simd.BroadcastUint16s(0)
	oneUint16 = simd.BroadcastUint16s(1)
	maximumSizeUint16 = simd.BroadcastUint16s(BitSpriteSlopeCodeCount - 1)
	coordinateMask16 = simd.BroadcastUint16s(packedCoordinateMask)
	nibbleMask16 = simd.BroadcastUint16s(0xF)
	maximumStepZ16 = simd.BroadcastUint16s(0xF)
	slopeCycleU16 = simd.BroadcastUint16s(atlasSlopeStates)
	zeroInt16 = simd.BroadcastInt16s(0)
	oneInt16 = simd.BroadcastInt16s(1)
	minimumSlopeI16 = simd.BroadcastInt16s(-7)
	maximumSlopeI16 = simd.BroadcastInt16s(7)
	slopeCycleI16 = simd.BroadcastInt16s(atlasSlopeStates)
	halfSlopeCycleI16 = simd.BroadcastInt16s(7)
	for value := range nibbleValues16 {
		nibbleValues16[value] = simd.BroadcastUint16s(uint16(value))
	}
	hoverMultipliers := [...]uint16{71, 57, 57, 57, 57, 3}
	hoverBiases := [...]uint16{389, 312, 312, 327, 355, 21}
	collisionMultipliers := [...]uint16{47, 75, 75, 19, 19, 1}
	collisionBiases := [...]uint16{270, 435, 435, 114, 130, 8}
	for amountScale := range rockAmountScales {
		hoverRadiusMultiplier[amountScale] = simd.BroadcastUint16s(hoverMultipliers[amountScale])
		hoverRadiusBias[amountScale] = simd.BroadcastUint16s(hoverBiases[amountScale])
		collisionRadiusMultiplier[amountScale] = simd.BroadcastUint16s(collisionMultipliers[amountScale])
		collisionRadiusBias[amountScale] = simd.BroadcastUint16s(collisionBiases[amountScale])
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
