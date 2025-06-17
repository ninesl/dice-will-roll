package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type Direction uint8

const (
	UP = iota
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
	// Usage: DirectionMap[Direction].X * math.Abs(renderable.Velocity.X)
	DirectionMap = map[Direction]Vec2{
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
	X, Y float64
}

// used for determining color values
type Vec3 struct {
	R, G, B float32
}

func normalize(v int) float32 {
	return float32(v) / 255.0
}

// give 0-255 for r g b values return normalized to kages 0.0 - 1.0
func Color(r, g, b int) Vec3 {
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
