package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
)

var (
	GAME_BOUNDS_X int
	GAME_BOUNDS_Y int
)

// Mode is a representation different game states that modify
// controls, what is getting displayed, etc.
type Action uint16

const (
	ROLLING Action = iota
	ROLL
	SCORE
	HELD
	DRAG   // locked to mouse cursor
	PRESS  // when the mouse is pressed
	SELECT // when the mouse is released ie. clicked
)

func (g *Game) UpdateCusor() {
	x, y := ebiten.CursorPosition()
	g.x = float64(x)
	g.y = float64(y)
}
func (g *Game) cursorWithin(zone render.ZoneRenderable) bool {
	return g.x > zone.MinWidth && g.x < zone.MaxWidth && g.y > zone.MinHeight && g.y < zone.MaxHeight
}

// interface impl
func (g *Game) Bounds() (int, int) {
	return int(g.TileSize) * 16, int(g.TileSize) * 9
}

func main() {
	ebiten.SetWindowSize(1600, 900) // resolution
	ebiten.SetWindowTitle("Dice Will Roll")

	game := LoadGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
