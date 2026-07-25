package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/settings"
)

// A zone contains the bounds of an arbitrary defined area on the screen
//
// zones specific implementation is specific to what it is, see parts.txt
//
//	// Zones are:
//	ROLL
//	GEMS
//	SCORE
//
// TODO: zones make up Game Screens, ie LOOP MINE BASE etc.

// TODO:FIXME: refactor zone and zonerender into 1 struct. minwidth/max etc determined by image rect/Bounds?
type Zone struct {
	MinWidth  float32
	MaxWidth  float32
	MinHeight float32
	MaxHeight float32
	// Image     *ebiten.Image
}

func (z *ZoneRenderable) ContainsPoint(x, y float32) bool {
	return x >= z.MinWidth && x <= z.MaxWidth && y >= z.MinHeight && y <= z.MaxHeight
}

// is true if any part of the die is within the bounds of the zone
func (z *ZoneRenderable) ContainsDie(die *DieRenderable) bool {
	return die.Rect().Overlaps(z.Image.Bounds())
}

type ZoneRenderable struct {
	Image *ebiten.Image
	Zone
}

// func (z *ZoneRenderable) Update() {
// }

// func (z *ZoneRenderable) Sprite() *ebiten.Image {
// 	return z.image
// }

func (z *ZoneRenderable) Position() Vec2 {
	return Vec2{X: z.MinWidth, Y: z.MinHeight}
}

var (
	// bounds of where the dice will roll
	ROLLZONE ZoneRenderable
	// where a die is when HELD
	SCOREZONE ZoneRenderable
	// small box in the middle of the screen
	SmallRollZone ZoneRenderable
	// larger area
	BigRollZone ZoneRenderable
)

func SetZones() {
	minWidth := float32(settings.Screen.ResolutionX / 12)
	minHeight := float32(settings.Screen.ResolutionY / 7)

	BigRollZone = ZoneRenderable{
		Zone: Zone{
			MinWidth:  0,
			MaxWidth:  float32(settings.Screen.ResolutionX),
			MinHeight: 0, // minHeight,
			MaxHeight: float32(settings.Screen.ResolutionY),
		},
		Image: CreateImage(
			settings.Screen.ResolutionX,
			settings.Screen.ResolutionY,
			color.RGBA{R: 123, G: 123, B: 123, A: 128},
		),
	}

	SmallRollZone = ZoneRenderable{
		Zone: Zone{
			MinWidth:  minWidth,
			MaxWidth:  float32(settings.Screen.ResolutionX) - minWidth,
			MinHeight: minHeight,
			// MinHeight: 0,
			MaxHeight: float32(settings.Screen.ResolutionY) - minHeight,
		},
		Image: CreateImage(
			settings.Screen.ResolutionX-int(minWidth-minWidth),
			settings.Screen.ResolutionY-int(minHeight-minHeight),
			// int((GAME_BOUNDS_Y-minHeight)-minHeight),
			color.RGBA{R: 50, G: 50, B: 50, A: 128},
		),
	}

	// ROLLZONE = BigRollZone
	ROLLZONE = SmallRollZone

	SCOREZONE = ZoneRenderable{
		Zone: Zone{
			MinWidth:  0,
			MaxWidth:  float32(settings.Screen.ResolutionX),
			MinHeight: 0,
			MaxHeight: minHeight,
			// MaxHeight: SmallRollZone.MinHeight,
		},
		Image: CreateImage(
			int(settings.Screen.ResolutionX),
			int(minHeight),
			// int(SmallRollZone.MinHeight),
			color.RGBA{R: 100, G: 150, B: 80, A: 140},
		),
	}
}

// helperfunction for placeholder sprites
func CreateImage(width, height int, c color.Color) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(c)
	return img
}
