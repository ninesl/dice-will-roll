package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ninesl/dice-will-roll/music"
	"golang.org/x/image/font/basicfont"
)

const sampleRate = 44100

var loggedNumberKeys = []ebiten.Key{
	ebiten.Key0,
	ebiten.Key1,
	ebiten.Key2,
	ebiten.Key3,
	ebiten.Key4,
	ebiten.Key5,
	ebiten.Key6,
	ebiten.Key7,
	ebiten.Key8,
	ebiten.Key9,
}

var capturedHookKeyIndexes = map[ebiten.Key]int{
	ebiten.Key0: 0,
	ebiten.Key1: 1,
	ebiten.Key2: 2,
	ebiten.Key3: 3,
	ebiten.Key4: 4,
	ebiten.Key5: 5,
	ebiten.Key6: 6,
	ebiten.Key7: 7,
	ebiten.Key8: 8,
	ebiten.Key9: 9,
}

var labelFace = text.NewGoXFace(basicfont.Face7x13)

var laneLabelColors = []color.Color{
	color.RGBA{R: 255, G: 255, B: 255, A: 255},
	color.RGBA{R: 255, G: 220, B: 80, A: 255},
	color.RGBA{R: 80, G: 220, B: 255, A: 255},
	color.RGBA{R: 255, G: 120, B: 120, A: 255},
	color.RGBA{R: 120, G: 255, B: 160, A: 255},
	color.RGBA{R: 255, G: 160, B: 255, A: 255},
	color.RGBA{R: 160, G: 180, B: 255, A: 255},
	color.RGBA{R: 255, G: 180, B: 100, A: 255},
	color.RGBA{R: 180, G: 255, B: 240, A: 255},
	color.RGBA{R: 220, G: 180, B: 255, A: 255},
}

type appState struct {
	audioPath string
	newFile   bool
	deleted   bool

	audioContext *audio.Context
	player       *audio.Player
	nowPlaying   *music.NowPlaying
	track        music.Track
	captured     map[int][]int64
	newCaptured  map[int][]int64

	durationMS int64
	CurrentMS  int64
	playing    bool

	waveformImage *ebiten.Image
	waveformRect  image.Rectangle

	msInputValue string
	errText      string
}

type root struct {
	guigui.DefaultWidget

	state appState

	background basicwidget.Background
	fileText   basicwidget.Text
	statusText basicwidget.Text
	clockText  basicwidget.Text
	msText     basicwidget.Text
	errText    basicwidget.Text

	restartButton basicwidget.Button
	msDownButton  basicwidget.Button
	msInput       basicwidget.TextInput
	msUpButton    basicwidget.Button
	callbacksSet  bool

	waveform waveformWidget

	layoutItems []guigui.LinearLayoutItem
	buttonItems []guigui.LinearLayoutItem
	msItems     []guigui.LinearLayoutItem
}

type waveformWidget struct {
	guigui.DefaultWidget

	state *appState
}

func (r *root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.fileText)
	adder.AddWidget(&r.statusText)
	adder.AddWidget(&r.clockText)
	adder.AddWidget(&r.msText)
	adder.AddWidget(&r.waveform)
	adder.AddWidget(&r.restartButton)
	adder.AddWidget(&r.msDownButton)
	adder.AddWidget(&r.msInput)
	adder.AddWidget(&r.msUpButton)
	if r.state.errText != "" {
		adder.AddWidget(&r.errText)
	}

	r.waveform.state = &r.state
	r.fileText.SetValue("File: " + r.state.audioPath)
	if r.state.playing {
		r.statusText.SetValue("(Now Playing)")
	} else {
		r.statusText.SetValue("(Paused)")
	}
	r.clockText.SetValue("Time: " + formatClock(r.state.CurrentMS))
	r.msText.SetValue(fmt.Sprintf("TotalMS: %d / %d", r.state.CurrentMS, r.state.durationMS))
	r.errText.SetValue(r.state.errText)
	r.errText.SetColor(color.RGBA{R: 255, A: 255})

	r.initCallbacks()
	r.restartButton.SetText("Restart")
	if r.state.playing {
		r.msDownButton.SetText("-5ms")
		r.msUpButton.SetText("+1ms")
		r.msDownButton.SetSemanticColor(basicwidgetdraw.SemanticColorBase)
		r.msUpButton.SetSemanticColor(basicwidgetdraw.SemanticColorBase)
	} else {
		r.msDownButton.SetText("-5ms")
		r.msUpButton.SetText("+1ms")
		r.msDownButton.SetSemanticColor(basicwidgetdraw.SemanticColorAccent)
		r.msUpButton.SetSemanticColor(basicwidgetdraw.SemanticColorAccent)
	}
	r.msInput.SetValue(r.state.msInputValue)
	r.msInput.SetPlaceholder("TotalMS")
	r.msInput.SetHorizontalAlign(basicwidget.HorizontalAlignRight)

	return nil
}

func (r *root) initCallbacks() {
	if r.callbacksSet {
		return
	}
	r.callbacksSet = true

	r.restartButton.OnUp(func(context *guigui.Context) {
		r.restart()
		guigui.RequestRebuild(r)
	})
	r.msDownButton.OnUp(func(context *guigui.Context) {
		if r.state.playing {
			return
		}
		r.seekAndPause(r.state.CurrentMS - 5)
		guigui.RequestRebuild(r)
	})
	r.msUpButton.OnUp(func(context *guigui.Context) {
		if r.state.playing {
			return
		}
		r.seekAndPause(r.state.CurrentMS + 1)
		guigui.RequestRebuild(r)
	})
	r.msInput.OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		r.state.msInputValue = text
		if !committed {
			return
		}
		ms, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			r.state.errText = "invalid TotalMS"
			guigui.RequestRebuild(r)
			return
		}
		r.state.errText = ""
		r.seekAndPause(ms)
		guigui.RequestRebuild(r)
	})
}

func (r *root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, bounds)
	u := basicwidget.UnitSize(context)

	r.buttonItems = slices.Delete(r.buttonItems, 0, len(r.buttonItems))
	r.buttonItems = append(r.buttonItems,
		guigui.LinearLayoutItem{Widget: &r.restartButton, Size: guigui.FixedSize(6 * u)},
		guigui.LinearLayoutItem{Size: guigui.FlexibleSize(1)},
	)
	buttons := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       u / 2,
		Items:     r.buttonItems,
	}

	r.msItems = slices.Delete(r.msItems, 0, len(r.msItems))
	r.msItems = append(r.msItems,
		guigui.LinearLayoutItem{Widget: &r.msDownButton, Size: guigui.FixedSize(5 * u)},
		guigui.LinearLayoutItem{Widget: &r.msInput, Size: guigui.FixedSize(10 * u)},
		guigui.LinearLayoutItem{Widget: &r.msUpButton, Size: guigui.FixedSize(5 * u)},
		guigui.LinearLayoutItem{Size: guigui.FlexibleSize(1)},
	)
	msControls := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       u / 2,
		Items:     r.msItems,
	}

	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{Widget: &r.fileText, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Widget: &r.statusText, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Widget: &r.clockText, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Widget: &r.msText, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Widget: &r.waveform, Size: guigui.FixedSize(bounds.Dy() * 8 / 10)},
		guigui.LinearLayoutItem{Layout: &buttons, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Layout: &msControls, Size: guigui.FixedSize(u)},
	)
	if r.state.errText != "" {
		r.layoutItems = append(r.layoutItems, guigui.LinearLayoutItem{Widget: &r.errText, Size: guigui.FixedSize(u)})
	}

	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Gap:       u / 2,
		Padding:   guigui.Padding{Start: u / 2, Top: u / 2, End: u / 2, Bottom: u / 2},
		Items:     r.layoutItems,
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (r *root) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	r.printPressedKeys()

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if r.state.playing {
			r.state.playing = false
		} else {
			r.state.playing = true
		}
		guigui.RequestRebuild(r)
		guigui.RequestRedraw(&r.waveform)
	}
	r.applyPlaybackState()
	if !r.state.playing {
		if ebiten.IsKeyPressed(ebiten.KeyRight) {
			r.seekAndPause(r.state.CurrentMS + 1)
			guigui.RequestRebuild(r)
			guigui.RequestRedraw(&r.waveform)
			return nil
		}
		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			r.seekAndPause(r.state.CurrentMS - 5)
			guigui.RequestRebuild(r)
			guigui.RequestRedraw(&r.waveform)
			return nil
		}
	}

	if r.state.player == nil {
		return nil
	}

	ms := int64(r.state.player.Position() / time.Millisecond)
	if r.state.nowPlaying != nil {
		ms = r.state.nowPlaying.MS()
	}
	if r.state.playing && r.state.durationMS > 0 && ms >= r.state.durationMS {
		_ = r.state.player.SetPosition(0)
		ms = 0
	}
	if ms != r.state.CurrentMS {
		r.setCurrentMS(ms)
		guigui.RequestRebuild(r)
		guigui.RequestRedraw(&r.waveform)
	}

	return nil
}

func (r *root) printPressedKeys() {
	for _, key := range loggedNumberKeys {
		if inpututil.IsKeyJustPressed(key) {
			lane := capturedHookKeyIndexes[key]
			r.state.captured[lane] = append(r.state.captured[lane], r.state.CurrentMS)
			r.state.newCaptured[lane] = append(r.state.newCaptured[lane], r.state.CurrentMS)
			fmt.Printf("key=%s ms=%d\n", key.String(), r.state.CurrentMS)
		}
	}
}

func (r *root) applyPlaybackState() {
	if r.state.player == nil {
		return
	}

	if r.state.playing {
		r.state.errText = ""
		r.state.player.Play()
		return
	}

	r.setCurrentMS(int64(r.state.player.Position() / time.Millisecond))
	r.state.player.Pause()
}

func (r *root) restart() {
	r.state.player.Pause()
	_ = r.state.player.SetPosition(0)
	r.setCurrentMS(0)
}

func (r *root) seekAndPause(ms int64) {
	ms = clampMS(ms, r.state.durationMS)
	r.state.player.Pause()
	_ = r.state.player.SetPosition(time.Duration(ms) * time.Millisecond)
	r.setCurrentMS(ms)
}

func (r *root) setCurrentMS(ms int64) {
	r.state.CurrentMS = clampMS(ms, r.state.durationMS)
	r.state.msInputValue = strconv.FormatInt(r.state.CurrentMS, 10)
}

func (w *waveformWidget) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if w.state == nil || w.state.waveformImage == nil {
		return
	}

	bounds := widgetBounds.Bounds()
	imgBounds := w.state.waveformImage.Bounds()
	scale := min(float64(bounds.Dx())/float64(imgBounds.Dx()), float64(bounds.Dy())/float64(imgBounds.Dy()))
	waveWidth := int(math.Round(float64(imgBounds.Dx()) * scale))
	waveHeight := int(math.Round(float64(imgBounds.Dy()) * scale))
	waveRect := image.Rectangle{
		Min: image.Pt(bounds.Min.X+(bounds.Dx()-waveWidth)/2, bounds.Min.Y+(bounds.Dy()-waveHeight)/2),
	}
	waveRect.Max = waveRect.Min.Add(image.Pt(waveWidth, waveHeight))
	w.state.waveformRect = waveRect

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(waveRect.Min.X), float64(waveRect.Min.Y))
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(w.state.waveformImage, op)

	if w.state.durationMS <= 0 {
		return
	}

	drawTimingGuides(dst, w.state.durationMS, waveRect)
	drawCapturedHooks(dst, w.state, waveRect)

	ratio := float64(w.state.CurrentMS) / float64(w.state.durationMS)
	ratio = math.Max(0, math.Min(1, ratio))
	x := float64(waveRect.Min.X) + ratio*float64(waveRect.Dx())
	ebitenutil.DrawRect(dst, x-1.5, float64(waveRect.Min.Y), 3, float64(waveRect.Dy()), color.RGBA{R: 255, A: 255})
}

func drawTimingGuides(dst *ebiten.Image, durationMS int64, waveRect image.Rectangle) {
	guideColor := color.RGBA{R: 120, G: 120, B: 120, A: 180}
	for ms := int64(1500); ms < durationMS; ms += 1500 {
		ratio := float64(ms) / float64(durationMS)
		x := int(math.Round(float64(waveRect.Min.X) + ratio*float64(waveRect.Dx())))
		ebitenutil.DrawRect(dst, float64(x)-0.5, float64(waveRect.Min.Y), 1, float64(waveRect.Dy()), guideColor)
		drawLabel(dst, strconv.FormatInt(ms, 10), x-16, waveRect.Max.Y+14, color.White)
	}
}

func drawCapturedHooks(dst *ebiten.Image, state *appState, waveRect image.Rectangle) {
	lineColor := color.RGBA{R: 180, B: 255, A: 255}
	for lane, key := range loggedNumberKeys {
		for _, ms := range state.captured[lane] {
			ratio := float64(ms) / float64(state.durationMS)
			ratio = math.Max(0, math.Min(1, ratio))
			x := int(math.Round(float64(waveRect.Min.X) + ratio*float64(waveRect.Dx())))
			ebitenutil.DrawRect(dst, float64(x), float64(waveRect.Min.Y), 1, float64(waveRect.Dy()), lineColor)

			labelY := hookLabelY(waveRect, lane)
			drawLabel(dst, keyLabel(key), x-3, labelY, laneLabelColors[lane%len(laneLabelColors)])
		}
	}
}

func drawLabel(dst *ebiten.Image, value string, x, y int, labelColor color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(labelColor)
	text.Draw(dst, value, labelFace, op)
}

func hookLabelY(waveRect image.Rectangle, lane int) int {
	fontHeight := 13
	step := fontHeight * 2
	usableHeight := max(waveRect.Dy()-fontHeight, fontHeight)
	steps := max(usableHeight/step, 1)
	return waveRect.Min.Y + (lane%steps)*step
}

func keyLabel(key ebiten.Key) string {
	if key == ebiten.Key0 {
		return "0"
	}
	return strings.TrimPrefix(key.String(), "Digit")
}

func (w *waveformWidget) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if w.state == nil || w.state.durationMS <= 0 || !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return guigui.HandleInputResult{}
	}

	cx, cy := ebiten.CursorPosition()
	if !image.Pt(cx, cy).In(w.state.waveformRect) {
		return guigui.HandleInputResult{}
	}

	rootWidget, ok := rootFromState(w.state)
	if !ok {
		return guigui.HandleInputResult{}
	}

	ratio := float64(cx-w.state.waveformRect.Min.X) / float64(w.state.waveformRect.Dx())
	rootWidget.seekAndPause(int64(ratio * float64(w.state.durationMS)))
	guigui.RequestRebuild(rootWidget)
	guigui.RequestRedraw(w)
	return guigui.HandleInputByWidget(w)
}

var activeRoot *root

func rootFromState(state *appState) (*root, bool) {
	if activeRoot == nil || &activeRoot.state != state {
		return nil, false
	}
	return activeRoot, true
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: hooktool path/to/audio.mp3 [--new] [--delete #,#,#]")
		os.Exit(2)
	}
	newFile, deletedLanes, err := parseArgs(os.Args[2:])
	if err != nil {
		log.Fatal(err)
	}

	audioPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	state, err := loadState(audioPath, newFile, deletedLanes)
	if err != nil {
		log.Fatal(err)
	}

	r := &root{state: state}
	activeRoot = r
	defer func() {
		if err := saveCapturedTrack(r.state); err != nil {
			log.Printf("failed to save captured track: %v", err)
		}
	}()

	if err := guigui.Run(r, &guigui.RunOptions{
		Title:      "Music Hook Tool",
		WindowSize: image.Pt(1100, 520),
	}); err != nil {
		log.Fatal(err)
	}
}

func parseArgs(args []string) (bool, []int, error) {
	newFile := false
	var deletedLanes []int
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--new":
			newFile = true
		case "--delete":
			if i+1 >= len(args) {
				return false, nil, fmt.Errorf("--delete requires a comma-separated lane list")
			}
			lanes, err := parseDeletedLanes(args[i+1])
			if err != nil {
				return false, nil, err
			}
			deletedLanes = append(deletedLanes, lanes...)
			i++
		default:
			return false, nil, fmt.Errorf("unknown argument %q", args[i])
		}
	}
	if newFile && len(deletedLanes) > 0 {
		return false, nil, fmt.Errorf("--new and --delete cannot be used together")
	}
	return newFile, deletedLanes, nil
}

func parseDeletedLanes(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	lanes := make([]int, 0, len(parts))
	for _, part := range parts {
		lane, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid delete lane %q", part)
		}
		if lane < 0 {
			return nil, fmt.Errorf("delete lane %d is negative", lane)
		}
		lanes = append(lanes, lane)
	}
	return lanes, nil
}

func loadState(audioPath string, newFile bool, deletedLanes []int) (appState, error) {
	waveformImage, err := loadWaveform(audioPath)
	if err != nil {
		return appState{}, err
	}

	audioContext := audio.NewContext(sampleRate)
	pcm, durationMS, err := loadPCM(audioPath)
	if err != nil {
		return appState{}, err
	}

	track := music.Track{}
	if !newFile {
		var err error
		track, err = loadTrackJSON(audioPath)
		if err != nil {
			return appState{}, err
		}
	}
	if track.Name == "" {
		track = music.Track{
			Name:  trackName(audioPath),
			File:  audioPath,
			Hooks: nil,
		}
	}
	if track.File == "" {
		track.File = audioPath
	}
	if len(deletedLanes) > 0 {
		if err := deleteHookLanes(&track, deletedLanes); err != nil {
			return appState{}, err
		}
	}
	captured := map[int][]int64{}
	populateCapturedHooks(captured, track.Hooks)

	playerTrack := music.Track{
		Name:  trackName(audioPath),
		File:  audioPath,
		Hooks: nil,
	}
	player := audioContext.NewPlayerFromBytes(pcm)
	nowPlaying := music.NewNowPlaying(playerTrack, player)

	return appState{
		audioPath:     audioPath,
		newFile:       newFile,
		deleted:       len(deletedLanes) > 0,
		audioContext:  audioContext,
		player:        player,
		nowPlaying:    nowPlaying,
		track:         track,
		captured:      captured,
		newCaptured:   map[int][]int64{},
		durationMS:    durationMS,
		CurrentMS:     0,
		playing:       false,
		waveformImage: waveformImage,
		msInputValue:  "0",
	}, nil
}

func loadTrackJSON(audioPath string) (music.Track, error) {
	data, err := os.ReadFile(trackJSONPath(audioPath))
	if err != nil {
		if os.IsNotExist(err) {
			return music.Track{}, nil
		}
		return music.Track{}, err
	}

	var track music.Track
	if err := json.Unmarshal(data, &track); err != nil {
		return music.Track{}, err
	}
	return track, nil
}

func populateCapturedHooks(captured map[int][]int64, hooks [][]int64) {
	for lane, laneHooks := range hooks {
		if len(laneHooks) == 0 {
			continue
		}
		captured[lane] = append(captured[lane], laneHooks...)
	}
}

func deleteHookLanes(track *music.Track, lanes []int) error {
	for _, lane := range lanes {
		if lane >= len(track.Hooks) || len(track.Hooks[lane]) == 0 {
			return fmt.Errorf("cannot delete missing lane %d", lane)
		}
	}
	for _, lane := range lanes {
		track.Hooks[lane] = nil
	}
	track.Hooks = compactHookLanes(track.Hooks)
	return nil
}

func saveCapturedTrack(state appState) error {
	track := state.track
	if !state.deleted && !hasCapturedHooks(state.newCaptured) {
		return nil
	}
	appendCapturedHooks(&track, state.newCaptured)
	sortHookLanes(track.Hooks)
	track.Hooks = compactHookLanes(track.Hooks)

	jsonPath := trackJSONPath(state.audioPath)
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		return err
	}

	data, err := marshalTrackJSON(track)
	if err != nil {
		return err
	}

	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("saved hooks=%s\n", jsonPath)
	return nil
}

func hasCapturedHooks(captured map[int][]int64) bool {
	for _, laneHooks := range captured {
		if len(laneHooks) > 0 {
			return true
		}
	}
	return false
}

func appendCapturedHooks(track *music.Track, captured map[int][]int64) {
	for lane := range loggedNumberKeys {
		laneHooks := captured[lane]
		if len(laneHooks) == 0 {
			continue
		}
		for len(track.Hooks) <= lane {
			track.Hooks = append(track.Hooks, nil)
		}
		track.Hooks[lane] = append(track.Hooks[lane], laneHooks...)
	}
}

func sortHookLanes(hooks [][]int64) {
	for _, laneHooks := range hooks {
		slices.Sort(laneHooks)
	}
}

func marshalTrackJSON(track music.Track) ([]byte, error) {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString(fmt.Sprintf("  \"name\": %q,\n", track.Name))
	b.WriteString(fmt.Sprintf("  \"file\": %q,\n", track.File))
	b.WriteString("  \"hooks\": [\n")
	for lane, laneHooks := range track.Hooks {
		laneData, err := json.Marshal(laneHooks)
		if err != nil {
			return nil, err
		}
		b.WriteString("    ")
		b.Write(laneData)
		if lane != len(track.Hooks)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ]\n")
	b.WriteString("}\n")
	return []byte(b.String()), nil
}

func compactHookLanes(hooks [][]int64) [][]int64 {
	compact := make([][]int64, 0, len(hooks))
	for _, laneHooks := range hooks {
		if len(laneHooks) > 0 {
			compact = append(compact, laneHooks)
		}
	}
	return compact
}

func trackJSONPath(audioPath string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(audioPath)), "json", "track_"+trackName(audioPath)+".json")
}

func trackName(audioPath string) string {
	base := filepath.Base(audioPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func loadPCM(audioPath string) ([]byte, int64, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(audioPath))
	var stream io.Reader
	switch ext {
	case ".mp3":
		stream, err = mp3.DecodeWithSampleRate(sampleRate, file)
	case ".ogg", ".oga":
		stream, err = vorbis.DecodeWithSampleRate(sampleRate, file)
	case ".wav":
		stream, err = wav.DecodeWithSampleRate(sampleRate, file)
	default:
		return nil, 0, fmt.Errorf("unsupported audio extension %q", ext)
	}
	if err != nil {
		return nil, 0, err
	}

	pcm, err := io.ReadAll(stream)
	if err != nil {
		return nil, 0, err
	}
	if len(pcm) == 0 {
		return nil, 0, fmt.Errorf("decoded audio is empty")
	}
	durationMS := int64(len(pcm)) * 1000 / int64(sampleRate*4)
	return pcm, durationMS, nil
}

func loadWaveform(audioPath string) (*ebiten.Image, error) {
	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-i", audioPath,
		"-filter_complex", "showwavespic=s=1600x300:colors=white",
		"-frames:v", "1",
		"-f", "image2pipe",
		"-vcodec", "png",
		"pipe:1",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg waveform failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	img, err := png.Decode(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

func formatClock(ms int64) string {
	minutes := ms / 60000
	seconds := (ms % 60000) / 1000
	millis := ms % 1000
	return fmt.Sprintf("%02d:%02d:%03d", minutes, seconds, millis)
}

func clampMS(ms, durationMS int64) int64 {
	if ms < 0 {
		return 0
	}
	if durationMS > 0 && ms > durationMS {
		return durationMS
	}
	return ms
}
