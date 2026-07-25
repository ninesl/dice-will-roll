package settings

import "github.com/hajimehoshi/ebiten/v2"

var (
	Screen ScreenSettings
)

type ScreenSettings struct {
	Monitor                        *ebiten.MonitorType
	MaxResolutionX, MaxResolutionY int
	ResolutionX, ResolutionY       int
	Fullscreen                     bool
	FontSize                       float64 //= float64(ResolutionY / 64)
	Tiles                          TileSettings
}

type TileSettings struct {
	// derived fields (but precomputed for cache locality)
	TileSize int //= GAME_BOUNDS_Y / 9 //Base tile size, roughly the size of the Die

	TileSize32      float32
	HalfTileSize32  float32
	DieTileSize     float32
	HalfDieTileSize float32

	EffectiveDieTileSize,
	DieTileInset,
	HalfEffectiveDie float32
}

func InitScreenSettings(monitor *ebiten.MonitorType) {
	ebiten.SetWindowTitle("Dice Will Roll")
	//TODO:FIXME: this is how we determine the max perf for a given device.
	// ebiten.SetTPS(ebiten.SyncWithFPS)
	// ebiten.SetVsyncEnabled(false)

	bX, bY := monitor.Size()
	Screen = ScreenSettings{
		Monitor:        monitor,
		MaxResolutionX: bX, MaxResolutionY: bY,
		Fullscreen: true,
	}
	Screen.SetResolution(
		Screen.MaxResolutionX,
		Screen.MaxResolutionY)
}

func (ss *ScreenSettings) SetScale(scale int) {
	ss.Tiles.TileSize = ss.ResolutionY / scale
	ss.Tiles.TileSize32 = float32(ss.Tiles.TileSize)
	ss.Tiles.HalfTileSize32 = ss.Tiles.TileSize32 * 0.5

	ss.Tiles.DieTileSize = ss.Tiles.TileSize32
	ss.Tiles.HalfDieTileSize = ss.Tiles.HalfTileSize32

	ss.Tiles.DieTileSize = ss.Tiles.TileSize32
	ss.FontSize = float64(ss.ResolutionY) / float64(scale*8.0)

	// Pre-compute die collision constants (used for rock-die collision detection)
	ss.Tiles.EffectiveDieTileSize = ss.Tiles.DieTileSize * 0.75
	ss.Tiles.HalfEffectiveDie = ss.Tiles.EffectiveDieTileSize * 0.5
	ss.Tiles.DieTileInset = (ss.Tiles.DieTileSize - ss.Tiles.EffectiveDieTileSize) * 0.5

	// render.EffectiveDieTileSize = render.DieTileSize * 0.75
	// render.DieTileInset = (render.DieTileSize - render.EffectiveDieTileSize) / 2
	// render.HalfEffectiveDie = render.EffectiveDieTileSize / 2
	//
}

// could do some sort of downscaling in the future..? skip the bounds check
func (ss *ScreenSettings) SetResolution(x, y int) {
	ss.ResolutionX = min(x, ss.MaxResolutionX)
	ss.ResolutionY = min(y, ss.MaxResolutionY)

	ss.SetScale(9)

	ebiten.SetWindowSize(ss.ResolutionX, ss.ResolutionY)
	ebiten.SetFullscreen(ss.Fullscreen)

}
