package controls

import "github.com/ninesl/dice-will-roll/render"

type CursorInfo struct {
	LastPosition render.Vec2
	Position     render.Vec2
}

type MouseInfo struct {
	CursorInfo
	Clicked, Down, Released                bool
	RightClicked, RightDown, RightReleased bool
}
