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
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
)

// will need a way to update these in settings
var (
	GAME_BOUNDS_X, GAME_BOUNDS_Y int     = ebiten.Monitor().Size()
	ResolutionX                  int     = GAME_BOUNDS_X // placeholder, options later
	ResolutionY                  int     = GAME_BOUNDS_Y
	TILE_SIZE                    int     = GAME_BOUNDS_Y / 9
	FONT_SIZE                    float64 = float64(ResolutionY / 64)
	// tile size is always the width and height of the die image
	TileSize float64 = float64(TILE_SIZE)

	NUM_PLAYER_DICE int = 7
)

func init() {
	render.GAME_BOUNDS_X = float64(GAME_BOUNDS_X)
	render.GAME_BOUNDS_Y = float64(GAME_BOUNDS_Y)

	render.TileSize = TileSize
	render.HalfTileSize = float64(TILE_SIZE / 2)

	FONT_SIZE = float64(GAME_BOUNDS_Y / 64)

	ebiten.SetFullscreen(true)

}

type Game struct {
	Shaders       map[shaders.ShaderKey]*ebiten.Shader
	RocksImage    *ebiten.Image
	RocksRenderer *render.RocksRenderer // New rocks rendering system
	Dice          []*Die                // Player's dice
	Time          time.Time
	startTime     time.Time
	ActiveLevel   *Level // keeping track of rocks
	// is updated with UpdateCursor() in update loop
	cx, cy float64 // the x/y coordinates of the cursor
	time   float32 // tracks time for shaders. updated in g.Update()
}

//TODO: make mode and action different types

// Action is the underlying type for
// die.Mode that represents different game states that modify
// controls, what is getting displayed, etc.
type Action uint16

const (
	NONE Action = iota

	ROLLING // the die is moving around, collision checks etc.
	DRAG    // locked to mouse cursor
	HELD    // held in hand, waiting to be scored. will move to it's Fixed
	SCORING // actively scoring

	ROLL   // when the spacebar is pressed
	PRESS  // when the mouse is pressed
	SELECT // when the mouse is released ie. clicked
	SCORE  // when the score button is pressed
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

	// dieImgSize := TILE_SIZE * 2
	render.SetZones()

	dice := SetupPlayerDice()

	// Initialize rocks renderer with 1M rocks
	rocksConfig := render.RocksConfig{
		TotalRocks:    1000000, // 1 million rocks
		SpriteSize:    64,      // 64x64 pixel sprites
		NumVariants:   20,      // 20 different rock shapes
		MaxVisible:    10000,   // Max 10k visible at once
		WorldSize:     2000.0,  // 2000 unit world
		MinRockSize:   0.5,
		MaxRockSize:   2.0,
		MovementSpeed: 50.0,
	}

	g := &Game{
		Dice:          dice,
		Shaders:       shaders.LoadShaders(),
		RocksRenderer: render.NewRocksRenderer(rocksConfig),
		startTime:     time.Now(),
		ActiveLevel: NewLevel(LevelOptions{
			Rocks: 100,
			Hands: 3,
			Rolls: 2,
		}),
	}

	var rocksImage *ebiten.Image = ebiten.NewImage(g.Bounds())
	g.RocksImage = rocksImage

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
