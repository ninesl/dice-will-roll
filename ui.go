package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/music"
	"github.com/ninesl/dice-will-roll/render"
	"github.com/ninesl/dice-will-roll/settings"
)

var (
	DEBUG_FONT     *text.GoTextFaceSource
	DEBUG_FONTFACE *text.GoTextFace
)

func SetFonts() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.ArcadeN_ttf))
	if err != nil {
		log.Fatal(err)
	}
	DEBUG_FONT = s
	DEBUG_FONTFACE = &text.GoTextFace{
		Source: DEBUG_FONT,
		Size:   settings.Screen.FontSize,
	}
}

type SceneID int

type PlayerUIState struct {
	Scenes         [SceneNum]*Scene
	ActiveScreenID SceneID // the index of the Screen that is currently being used
}

type textElement struct {
	xy   render.Vec2
	size render.Vec2
	msg  func(...any) string
}

type DEBUGViewMode int

const (
	DEBUGPLAYView DEBUGViewMode = iota
)

func DEBUGView(screen *ebiten.Image, g *Game, textOpts *text.DrawOptions, viewMode DEBUGViewMode) {
	debugLines := []string{
		g.ActiveLevel.String(),
		fmt.Sprintf("%.2f fps / %.2f tps", ebiten.ActualFPS(), ebiten.ActualTPS()),
	}
	if musicMessage := DEBUGMusic(g.Music); musicMessage != "" {
		debugLines = append(debugLines, musicMessage)
	}
	debugLines = append(debugLines, "<space> to ROLL, <q> to SCORE")
	DEBUGDrawMessage(screen, textOpts, strings.Join(debugLines, "\n"), 0)
	DEBUGDiceValues(screen, textOpts, g.Dice)

}

func DEBUGMusic(musicState *music.NowPlaying) string {
	if musicState == nil {
		return ""
	}

	upcoming := [10]int64{}
	for lane := range upcoming {
		upcoming[lane] = musicState.UpcomingMS(music.HookLane(lane))
	}

	return fmt.Sprintf("music ms=%d upcoming=%#v", musicState.MS(), upcoming)
}

func DEBUGDrawMessage(screen *ebiten.Image, textOpts *text.DrawOptions, msg string, y float64) {
	textOpts.GeoM.Translate(0, float64(y))
	textOpts.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, DEBUG_FONTFACE, textOpts)
	textOpts.GeoM.Reset()
	textOpts.ColorScale.Reset()
}

func DEBUGInitTextElements() []textElement {
	var debugTxts = make([]textElement, 0)

	return debugTxts
}

func DEBUGValuesFromDice(dice []*Die) []int {
	var track []int
	for i := range dice {
		track = append(track, dice[i].ActiveFace().NumPips())
	}
	return track
}

func DEBUGDiceValues(screen *ebiten.Image, textOpts *text.DrawOptions, dice []*Die) {
	var (
		Rolling []*Die
		Held    []*Die
		Scoring []*Die
	)
	for i := range dice {
		d := dice[i]
		switch d.Mode {
		case ROLLING:
			Rolling = append(Rolling, d)
		case HELD:
			Held = append(Held, d)
		case SCORING:
			Scoring = append(Scoring, d)
		}
	}
	y := float64(settings.Screen.ResolutionY) - settings.Screen.FontSize - settings.Screen.LineSpacing*2
	DEBUGDrawMessage(screen, textOpts, fmt.Sprintf("%5s%v\n%5s%v\n%5s%v",
		"roll", DEBUGValuesFromDice(Rolling),
		"held", DEBUGValuesFromDice(Held),
		"score", DEBUGValuesFromDice(Scoring)), y)
}

var (
	FPStext = textElement{}
)

//	func (t textElement) Draw(*ebiten.Image, *DrawOptions) {
//		textOpts.GeoM.Translate(0, float64(y))
//		textOpts.ColorScale.ScaleWithColor(color.White)
//		text.Draw(screen, msg, &text.GoTextFace{
//			Source: DEBUG_FONT,
//			Size:   settings.Screen.FontSize,
//		}, textOpts)
//		textOpts.GeoM.Reset()
//		textOpts.ColorScale.Reset()
//	}
func (t textElement) DrawActive(*ebiten.Image, *DrawOptions) {}
func (t textElement) DrawHot(*ebiten.Image, *DrawOptions)    {}
func (t textElement) XY() render.Vec2                        { return t.xy }   // top left
func (t textElement) Size() render.Vec2                      { return t.size } // X is width, Y is Height

type Element interface {
	Draw(*ebiten.Image, *DrawOptions)
	DrawActive(*ebiten.Image, *DrawOptions)
	DrawHot(*ebiten.Image, *DrawOptions)
	XY() render.Vec2   // top left
	Size() render.Vec2 // X is width, Y is Height
	Hot() bool         // should be used sparingly..?
}

type Scene struct {
	Elements        []Element
	HotElementID    int // the one currently being HOVERED, 0 nothing is hovered
	ActiveElementID int
}

func (pui *PlayerUIState) StartScene(sceneID SceneID) {
	pui.ActiveScreenID = sceneID
	pui.Scenes[pui.ActiveScreenID].ActiveElementID = 0
	pui.Scenes[pui.ActiveScreenID].HotElementID = 0
}

func (g *Game) DrawUI(screen *ebiten.Image, drawOptions *DrawOptions) {
	pui := g.UIState
	activeScene := pui.Scenes[pui.ActiveScreenID]
	for id, e := range activeScene.Elements[1:] {
		id++ // shifting up bc 0 is considered nothing
		if activeScene.HotElementID == id {
			if activeScene.ActiveElementID == id {
				// clicking/clicked?
				e.DrawActive(screen, drawOptions)
			} else {
				e.DrawHot(screen, drawOptions)
				// hovering
			}
		} else {
			e.Draw(screen, drawOptions)
			// normal draw
			// different draw states (animating, changing after being clicked)
			// etc based on e itself
		}
	}
}

func (scene *Scene) DrawElement(screen *ebiten.Image, elementID int) {
}

const (
	DEBUGScene SceneID = iota
	PLAYScene
	SHELFScene
	TOWNScene
	SHOPScene
	SceneNum
)

func DebugScene() *Scene {
	return &Scene{}
}

var (
	AllScenes = [SceneNum]*Scene{
		DebugScene(), // DEBUGScene
		{},           // PLAYScene
		{},           // SHELFScene
		{},           // TOWNScene
		{},           // SHOPScene
	}
)

func NewUIState() *PlayerUIState {
	SetFonts()
	return &PlayerUIState{
		ActiveScreenID: DEBUGScene,
		Scenes:         AllScenes,
	}
}

func pointOnElement(e Element, pt render.Vec2) bool {
	return pt.X > e.XY().X && pt.X < e.XY().X+e.Size().X && pt.Y > e.XY().Y && pt.Y < e.XY().Y+e.Size().Y
}

func (s *Scene) Update() {}

func (g *Game) updateHotElement(cursor render.Vec2) {
	s := g.UIState.Scenes[g.UIState.ActiveScreenID]
	for eID, e := range s.Elements[1:] {
		eID++ // 	if pointOnElement(e, cursor) {
		if g.Mouse.Down {
			s.HotElementID = eID
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
				s.ActiveElementID = eID
				// do elementAction
			}
			if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0) {
				s.ActiveElementID = 0
				// do elementEndAction
			}
		}
		_ = e
	}
}
