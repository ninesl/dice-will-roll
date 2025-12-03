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
	TILE_SIZE                    int     = GAME_BOUNDS_Y / 9 //Base tile size, roughly the size of the Die
	FONT_SIZE                    float64 = float64(ResolutionY / 64)
	// tile size is always the width and height of the die image
	TileSize float32 = float32(TILE_SIZE)

	ClickTime = time.Millisecond * 250

	NUM_PLAYER_DICE int = 7
)

func init() {
	render.GAME_BOUNDS_X = float32(GAME_BOUNDS_X)
	render.GAME_BOUNDS_Y = float32(GAME_BOUNDS_Y)

	// render.TileSize = TileSize
	// render.HalfTileSize = float32(TILE_SIZE / 2)
	render.DieTileSize = TileSize                   // Die-specific tile size, same as base TileSize
	render.HalfDieTileSize = float32(TILE_SIZE / 2) // Half of DieTileSize for die center calculations

	// Pre-compute die collision constants (used for rock-die collision detection)
	render.EffectiveDieTileSize = render.DieTileSize * 0.75
	render.DieTileInset = (render.DieTileSize - render.EffectiveDieTileSize) / 2
	render.HalfEffectiveDie = render.EffectiveDieTileSize / 2

	FONT_SIZE = float64(GAME_BOUNDS_Y / 64)

	ebiten.SetFullscreen(true)

}

type Game struct {
	Shaders        map[shaders.ShaderKey]*ebiten.Shader
	RocksImage     *ebiten.Image
	RocksRenderer  *render.RocksRenderer // New rocks rendering system, //TODO:FIXME: make a new one per level?, game renders the same but active level reassigns
	Dice           []*Die                // Player's dice
	diceDataBuffer []render.Vec3         // Pre-allocated die data buffer (X=centerX, Y=centerY, Z=speed)
	startTime      time.Time
	holdTime       time.Time
	holdCx, holdCy float32
	ActiveLevel    *Level // keeping track of rocks
	// is updated with UpdateCursor() in update loop
	cx, cy float32 // the x/y coordinates of the cursor
	// holdCx, holdCy float32
	time float32 // tracks time for shaders. updated in g.Update()

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

	rockAmount := 1000

	// Initialize rocks renderer with hybrid real-time 3D SDF system
	rocksConfig := render.RocksConfig{
		TotalRocks: rockAmount,
		BaseColors: []render.Vec3{
			render.Grey, render.Brown,
			// render.RainbowColors[0],
			// render.RainbowColors[1],
			// render.RainbowColors[2],
			// render.RainbowColors[3],
			// render.RainbowColors[4],
			// render.RainbowColors[5],
			// render.RainbowColors[6],
		},
		RockTileSize: render.CalculateRockTileSize(TileSize, rockAmount), // Dynamically scaled based on rock amount
		WorldBoundsX: float32(render.GAME_BOUNDS_X),
		WorldBoundsY: float32(render.GAME_BOUNDS_Y),
	}

	g := &Game{
		Dice:           dice,
		Shaders:        shaders.LoadShaders(),
		RocksRenderer:  render.NewRocksRenderer(rocksConfig),
		diceDataBuffer: make([]render.Vec3, 0, NUM_PLAYER_DICE),
		startTime:      time.Now(),
		ActiveLevel: NewLevel(LevelOptions{
			Rocks: rockAmount,
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

	//TODO:FIXME: this is how we determine the max perf for a given device.
	// ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetVsyncEnabled(false)

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

func (g *Game) ResetHoldPoint() {
	g.holdTime = time.Time{}
	g.holdCx = 0
	g.holdCy = 0
}
