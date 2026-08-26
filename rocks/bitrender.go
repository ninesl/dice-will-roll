package rocks

import "github.com/hajimehoshi/ebiten/v2"

// Draw renders every packed rock in slice order through the shared atlas.
func Draw(r *Rocks, screen *ebiten.Image) {
	if len(r.Positions) != len(r.Sprites) {
		panic("positions and sprites must have equal lengths")
	}

	drawOptions := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	for rockIndex, packedPosition := range r.Positions {
		positionX, positionY, _, _ := UnpackPosition(packedPosition)
		size, rotation, slopeX, slopeY := UnpackSprite(r.Sprites[rockIndex])
		frameIndex := ((slopeX+7)*bitSpriteSlopeStates+slopeY+7)*BitSpriteRotationFrames + int32(rotation)
		scale := r.Atlas.Scales[size]

		drawOptions.GeoM.Reset()
		drawOptions.GeoM.Scale(scale, scale)
		drawOptions.GeoM.Translate(float64(positionX), float64(positionY))
		screen.DrawImage(r.Atlas.Frames[frameIndex], drawOptions)
	}
}
