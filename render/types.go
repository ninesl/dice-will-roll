package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// assigned during main init()
var (
	GAME_BOUNDS_X float32
	GAME_BOUNDS_Y float32

	TileSize        float32
	HalfTileSize    float32
	DieTileSize     float32 // Die-specific tile size for rendering and collisions
	HalfDieTileSize float32 // Half of DieTileSize for die center calculations
)

// generally used to make a die move into a direction.
//
// see
type Direction uint8

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
	XOffset float32
	// Usage: y to subtract when something is following cusor
	YOffset float32

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
)

type Vec2 struct {
	X, Y float32
}

// used for determining color values
type Vec3 struct {
	X, Y, Z float32
}

// makes it 0.0 - 1.0 for Kage
func normalize(v int) float32 {
	return float32(v) / 255.0
}

// give 0-255 for r g b values return normalized to kages 0.0 - 1.0
func KageColor(r, g, b int) Vec3 {
	return Vec3{
		X: normalize(r),
		Y: normalize(g),
		Z: normalize(b),
	}
}

// Returns a Vec3 representation of itself for a Kage uniform
func (v Vec3) KageVec3() []float32 {
	return []float32{v.X, v.Y, v.Z}
}

// Returns a Vec2 representation of itself for a Kage uniform
func (v Vec2) KageVec2() []float32 {
	return []float32{float32(v.X), float32(v.Y)}
}

// should be renderable
type Sprite struct {
	Image       *ebiten.Image
	SpriteSheet SpriteSheet
	ActiveFrame int

	// Vec2        Vec2
	// Updates the sprite in Game.Update() loop
}

// assumed all tiles are squares
type SpriteSheet struct {
	WidthTiles  int
	HeightTiles int
	TileSize    int
	TileAmount  int
}

func NewSpriteSheet(w, h, t int) SpriteSheet {
	return SpriteSheet{
		WidthTiles:  w,
		HeightTiles: h,
		TileSize:    t,
		TileAmount:  w * h,
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
