package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	// _ "embed"
	// _ "image/png" // for png encoder

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/dice"
	"github.com/ninesl/dice-will-roll/music"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/render/shaders"
	"github.com/ninesl/dice-will-roll/rocks"
	"github.com/ninesl/dice-will-roll/settings"
)

var (
	ClickTime = time.Millisecond * 250

	NUM_PLAYER_DICE int = 7
)

// Command-line flags
var (
	numRocks       = flag.Int("rocks", 10000, "Number of rocks to generate")
	cpuProfilePath = flag.String("cpuprofile", "", "Write a CPU profile to this file")
	memProfilePath = flag.String("memprofile", "", "Write a heap profile to this file on exit")
)

func init() {
	// render.GAME_BOUNDS_X = float32(GAME_BOUNDS_X)
	// render.GAME_BOUNDS_Y = float32(GAME_BOUNDS_Y)
	//
	// // render.TileSize = TileSize
	// // render.HalfTileSize = float32(TILE_SIZE / 2)
	// render.DieTileSize = TileSize                   // Die-specific tile size, same as base TileSize
	// render.HalfDieTileSize = float32(TILE_SIZE / 2) // Half of DieTileSize for die center calculations
	//
	// // Pre-compute die collision constants (used for rock-die collision detection)
	// render.EffectiveDieTileSize = render.DieTileSize * 0.75
	// render.DieTileInset = (render.DieTileSize - render.EffectiveDieTileSize) / 2
	// render.HalfEffectiveDie = render.EffectiveDieTileSize / 2
	//
	// FONT_SIZE = float64(GAME_BOUNDS_Y / 64)
	//
}

// TODO: last position...?
type CursorInfo struct {
	LastPosition render.Vec2
	Position     render.Vec2
}

type MouseInfo struct {
	CursorInfo
	Clicked, Down, Released                bool
	RightClicked, RightDown, RightReleased bool
}

type Game struct {
	Shaders map[shaders.ShaderKey]*ebiten.Shader
	UIState *PlayerUIState

	// RocksImage    *ebiten.Image
	RocksRenderer *rocks.RocksRenderer // New rocks rendering system,
	opts          *DrawOptions

	ActiveLevel *Level // keeping track of rocks
	Music       *music.NowPlaying

	// //TODO:FIXME: make a new one per level?, game renders the same but active level reassigns
	Dice               []*Die        // Player's dice
	diceCenterBuffer   []render.Vec3 // Pre-allocated die center buffer (X=centerX, Y=centerY, Z=360rotation axis)
	diceVelocityBuffer []render.Vec2 // Pre-allocated die velocity buffer (X=velocityX, Y=velocityY)
	heldDie            []*Die        // Reused scratch buffer of currently held dice.
	hold               []dice.Die    // Reused scratch buffer for hand ranking held dice.

	Mouse MouseInfo

	startTime      time.Time
	holdTime       time.Time
	activeDieIdx   int // active die index, g.ActiveDie() to get the *Die
	holdCx, holdCy float32
	// is updated with UpdateCursor() in update loop
	//cx, cy    float32 // the x/y coordinates of the cursor
	// holdCx, holdCy float32
	time float32 // tracks time for shaders and animations. updated in g.Update() every tick
}

func (g *Game) ActiveDie() *Die {
	return g.Dice[g.activeDieIdx]
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

func LoadGame() *Game {
	// dieImgSize := TILE_SIZE * 2
	render.SetZones()
	nowPlaying, err := loadGameMusic()
	if err != nil {
		log.Fatal(err)
	}
	nowPlaying.Play()

	playerDice := SetupPlayerDice()

	rockAmount := *numRocks
	// Initialize rocks renderer with hybrid real-time 3D SDF system
	rocksConfig := rocks.RocksConfig{
		TotalRocks: []int{rockAmount},
		BaseColors: []render.Vec3{
			render.Grey,
			// render.Brown,
			// render.RainbowColors[0],
			// render.RainbowColors[1],
			// render.RainbowColors[2],
			// render.RainbowColors[3],
			// render.RainbowColors[4],
			// render.RainbowColors[5],
			// render.RainbowColors[6],
		},
		RockTileSize: rocks.CalculateRockTileSize(settings.Screen.Tiles.TileSize32, rockAmount),
		// Dynamically scaled based on rock amount
		WorldBoundsX:          float32(settings.Screen.ResolutionX),
		WorldBoundsY:          float32(settings.Screen.ResolutionY),
		ColorTransitionFrames: 30, // 30 frames (~0.5 seconds at 60fps)
	}

	g := &Game{
		UIState:       NewUIState(),
		Dice:          playerDice,
		Shaders:       shaders.LoadShaders(),
		RocksRenderer: rocks.NewRocksRenderer(rocksConfig),
		Music:         nowPlaying,
		opts: &DrawOptions{
			image: &ebiten.DrawImageOptions{},
			text: &text.DrawOptions{
				LayoutOptions: text.LayoutOptions{LineSpacing: settings.Screen.LineSpacing},
			},
			shader: &ebiten.DrawRectShaderOptions{}},
		diceCenterBuffer:   make([]render.Vec3, 0, NUM_PLAYER_DICE),
		diceVelocityBuffer: make([]render.Vec2, 0, NUM_PLAYER_DICE),
		heldDie:            make([]*Die, 0),
		hold:               make([]dice.Die, 0),
		startTime:          time.Now(),
		ActiveLevel: NewLevel(LevelOptions{
			Rocks: rockAmount,
			Hands: 10,
			Rolls: 2,
		}),
	}

	// var rocksImage *ebiten.Image = ebiten.NewImage(g.Bounds())
	// g.RocksImage = rocksImage
	//
	// g.DEBUG.dieImgTransparent = render.CreateImage(dieImgSize, dieImgSize, color.RGBA{56, 56, 56, 100})

	return g
}

func loadGameMusic() (*music.NowPlaying, error) {
	trackFile, err := music.TracksFS.Open("tracks/json/track_iommiwatts.json")
	if err != nil {
		return nil, err
	}
	defer trackFile.Close()

	var track music.Track
	if err := json.NewDecoder(trackFile).Decode(&track); err != nil {
		return nil, err
	}

	audioFile, err := music.TracksFS.Open("tracks/files/iommiwatts.mp3")
	if err != nil {
		return nil, err
	}
	defer audioFile.Close()

	const sampleRate = 44100
	stream, err := mp3.DecodeWithSampleRate(sampleRate, audioFile)
	if err != nil {
		return nil, err
	}

	pcm, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	ctx := audio.NewContext(sampleRate)
	player := ctx.NewPlayerFromBytes(pcm)
	player.SetVolume(0.1)
	nowPlaying := music.NewNowPlaying(track, player)
	nowPlaying.DurationMS = int64(len(pcm)) * 1000 / int64(sampleRate*4)

	return nowPlaying, nil
}

func (g *Game) String() string {
	return fmt.Sprintf("(%d.%d)\nROLLZONE %#v\n",
		settings.Screen.ResolutionX,
		settings.Screen.ResolutionY, render.ROLLZONE)
}

// // interface impl
//
//	func (g *Game) Bounds() (int, int) {
//		return settings.Screen.ResolutionX, settings.Screen.ResolutionY
//	}
//
// return the pixels in the game
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return settings.Screen.ResolutionX, settings.Screen.ResolutionY
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	flag.Parse()
	settings.InitScreenSettings(ebiten.Monitor())
	rocks.Init(settings.Screen)
	ebiten.SetFullscreen(true)

	fmt.Printf("%#+v\n", settings.Screen)

	game := LoadGame()
	return ebiten.RunGame(game)
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
