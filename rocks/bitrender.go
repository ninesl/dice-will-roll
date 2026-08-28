package rocks

import "github.com/hajimehoshi/ebiten/v2"

func FilterName(filter ebiten.Filter) string {
	switch filter {
	case ebiten.FilterNearest:
		return "Nearest"
	case ebiten.FilterLinear:
		return "Linear"
	case ebiten.FilterPixelated:
		return "Pixelated"
	default:
		return "Unknown"
	}
}

// Draw renders every packed rock in slice order using atlas
func Draw(rocks *Rocks, screen *ebiten.Image) {
	drawOptions := rocks.DrawOptions
	scales := rocks.Atlas.Scales[rocks.AmountScale]
	halfDrawSizes := rocks.Atlas.HalfDrawSizes[rocks.AmountScale]

	for i, pos := range rocks.Positions {
		positionX, positionY, _, _ := UnpackPosition(pos)
		sizeScore, slopeZ, _, _, _, _, _, _, slopeX, slopeY := UnpackSprite(rocks.Sprites[i])
		scale := scales[sizeScore]
		halfDrawSize := halfDrawSizes[sizeScore]

		drawOptions.GeoM.Reset()
		drawOptions.GeoM.Scale(scale, scale)
		drawOptions.GeoM.Translate(
			float64(positionX)-halfDrawSize,
			float64(positionY)-halfDrawSize,
		)
		screen.DrawImage(rocks.Atlas.Frames[filterIndex(slopeX, slopeY, slopeZ)], drawOptions)
	}
}
