package main

import (
	"github.com/ninesl/dice-will-roll/render"
)

// type Button struct {
// 	XY render.Vec2
// 	// sprite or shader
// 	SpriteIndex int
// 	SpriteSheet render.SpriteSheet
// }

// An area is a definition one rectangle/square on the screen
type Area struct {
	XY   render.Vec2 // top left
	Size render.Vec2 // X is width, Y is Height
}
