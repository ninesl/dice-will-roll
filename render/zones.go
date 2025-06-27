package render

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
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
type Zone struct {
	MinWidth  float64
	MaxWidth  float64
	MinHeight float64
	MaxHeight float64
}

// helper function
func (z *Zone) ContainsDie(die *DieRenderable) bool {
	fmt.Sprintln("%+v\n%+v", z, die.Vec2)

	right := die.Vec2.X+TileSize > z.MinWidth
	left := die.Vec2.X < z.MaxWidth

	xBounds := (right || left)

	top := die.Vec2.Y < z.MaxHeight
	bottom := die.Vec2.Y+TileSize > z.MinHeight

	if top || bottom && xBounds {
		return true
	}

	return false
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
	minWidth := GAME_BOUNDS_X / 12
	minHeight := GAME_BOUNDS_Y / 7

	BigRollZone = ZoneRenderable{
		Zone: Zone{
			MinWidth:  0,
			MaxWidth:  GAME_BOUNDS_X,
			MinHeight: 0, // minHeight,
			MaxHeight: GAME_BOUNDS_Y,
		},
		Image: CreateImage(
			int((GAME_BOUNDS_X)),
			int((GAME_BOUNDS_Y /* - minHeight*/)),
			color.RGBA{R: 123, G: 123, B: 123, A: 128},
		),
	}

	SmallRollZone = ZoneRenderable{
		Zone: Zone{
			MinWidth:  minWidth,
			MaxWidth:  GAME_BOUNDS_X - minWidth,
			MinHeight: minHeight,
			// MinHeight: 0,
			MaxHeight: GAME_BOUNDS_Y - minHeight,
		},
		Image: CreateImage(
			int(GAME_BOUNDS_X-minWidth-minWidth),
			int(GAME_BOUNDS_Y-minHeight-minHeight),
			// int((GAME_BOUNDS_Y-minHeight)-minHeight),
			color.RGBA{R: 50, G: 50, B: 50, A: 128},
		),
	}

	// ROLLZONE = BigRollZone
	ROLLZONE = SmallRollZone

	SCOREZONE = ZoneRenderable{
		Zone: Zone{
			MinWidth:  0,
			MaxWidth:  GAME_BOUNDS_X,
			MinHeight: 0,
			MaxHeight: GAME_BOUNDS_Y / 5,
			// MaxHeight: SmallRollZone.MinHeight,
		},
		Image: CreateImage(
			int(GAME_BOUNDS_X),
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
