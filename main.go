package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	// _ "embed"
	// _ "image/png" // for png encoder

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// TODO: embedded FS for loading assets
//
// //go:embed assets/images/dice.png
// var dicePng []byte

// func LoadImage() image.Image {
// 	img, _, err := image.Decode(bytes.NewReader(dicePng))
// 	if err != nil {
// 		log.Fatalf("failed to decode embedded image: %v", err)
// 	}
// 	return img
// }

var (
	GAME_BOUNDS_X int = TILE_SIZE * 16
	GAME_BOUNDS_Y int = TILE_SIZE * 9
	ResolutionX   int = 1600
	ResolutionY   int = 900
)

const (
	TILE_SIZE int = 64

	// tile size is always the width and height of the die image
	TileSize float64 = float64(TILE_SIZE)
)

func init() {
	render.GAME_BOUNDS_X = float64(GAME_BOUNDS_X)
	render.GAME_BOUNDS_Y = float64(GAME_BOUNDS_Y)

	render.TileSize = TileSize
	render.HalfTileSize = float64(TILE_SIZE / 2)
}

type Game struct {
	Shaders   map[shaders.ShaderKey]*ebiten.Shader
	Dice      []*Die        // Player's dice
	Hand      dice.HandRank // current hand rank of all held dice
	Time      time.Time
	startTime time.Time
	// is updated with UpdateCursor() in update loop
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

var (
	DEBUG_FONT *text.GoTextFaceSource
)

func SetFonts() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.ArcadeN_ttf))
	if err != nil {
		log.Fatal(err)
	}
	DEBUG_FONT = s
}

func LoadGame() *Game {
	SetFonts()

	dieImgSize := TILE_SIZE * 2
	render.SetZones()

	dice := SetupPlayerDice(dieImgSize)

	g := &Game{
		Dice:      dice,
		Shaders:   shaders.LoadShaders(),
		startTime: time.Now(),
	}

	// g.DEBUG.dieImgTransparent = render.CreateImage(dieImgSize, dieImgSize, color.RGBA{56, 56, 56, 100})

	return g
}

func (g *Game) String() string {
	return fmt.Sprintf("GAMEBOUNDS X %d\nGAMEBOUNDS Y %ds\nROLLZONE %#v\n", GAME_BOUNDS_X, GAME_BOUNDS_Y, render.ROLLZONE)
}

// interface impl
func (g *Game) Bounds() (int, int) {
	return GAME_BOUNDS_X, GAME_BOUNDS_Y
}

// return the pixels in the game
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return GAME_BOUNDS_X, GAME_BOUNDS_Y
}

func main() {
	ebiten.SetWindowSize(ResolutionX, ResolutionY)
	ebiten.SetWindowTitle("Dice Will Roll")

	game := LoadGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

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
