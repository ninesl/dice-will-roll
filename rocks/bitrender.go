package rocks

import "github.com/hajimehoshi/ebiten/v2"

var drawOptions = &ebiten.DrawImageOptions{Filter: ebiten.FilterPixelated}

// Draw renders every packed rock in slice order through the shared atlas.
func Draw(rocks *Rocks, screen *ebiten.Image) {
	if len(rocks.Positions) != len(rocks.Sprites) {
		panic("positions and sprites must have equal lengths")
	}
	scales := rocks.Atlas.Scales[rocks.Atlas.AmountScale]
	halfDrawSizes := rocks.Atlas.HalfDrawSizes[rocks.Atlas.AmountScale]

	for i, pos := range rocks.Positions {
		positionX, positionY, _, _ := UnpackPosition(pos)
		size, rotation, _, _, _, slopeX, slopeY := UnpackSprite(rocks.Sprites[i])
		scale := scales[size]
		halfDrawSize := halfDrawSizes[size]

		drawOptions.GeoM.Reset()
		drawOptions.GeoM.Scale(scale, scale)
		drawOptions.GeoM.Translate(
			float64(positionX)-halfDrawSize,
			float64(positionY)-halfDrawSize,
		)
		screen.DrawImage(rocks.Atlas.Frames[filterIndex(slopeX, slopeY, rotation)], drawOptions)
	}
}
