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

// Lerp linearly interpolates between two Vec3 colors
func (v Vec3) Lerp(target Vec3, t float32) Vec3 {
	return Vec3{
		X: v.X + (target.X-v.X)*t,
		Y: v.Y + (target.Y-v.Y)*t,
		Z: v.Z + (target.Z-v.Z)*t,
	}
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

// ImagePool manages a pool of reusable temporary images for rendering
// Images are lazily allocated and automatically cleared when retrieved
type ImagePool struct {
	images [](*ebiten.Image)
	index  int
	width  int
	height int
}

// NewImagePool creates a new empty image pool with the specified dimensions
func NewImagePool(width, height int) *ImagePool {
	return &ImagePool{
		images: make([]*ebiten.Image, 0),
		index:  0,
		width:  width,
		height: height,
	}
}

// GetNext returns the next available image from the pool
// Lazily allocates a new image if the pool is exhausted
// The returned image is automatically cleared and ready to use
func (p *ImagePool) GetNext() *ebiten.Image {
	// Grow pool if needed
	if p.index >= len(p.images) {
		img := ebiten.NewImage(p.width, p.height)
		p.images = append(p.images, img)
	}

	// Get image and increment counter
	img := p.images[p.index]
	p.index++

	// Clear to ensure clean state
	img.Clear()

	return img
}

// Reset resets the pool index to 0 for the next frame
// Call this at the start of each frame before using the pool
func (p *ImagePool) Reset() {
	p.index = 0
}

// Clear clears all images in the pool and resets the index to 0
// This is more thorough than Reset() - use when you need to ensure
// all images are in a clean state (e.g., after major state changes)
func (p *ImagePool) Clear() {
	for _, img := range p.images {
		img.Clear()
	}
	p.index = 0
}

// Len returns the current size of the pool (number of allocated images)
func (p *ImagePool) Len() int {
	return len(p.images)
}
