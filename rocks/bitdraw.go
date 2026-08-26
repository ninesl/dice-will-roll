package rocks

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ninesl/dice-will-roll/render"
)

const (
	BitSpriteSlopeCodeCount = 16
	bitSpriteSlopeStates    = 15

	bitSpriteDegreesPerFrame = 24
	BitSpriteRotationFrames  = 360 / bitSpriteDegreesPerFrame
)

type rockSpriteLookup [bitSpriteSlopeStates * bitSpriteSlopeStates * BitSpriteRotationFrames]*ebiten.Image
type rockScaleLookup [BitSpriteSlopeCodeCount]float64

// Gets intialized with flat lookup for every combo of packed slope/rotation for a sprite for full 360 degrees
type RockSpriteAtlas struct {
	*render.Sprite

	// TODO: benchmark *rockSpriteLookup
	Frames rockSpriteLookup
	Scales rockScaleLookup
}

func initRockScales() rockScaleLookup {
	var scales rockScaleLookup
	for sizeCode := 1; sizeCode < len(scales); sizeCode++ {
		scales[sizeCode] = 0.2 + float64(sizeCode-1)/14
	}
	return scales
}

func bitSpriteSlopeRadians(slopeCode uint8) float32 {
	slopeIndex := int(slopeCode) - 1
	angleDegrees := slopeIndex * (360 / bitSpriteSlopeStates)
	return float32(angleDegrees) * (math.Pi / 180)
}

// InitRockAtlas paints one atlas, then indexes shared subimages in dense frame order.
func InitRockAtlas(shader *ebiten.Shader, tileSize float32) *RockSpriteAtlas {
	frameCount := bitSpriteSlopeStates * bitSpriteSlopeStates * BitSpriteRotationFrames
	sheetColumns := calculateSheetCols(frameCount)
	sheetRows := (frameCount + sheetColumns - 1) / sheetColumns
	pixelSize := int(tileSize)
	sheetImage := ebiten.NewImage(pixelSize*sheetColumns, pixelSize*sheetRows)
	frameImage := ebiten.NewImage(pixelSize, pixelSize)
	spriteSheet := render.NewSpriteSheet(sheetColumns, sheetRows, pixelSize)

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
		uniforms["RotationX"] = bitSpriteSlopeRadians(slopeX)

		for slopeY := uint8(1); slopeY < BitSpriteSlopeCodeCount; slopeY++ {
			uniforms["RotationY"] = bitSpriteSlopeRadians(slopeY)

			for rotationFrame := range BitSpriteRotationFrames {
				uniforms["RotationZ"] = float32(rotationFrame*bitSpriteDegreesPerFrame) * (math.Pi / 180)
				frameImage.Clear()
				frameImage.DrawRectShader(pixelSize, pixelSize, shader, shaderOptions)

				slopeIndex := (int(slopeX)-1)*bitSpriteSlopeStates + int(slopeY) - 1
				frameIndex := slopeIndex*BitSpriteRotationFrames + rotationFrame
				column := frameIndex % sheetColumns
				row := frameIndex / sheetColumns
				drawOptions.GeoM.Reset()
				drawOptions.GeoM.Translate(float64(column*pixelSize), float64(row*pixelSize))
				sheetImage.DrawImage(frameImage, drawOptions)
			}
		}
	}

	var frames rockSpriteLookup
	for frameIndex := range frames {
		frames[frameIndex] = sheetImage.SubImage(spriteSheet.Rect(frameIndex)).(*ebiten.Image)
	}

	return &RockSpriteAtlas{
		Sprite: &render.Sprite{
			Image:       sheetImage,
			SpriteSheet: spriteSheet,
		},
		Frames: frames,
		Scales: initRockScales(),
	}
}
