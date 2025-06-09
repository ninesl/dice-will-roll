package render

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	GAME_BOUNDS_X float64
	GAME_BOUNDS_Y float64
)

// A zone is where the cursor is
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

	right := die.Vec2.X+die.TileSize > z.MinWidth
	left := die.Vec2.X < z.MaxWidth

	xBounds := (right || left)

	top := die.Vec2.Y < z.MaxHeight
	bottom := die.Vec2.Y+die.TileSize > z.MinHeight

	if top || bottom && xBounds {
		return true
	}

	return false
}

type ZoneRenderable struct {
	sprite *ebiten.Image
	Zone
}

func (z *ZoneRenderable) Update() {
}

func (z *ZoneRenderable) Sprite() *ebiten.Image {
	return z.sprite
}

func (z *ZoneRenderable) Position() Vec2 {
	return Vec2{X: z.MinWidth, Y: z.MinHeight}
}

var (
	ROLLZONE      ZoneRenderable
	SCOREZONE     ZoneRenderable
	SmallRollZone ZoneRenderable
	BigRollZone   ZoneRenderable
)

func SetZones() {
	minWidth := GAME_BOUNDS_X / 5
	minHeight := GAME_BOUNDS_Y / 5

	BigRollZone = ZoneRenderable{
		Zone: Zone{
			MinWidth:  0,
			MaxWidth:  GAME_BOUNDS_X,
			MinHeight: 0, // minHeight,
			MaxHeight: GAME_BOUNDS_Y,
		},
		sprite: CreateImage(
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
			MaxHeight: GAME_BOUNDS_Y - minHeight,
		},
		sprite: CreateImage(
			int((GAME_BOUNDS_X-minWidth)-minWidth),
			int((GAME_BOUNDS_Y-minHeight)-minHeight),
			color.RGBA{R: 50, G: 50, B: 50, A: 128},
		),
	}

	ROLLZONE = BigRollZone
	// ROLLZONE = SmallRollZone

	SCOREZONE = ZoneRenderable{
		Zone: Zone{
			MinWidth:  0,
			MaxWidth:  GAME_BOUNDS_X,
			MinHeight: 0,
			MaxHeight: SmallRollZone.MinHeight,
		},
		sprite: CreateImage(
			int(GAME_BOUNDS_X),
			int(SmallRollZone.MinHeight),
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
