package shaders

import (
	_ "embed"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// types for Shader params can be
//
// Kage shader uniforms support int, float32, and []float32 types.
type ShaderParams map[string]any

// stores shaders, loads shaders, access to shaders

var (
	ErrShader    error = fmt.Errorf("shader could not be set")
	ErrNilShader error = fmt.Errorf("shader could not be found")
)

//go:embed die.kage
var kageDie []byte

func loadShader(kageShader []byte) *ebiten.Shader {
	shader, err := ebiten.NewShader(kageShader)
	if err != nil {
		panic(err)
	}
	return shader
}

type ShaderKey uint16

const (
	DieShaderKey ShaderKey = iota
)

func LoadShaders() map[ShaderKey]*ebiten.Shader {
	var shaders = map[ShaderKey]*ebiten.Shader{}

	shaders[DieShaderKey] = loadShader(kageDie)

	return shaders
}
