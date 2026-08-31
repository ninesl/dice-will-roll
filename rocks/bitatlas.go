package rocks

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
)

const (
	BitSpriteSlopeCodeCount = 16
	atlasSlopeStates        = 15
	AtlasSlopeZFrames       = 16
	nativeRockSizeScore     = 6
	maximumRockSizeScale    = 1.8
	PackedRockSpritePixels  = 80
)

type rockSpriteLookup [atlasSlopeStates * atlasSlopeStates * AtlasSlopeZFrames]*ebiten.Image
type rockScaleLookup [len(rockAmountScales)][BitSpriteSlopeCodeCount]float32
type rockCollisionLookup [len(rockAmountScales)][BitSpriteSlopeCodeCount]uint32

// 100, 1000, 10000, 10000, 1_000_000
var rockAmountScales = [...]float32{1.25, 1.0, 1.0, .5, .25, .1}

// Gets initialized with a flat lookup for every packed X/Y slope and Z slope frame.
type RockSpriteAtlas struct {
	*render.Sprite

	Frames           rockSpriteLookup
	Scales           rockScaleLookup
	HalfDrawSizes    rockScaleLookup
	CollisionLookups rockCollisionLookup
	MouseRadius      [len(rockAmountScales)]uint32
	SizeScale        int
}

func atlasSlopeRadians(slopeCode uint8) float32 {
	slopeIndex := int(slopeCode) - 1
	angleDegrees := slopeIndex * (360 / atlasSlopeStates)
	return float32(angleDegrees) * (math.Pi / 180)
}

// filterIndex maps X/Y slopes -7..7 and Z slope frames 0..15 to the dense frame range.
// Adding 7 maps each slope to 0..14.
func filterIndex(slopeX, slopeY, slopeZ int) int {
	return ((slopeX+7)*atlasSlopeStates+slopeY+7)*AtlasSlopeZFrames + slopeZ
}

// RockAmountScaleIndex derives a collection's scale tier from its rock count.
func RockAmountScaleIndex(rockNum int) int {
	const firstThreshold = 100

	amountScale := 0
	threshold := firstThreshold
	for amountScale < len(rockAmountScales)-1 && rockNum > threshold {
		amountScale++
		threshold *= 10
	}
	return amountScale
}

func rockSizeScale(sizeScore int) float32 {
	const scaleStep = (maximumRockSizeScale - 1.0) / (BitSpriteSlopeCodeCount - 1 - nativeRockSizeScore)
	return 1.0 + float32(sizeScore-nativeRockSizeScore)*scaleStep
}

// InitRockAtlas paints one atlas, then indexes shared subimages in dense frame order.
func InitRockAtlas(shader *ebiten.Shader) *RockSpriteAtlas {
	frameCount := atlasSlopeStates * atlasSlopeStates * AtlasSlopeZFrames
	sheetColumns := calculateSheetCols(frameCount)
	sheetRows := (frameCount + sheetColumns - 1) / sheetColumns
	pixelSize := PackedRockSpritePixels
	sheetImage := ebiten.NewImage(pixelSize*sheetColumns, pixelSize*sheetRows)
	frameImage := ebiten.NewImage(pixelSize, pixelSize)

	uniforms := map[string]interface{}{
		"Time":            0.0,
		"Resolution":      []float32{float32(pixelSize), float32(pixelSize)},
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

			for slopeZ := range AtlasSlopeZFrames {
				uniforms["RotationZ"] = float32(slopeZ) * (2 * math.Pi / AtlasSlopeZFrames)
				frameImage.Clear()
				frameImage.DrawRectShader(pixelSize, pixelSize, shader, shaderOptions)

				slopeIndex := (int(slopeX)-1)*atlasSlopeStates + int(slopeY) - 1
				frameIndex := slopeIndex*AtlasSlopeZFrames + slopeZ
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

	atlas := &RockSpriteAtlas{
		Sprite: &render.Sprite{
			Image:       sheetImage,
			SpriteSheet: spriteSheet,
		},
		Frames: frames,
	}
	initializeScaleLookups(atlas, pixelSize)
	initializeCollisionLookups(atlas)
	initializeMouseRadius(atlas)
	return atlas
}

func initializeMouseRadius(atlas *RockSpriteAtlas) {
	for amountScaleIndex := range rockAmountScales {
		atlas.MouseRadius[amountScaleIndex] = uint32(math.Ceil(
			float64(atlas.HalfDrawSizes[amountScaleIndex][BitSpriteSlopeCodeCount-1])))
		if atlas.MouseRadius[amountScaleIndex] > 181 {
			panic("rock mouse radius exceeds safe 16-bit squared-distance range")
		}
	}
}

func initializeScaleLookups(atlas *RockSpriteAtlas, pixelSize int) {
	for amountScaleIndex, amountScale := range rockAmountScales {
		for sizeScore := 1; sizeScore < BitSpriteSlopeCodeCount; sizeScore++ {
			drawScale := rockSizeScale(sizeScore) * amountScale
			atlas.Scales[amountScaleIndex][sizeScore] = drawScale
			atlas.HalfDrawSizes[amountScaleIndex][sizeScore] = float32(pixelSize) * drawScale / 2
			radiusValue := math.Ceil(float64(atlas.HalfDrawSizes[amountScaleIndex][sizeScore]))
			if radiusValue > 181 {
				panic("rock hover radius exceeds safe 16-bit squared-distance range")
			}
		}
	}
}

func initializeCollisionLookups(atlas *RockSpriteAtlas) {
	for amountScaleIndex := range rockAmountScales {
		for sizeScore := 1; sizeScore < BitSpriteSlopeCodeCount; sizeScore++ {
			atlas.CollisionLookups[amountScaleIndex][sizeScore] = uint32(math.Ceil(
				float64(atlas.HalfDrawSizes[amountScaleIndex][sizeScore] * 2.0 / 3.0)))
			if atlas.CollisionLookups[amountScaleIndex][sizeScore] > math.MaxInt16 {
				panic("rock collision radius exceeds signed 16-bit range")
			}
		}
	}
}
