package rocks

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
)

const (
	BitSpriteSlopeCodeCount = 16
	atlasSlopeStates        = 15

	bitSpriteDegreesPerFrame = 24
	AtlasRotationFrames      = 360 / bitSpriteDegreesPerFrame
)

type rockSpriteLookup [atlasSlopeStates * atlasSlopeStates * AtlasRotationFrames]*ebiten.Image
type rockScaleLookup [len(rockAmountScales)][BitSpriteSlopeCodeCount]float64

var rockAmountScales = [...]float64{2.0, 1.5, 1.0}

// Gets intialized with flat lookup for every combo of packed slope/rotation for a sprite for full 360 degrees
type RockSpriteAtlas struct {
	*render.Sprite

	// TODO: benchmark *rockSpriteLookup
	Frames        rockSpriteLookup
	Scales        rockScaleLookup
	HalfDrawSizes rockScaleLookup
	AmountScale   int
}

func atlasSlopeRadians(slopeCode uint8) float32 {
	slopeIndex := int(slopeCode) - 1
	angleDegrees := slopeIndex * (360 / atlasSlopeStates)
	return float32(angleDegrees) * (math.Pi / 180)
}

// filterIndex maps slopes -7..7 and rotations 0..14 to the dense frame range
// 0..3374. Adding 7 maps each slope to 0..14. Each X slope contains 225
// frames, and each Y slope contains 15 rotation frames.
func filterIndex(slopeX, slopeY, rotation int) int {
	return ((slopeX+7)*atlasSlopeStates+slopeY+7)*AtlasRotationFrames + rotation
}

// rockAmountScaleIndex derives the global scale tier from the initial requested
// rock amount. The tier remains fixed as rocks are removed or split.
func rockAmountScaleIndex(rockNum int) int {
	switch {
	case rockNum <= 100:
		return 0
	case rockNum <= 1000:
		return 1
	default:
		return 2
	}
}

// InitRockAtlas paints one atlas, then indexes shared subimages in dense frame order.
func InitRockAtlas(shader *ebiten.Shader, tileSize float32, rockNum int) *RockSpriteAtlas {
	frameCount := atlasSlopeStates * atlasSlopeStates * AtlasRotationFrames
	sheetColumns := calculateSheetCols(frameCount)
	sheetRows := (frameCount + sheetColumns - 1) / sheetColumns
	pixelSize := int(tileSize)
	sheetImage := ebiten.NewImage(pixelSize*sheetColumns, pixelSize*sheetRows)
	frameImage := ebiten.NewImage(pixelSize, pixelSize)

	uniforms := map[string]interface{}{
		"Time":            0.0,
		"Resolution":      []float32{tileSize, tileSize},
		"Mouse":           render.Vec2{}.KageVec2(),
		"RotationX":       float32(0),
		"RotationY":       float32(0),
		"RotationZ":       float32(0),
		"LightSource":     []float32{0, 0, -3},
		"InnerColorDark":  render.WhiteDark.KageVec3(),
		"InnerColorLight": render.WhiteMid.KageVec3(),
		"OuterColorDark":  render.WhiteLight.KageVec3(),
		"OuterColorLight": render.WhiteBright.KageVec3(),
	}
	shaderOptions := &ebiten.DrawRectShaderOptions{Uniforms: uniforms}
	drawOptions := &ebiten.DrawImageOptions{}
	for slopeX := uint8(1); slopeX < BitSpriteSlopeCodeCount; slopeX++ {
		uniforms["RotationX"] = atlasSlopeRadians(slopeX)

		for slopeY := uint8(1); slopeY < BitSpriteSlopeCodeCount; slopeY++ {
			uniforms["RotationY"] = atlasSlopeRadians(slopeY)

			for rotationFrame := range AtlasRotationFrames {
				uniforms["RotationZ"] = float32(rotationFrame*bitSpriteDegreesPerFrame) * (math.Pi / 180)
				frameImage.Clear()
				frameImage.DrawRectShader(pixelSize, pixelSize, shader, shaderOptions)

				slopeIndex := (int(slopeX)-1)*atlasSlopeStates + int(slopeY) - 1
				frameIndex := slopeIndex*AtlasRotationFrames + rotationFrame
				column := frameIndex % sheetColumns
				row := frameIndex / sheetColumns
				drawOptions.GeoM.Reset()
				drawOptions.GeoM.Translate(float64(column*pixelSize), float64(row*pixelSize))
				sheetImage.DrawImage(frameImage, drawOptions)
			}
		}
	}

	spriteSheet := render.NewSpriteSheet(sheetColumns, sheetRows, pixelSize)

	var frames rockSpriteLookup
	for frameIndex := range frames {
		frames[frameIndex] = sheetImage.SubImage(spriteSheet.Rect(frameIndex)).(*ebiten.Image)
	}

	var scales rockScaleLookup
	var halfDrawSizes rockScaleLookup
	for amountScaleIndex, amountScale := range rockAmountScales {
		for sizeCode := 1; sizeCode < BitSpriteSlopeCodeCount; sizeCode++ {
			sizeScale := 0.2 + float64(sizeCode-1)/14
			drawScale := sizeScale * amountScale
			scales[amountScaleIndex][sizeCode] = drawScale
			halfDrawSizes[amountScaleIndex][sizeCode] = float64(pixelSize) * drawScale / 2
		}
	}

	return &RockSpriteAtlas{
		Sprite: &render.Sprite{
			Image:       sheetImage,
			SpriteSheet: spriteSheet,
		},
		Frames:        frames,
		Scales:        scales,
		HalfDrawSizes: halfDrawSizes,
		AmountScale:   rockAmountScaleIndex(rockNum),
	}
}
