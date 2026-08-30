package controls

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ninesl/dice-will-roll/render"
)

type CursorInfo struct {
	LastPosition render.Vec2
	Position     render.Vec2
}

type MouseInfo struct {
	CursorInfo
	Active                                 bool
	Clicked, Down, Released                bool
	RightClicked, RightDown, RightReleased bool
}

func (m *MouseInfo) Update() {
	x, y := ebiten.CursorPosition()
	m.LastPosition = m.Position
	m.Position.X = float32(x)
	m.Position.Y = float32(y)
	m.Active = true

	m.Down = ebiten.IsMouseButtonPressed(ebiten.MouseButton0)
	m.Clicked = inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0)
	m.Released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0)
	m.RightDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	m.RightClicked = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	m.RightReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight)
}
