package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

var (
	GAME_BOUNDS_X int
	GAME_BOUNDS_Y int
)

const TILE_SIZE int = 128

type Game struct {
	// DiceSprite *render.Sprite
	Shaders   map[shaders.ShaderKey]*ebiten.Shader
	Dice      []*Die
	Fixed     render.Vec2
	Time      time.Time
	startTime time.Time
	TileSize  float64
	//   can be updated with LocateCursor()
	x, y  float64 // the x/y coordinates of the cursor
	DEBUG DEBUG
}

// Mode is a representation different game states that modify
// controls, what is getting displayed, etc.
type Action uint16

const (
	NONE Action = iota

	ROLLING // the die is moving around, collision checks etc.
	DRAG    // locked to mouse cursor
	HELD    // held in hand, waiting to be scored. will move to it's Fixed

	SCORE  // when the score button is pressed
	ROLL   // when the spacebar is pressed
	PRESS  // when the mouse is pressed
	SELECT // when the mouse is released ie. clicked
)

func (a Action) String() string {
	str := "NONE"
	switch a {
	case ROLLING:
		str = "ROLLING"
	case DRAG:
		str = "DRAG"
	case HELD:
		str = "HELD"
	case SCORE:
		str = "SCORE"
	case ROLL:
		str = "ROLL"
	case PRESS:
		str = "PRESS"
	case SELECT:
		str = "SELECT"
	}
	return str
}

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
	// ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Dice Will Roll")

	game := LoadGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
