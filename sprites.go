package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type Vec2 struct {
	X, Y float64
}

// is a renderable
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

// // updates the active frame by 1, resets back to 0
// func (s *SpriteSheet) Animate() {
// 	s.ActiveFrame++
// 	if s.ActiveFrame > s.TileAmount {
// 		s.ActiveFrame = 0
// 	}
// }
