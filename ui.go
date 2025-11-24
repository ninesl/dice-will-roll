package main

import "github.com/ninesl/dice-will-roll/render"

// type Button struct {
// 	XY render.Vec2
// 	// sprite or shader
// 	SpriteIndex int
// 	SpriteSheet render.SpriteSheet
// }

// SEPERATE SYSTEM ENTIRELY FOR BUTTON SPECIFIC ELEMENTS. CLICKING/ACTIONS ARE FOR DICE!/the actual 'game' vs the rest of the loop.

type ScreenItem interface {
	XY() *render.Vec2
	DimensionSize() render.Vec2
}

type AnimatedItem interface {
	ScreenItem
	// XY() *render.Vec2
	// DimensionSize() render.Vec2
	UpdateFrame()
}

type GenericButton struct {
	XY   *render.Vec2
	Size render.Vec2
}

// only check each Button if the mouse is okay. Look at RENDERZONES, etc
func (g *Game) cursorWithinBounds(b ScreenItem) bool {
	return g.cx < +b.XY().X+b.DimensionSize().X && g.cx > b.XY().X &&
		g.cy < b.XY().Y+b.DimensionSize().Y && g.cy > b.XY().Y
}
