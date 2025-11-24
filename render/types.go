package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// assigned during main init()
var (
	GAME_BOUNDS_X float64
	GAME_BOUNDS_Y float64

	TileSize     float64
	HalfTileSize float64
)

// generally used to make a die move into a direction.
//
// see
type Direction uint8

// RockSpeedIndex is an index into the SpeedMap array
type RockSpeedIndex uint8

// DirectionSign represents movement direction as a bool (false = negative, true = positive)
// For X-axis: false = left (-1.0), true = right (+1.0)
// For Y-axis: false = up (-1.0), true = down (+1.0)
// Use !sign to flip direction on bounce
type DirectionSign bool

const (
	Negative DirectionSign = false // -1.0 multiplier
	Positive DirectionSign = true  // +1.0 multiplier
)

// Multiplier converts the DirectionSign to a float64 multiplier for velocity calculations
func (d DirectionSign) Multiplier() float64 {
	if d {
		return 1.0 // Positive direction
	}
	return -1.0 // Negative direction
}

const (
	UP Direction = iota
	DOWN
	LEFT
	RIGHT
	UPLEFT
	UPRIGHT
	DOWNRIGHT
	DOWNLEFT
)

var (
	// Usage: x to subtract when something is following cusor
	XOffset float64
	// Usage: y to subtract when something is following cusor
	YOffset float64

	// used to force a direction
	//
	// Usage: DirectionMap[Direction].X * math.Abs(renderable.Velocity.X)
	// DirectionMap = map[Direction]Vec2{
	DirectionArr = []Vec2{ // Direction indexes are aligned
		UP:        Vec2{X: 0, Y: -1},
		DOWN:      Vec2{X: 0, Y: 1},
		LEFT:      Vec2{X: -1, Y: 0},
		RIGHT:     Vec2{X: 1, Y: 0},
		UPLEFT:    Vec2{X: -1, Y: 1},
		UPRIGHT:   Vec2{X: 1, Y: -1},
		DOWNRIGHT: Vec2{X: 1, Y: 1},
		DOWNLEFT:  Vec2{X: -1, Y: 1},
	}

	// SpeedMap provides velocity multipliers for memory-efficient rock speed variation
	// Rocks use uint8 indices into this array instead of storing float64 speeds
	SpeedMap = []float64{
		2.0, // index 0: slowest
		3.0, // index 1: slow
		4.0, // index 2: moderate
		5.0, // index 3: fast
		6.0, // index 4: fastest
	}
)

type Vec2 struct {
	X, Y float64
}

// used for determining color values
type Vec3 struct {
	R, G, B float32
}

// makes it 0.0 - 1.0 for Kage
func normalize(v int) float32 {
	return float32(v) / 255.0
}

// give 0-255 for r g b values return normalized to kages 0.0 - 1.0
func KageColor(r, g, b int) Vec3 {
	return Vec3{
		R: normalize(r),
		G: normalize(g),
		B: normalize(b),
	}
}

// Returns a Vec3 representation of itself for a Kage uniform
func (v Vec3) KageVec3() []float32 {
	return []float32{v.R, v.G, v.B}
}

// Returns a Vec2 representation of itself for a Kage uniform
func (v Vec2) KageVec2() []float32 {
	return []float32{float32(v.X), float32(v.Y)}
}

// should be renderable
type Sprite struct {
	Image       *ebiten.Image
	SpriteSheet SpriteSheet
	// Vec2        Vec2
	// Updates the sprite in Game.Update() loop
}

// assumed all tiles are squares
type SpriteSheet struct {
	WidthTiles  int
	HeightTiles int
	TileSize    int
	TileAmount  int
	ActiveFrame int
}

func NewSpriteSheet(w, h, t int) SpriteSheet {
	return SpriteSheet{
		WidthTiles:  w,
		HeightTiles: h,
		TileSize:    t,
		TileAmount:  w * h,
		ActiveFrame: 0,
	}
}

// Gets the 'index' of the sheet
func (s *SpriteSheet) Rect(index int) image.Rectangle {
	x := (index % s.WidthTiles) * s.TileSize
	y := (index / s.WidthTiles) * s.TileSize

	return image.Rect(
		x, y, x+s.TileSize, y+s.TileSize,
	)
}
