package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/controls"
	"github.com/ninesl/dice-will-roll/render"
)

const (
	defaultShaderPath             = "render/shaders/kages/rocks/moon_rock.kage"
	rotationRadiansPerPixel       = math.Pi / 180
	rotationUniformParameterIndex = 0
)

type moonRockUniforms struct {
	Rotation render.Vec3

	LightSource     [3]float32
	InnerColorDark  [3]float32
	InnerColorLight [3]float32
	OuterColorDark  [3]float32
	OuterColorLight [3]float32
}

type uniformParameter struct {
	name    string
	values  []*float32
	text    string
	invalid bool

	label basicwidget.Text
	input guigui.WidgetWithSize[*basicwidget.TextInput]
}

type root struct {
	guigui.DefaultWidget

	shaderPath string
	status     string
	loadFailed bool
	uniforms   moonRockUniforms

	background basicwidget.Background
	pathText   basicwidget.Text
	statusText basicwidget.Text
	loadButton basicwidget.Button
	preview    shaderPreview

	uniformHeading    basicwidget.Text
	uniformForm       basicwidget.Form
	uniformPanel      basicwidget.Panel
	uniformParameters []uniformParameter
	uniformFormItems  []basicwidget.FormItem

	buttonItems  []guigui.LinearLayoutItem
	contentItems []guigui.LinearLayoutItem
	layoutItems  []guigui.LinearLayoutItem
}

func (r *root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.pathText)
	adder.AddWidget(&r.preview)
	adder.AddWidget(&r.uniformPanel)
	adder.AddWidget(&r.loadButton)
	adder.AddWidget(&r.statusText)

	r.pathText.SetValue("Shader: " + r.shaderPath)
	r.pathText.SetSelectable(true)

	r.configureUniformPanel(context)

	r.loadButton.SetText("Load")
	r.loadButton.SetType(basicwidget.ButtonTypePrimary)
	r.loadButton.OnUp(func(context *guigui.Context) {
		r.loadShader()
		guigui.RequestRebuild(r)
	})

	r.statusText.SetValue(r.status)
	r.statusText.SetSelectable(true)
	r.statusText.SetMultiline(true)
	r.statusText.SetWrapMode(basicwidget.WrapModeAnywhere)
	if r.loadFailed {
		r.statusText.SetSemanticColor(basicwidgetdraw.SemanticColorDanger)
	} else {
		r.statusText.SetSemanticColor(basicwidgetdraw.SemanticColorSuccess)
	}

	return nil
}

func (r *root) configureUniformPanel(context *guigui.Context) {
	r.uniformHeading.SetValue("Uniforms (vectors are comma-separated)")
	r.uniformHeading.SetBold(true)

	r.uniformFormItems = slices.Delete(r.uniformFormItems, 0, len(r.uniformFormItems))
	r.uniformFormItems = append(r.uniformFormItems, basicwidget.FormItem{PrimaryWidget: &r.uniformHeading})
	inputWidth := 9 * basicwidget.UnitSize(context)
	for index := range r.uniformParameters {
		parameter := &r.uniformParameters[index]
		parameter.label.SetValue(parameter.name)

		input := parameter.input.Widget()
		input.SetValue(parameter.text)
		input.SetError(parameter.invalid)
		input.SetHorizontalAlign(basicwidget.HorizontalAlignRight)
		input.SetTabular(true)
		parameter.input.SetFixedWidth(inputWidth)
		input.OnValueChanged(func(_ *guigui.Context, value string, _ bool) {
			r.setUniformParameter(index, value)
		})

		r.uniformFormItems = append(r.uniformFormItems, basicwidget.FormItem{
			PrimaryWidget:   &parameter.label,
			SecondaryWidget: &parameter.input,
		})
	}
	r.uniformForm.SetItems(r.uniformFormItems)
	r.uniformPanel.SetStyle(basicwidget.PanelStyleSide)
	r.uniformPanel.SetBorders(basicwidget.PanelBorders{Start: true})
	r.uniformPanel.SetContentConstraints(basicwidget.PanelContentConstraintsFixedWidth)
	r.uniformPanel.SetContent(&r.uniformForm)
}

func (r *root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, bounds)
	u := basicwidget.UnitSize(context)

	r.contentItems = slices.Delete(r.contentItems, 0, len(r.contentItems))
	r.contentItems = append(r.contentItems,
		guigui.LinearLayoutItem{Widget: &r.preview, Size: guigui.FlexibleSize(1)},
		guigui.LinearLayoutItem{Widget: &r.uniformPanel, Size: guigui.FixedSize(min(20*u, bounds.Dx()/2))},
	)
	content := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       u / 2,
		Items:     r.contentItems,
	}

	r.buttonItems = slices.Delete(r.buttonItems, 0, len(r.buttonItems))
	r.buttonItems = append(r.buttonItems,
		guigui.LinearLayoutItem{Widget: &r.loadButton, Size: guigui.FixedSize(6 * u)},
		guigui.LinearLayoutItem{Size: guigui.FlexibleSize(1)},
	)
	buttons := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       u / 2,
		Items:     r.buttonItems,
	}

	statusHeight := u
	if r.loadFailed {
		statusHeight = 4 * u
	}
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{Widget: &r.pathText, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Layout: &content, Size: guigui.FlexibleSize(1)},
		guigui.LinearLayoutItem{Layout: &buttons, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Widget: &r.statusText, Size: guigui.FixedSize(statusHeight)},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Gap:       u / 2,
		Padding:   guigui.Padding{Start: u / 2, Top: u / 2, End: u / 2, Bottom: u / 2},
		Items:     r.layoutItems,
	}).LayoutWidgets(context, bounds, layouter)
}

func (r *root) loadShader() {
	shaderBytes, err := os.ReadFile(r.shaderPath)
	if err != nil {
		r.status = fmt.Sprintf("Load failed: %v", err)
		r.loadFailed = true
		return
	}

	shader, err := ebiten.NewShader(shaderBytes)
	if err != nil {
		r.status = fmt.Sprintf("Compile failed: %v", err)
		r.loadFailed = true
		return
	}

	if r.preview.shader != nil {
		r.preview.shader.Deallocate()
	}
	r.preview.shader = shader
	r.status = "Loaded successfully at " + time.Now().Format("15:04:05")
	r.loadFailed = false
}

func (r *root) initializeUniformParameters() {
	r.uniformParameters = []uniformParameter{
		{name: "Rotation (x, y, z)", values: []*float32{&r.uniforms.Rotation.X, &r.uniforms.Rotation.Y, &r.uniforms.Rotation.Z}},
		{name: "LightSource (x, y, z)", values: []*float32{&r.uniforms.LightSource[0], &r.uniforms.LightSource[1], &r.uniforms.LightSource[2]}},
		{name: "InnerColorDark (r, g, b)", values: []*float32{&r.uniforms.InnerColorDark[0], &r.uniforms.InnerColorDark[1], &r.uniforms.InnerColorDark[2]}},
		{name: "InnerColorLight (r, g, b)", values: []*float32{&r.uniforms.InnerColorLight[0], &r.uniforms.InnerColorLight[1], &r.uniforms.InnerColorLight[2]}},
		{name: "OuterColorDark (r, g, b)", values: []*float32{&r.uniforms.OuterColorDark[0], &r.uniforms.OuterColorDark[1], &r.uniforms.OuterColorDark[2]}},
		{name: "OuterColorLight (r, g, b)", values: []*float32{&r.uniforms.OuterColorLight[0], &r.uniforms.OuterColorLight[1], &r.uniforms.OuterColorLight[2]}},
	}
	for index := range r.uniformParameters {
		parameter := &r.uniformParameters[index]
		parameter.text = formatUniformValues(parameter.values)
	}
}

func (r *root) setUniformParameter(index int, text string) {
	parameter := &r.uniformParameters[index]
	parameter.text = text
	values, ok := parseUniformValues(text, len(parameter.values))
	parameter.invalid = !ok
	parameter.input.Widget().SetError(!ok)
	if !ok {
		return
	}

	for index, value := range values {
		*parameter.values[index] = value
	}
	guigui.RequestRedraw(&r.preview)
}

func (r *root) synchronizeRotationParameters() {
	parameter := &r.uniformParameters[rotationUniformParameterIndex]
	parameter.text = formatUniformValues(parameter.values)
	parameter.invalid = false
	input := parameter.input.Widget()
	input.SetError(false)
	input.ForceSetValue(parameter.text)
}

func formatUniformValues(values []*float32) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.FormatFloat(float64(*value), 'g', 6, 32)
	}
	return strings.Join(parts, ", ")
}

func parseUniformValues(text string, expected int) ([]float32, bool) {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	if len(parts) != expected {
		return nil, false
	}

	values := make([]float32, len(parts))
	for index, part := range parts {
		value, err := strconv.ParseFloat(part, 32)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, false
		}
		values[index] = float32(value)
	}
	return values, true
}

func defaultMoonRockUniforms() moonRockUniforms {
	return moonRockUniforms{
		LightSource:     [3]float32{0, 0, -3},
		InnerColorDark:  uniformVec3(render.WhiteDark),
		InnerColorLight: uniformVec3(render.WhiteMid),
		OuterColorDark:  uniformVec3(render.WhiteLight),
		OuterColorLight: uniformVec3(render.WhiteBright),
	}
}

func uniformVec3(value render.Vec3) [3]float32 {
	return [3]float32{value.X, value.Y, value.Z}
}

type shaderPreview struct {
	guigui.DefaultWidget

	shader            *ebiten.Shader
	offscreen         *ebiten.Image
	uniforms          *moonRockUniforms
	mouse             controls.MouseInfo
	dragStart         image.Point
	dragStartRotation render.Vec3
	dragging          bool
	rightDrag         bool
	onRotationChanged func()
}

func (s *shaderPreview) Tick(_ *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	s.mouse.Update()
	if s.uniforms == nil {
		return nil
	}

	bounds := widgetBounds.Bounds()
	cursor := image.Pt(int(s.mouse.Position.X), int(s.mouse.Position.Y))
	inside := cursor.In(bounds)
	if inside && (s.mouse.Clicked || s.mouse.RightClicked) {
		s.dragStart = cursor
		s.dragStartRotation = s.uniforms.Rotation
		s.dragging = true
		s.rightDrag = s.mouse.RightClicked
	}
	if !s.dragging {
		return nil
	}
	if s.rightDrag {
		if s.mouse.RightReleased || !s.mouse.RightDown {
			s.dragging = false
			return nil
		}
	} else if s.mouse.Released || !s.mouse.Down {
		s.dragging = false
		return nil
	}
	if !inside {
		return nil
	}

	deltaX := float32(cursor.X-s.dragStart.X) * rotationRadiansPerPixel
	deltaY := float32(cursor.Y-s.dragStart.Y) * rotationRadiansPerPixel
	rotationX := s.dragStartRotation.X + deltaX
	if s.rightDrag {
		rotationZ := s.dragStartRotation.Z + deltaY
		if s.uniforms.Rotation.X == rotationX && s.uniforms.Rotation.Z == rotationZ {
			return nil
		}
		s.uniforms.Rotation.Z = rotationZ
	} else {
		rotationY := s.dragStartRotation.Y + deltaY
		if s.uniforms.Rotation.X == rotationX && s.uniforms.Rotation.Y == rotationY {
			return nil
		}
		s.uniforms.Rotation.Y = rotationY
	}
	s.uniforms.Rotation.X = rotationX
	if s.onRotationChanged != nil {
		s.onRotationChanged()
	}
	guigui.RequestRedraw(s)
	return nil
}

func (s *shaderPreview) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	if bounds.Empty() {
		return
	}

	if s.offscreen == nil || s.offscreen.Bounds().Size() != bounds.Size() {
		if s.offscreen != nil {
			s.offscreen.Deallocate()
		}
		s.offscreen = ebiten.NewImage(bounds.Dx(), bounds.Dy())
	}
	s.offscreen.Fill(color.RGBA{R: 18, G: 20, B: 24, A: 255})

	if s.shader != nil && s.uniforms != nil {
		uniforms := s.uniforms
		options := &ebiten.DrawRectShaderOptions{
			Uniforms: map[string]any{
				"Rotation":        uniforms.Rotation.KageVec3(),
				"LightSource":     uniforms.LightSource,
				"InnerColorDark":  uniforms.InnerColorDark,
				"InnerColorLight": uniforms.InnerColorLight,
				"OuterColorDark":  uniforms.OuterColorDark,
				"OuterColorLight": uniforms.OuterColorLight,
			},
		}
		s.offscreen.DrawRectShader(bounds.Dx(), bounds.Dy(), s.shader, options)
	}

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	dst.DrawImage(s.offscreen, options)
}

func main() {
	shaderPath := flag.String("shader", defaultShaderPath, "Path to the live Kage shader file")
	flag.Parse()

	absShaderPath, err := filepath.Abs(*shaderPath)
	if err != nil {
		log.Fatal(err)
	}

	r := &root{
		shaderPath: absShaderPath,
		uniforms:   defaultMoonRockUniforms(),
	}
	r.preview.uniforms = &r.uniforms
	r.initializeUniformParameters()
	r.preview.onRotationChanged = r.synchronizeRotationParameters
	r.loadShader()
	defer func() {
		if r.preview.shader != nil {
			r.preview.shader.Deallocate()
		}
		if r.preview.offscreen != nil {
			r.preview.offscreen.Deallocate()
		}
	}()

	if err := guigui.Run(r, &guigui.RunOptions{
		Title:         "Shader Preview: " + filepath.Base(absShaderPath),
		WindowSize:    image.Pt(960, 760),
		WindowMinSize: image.Pt(640, 480),
	}); err != nil {
		log.Fatal(err)
	}
}
