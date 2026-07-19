package shaders

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
)

// types for Shader params can be
//
// Kage shader uniforms support int, float32, and []float32 types.
type ShaderParams map[string]interface{}

// stores shaders, loads shaders, access to shaders
var (
	ErrShader    error = fmt.Errorf("shader could not be set")
	ErrNilShader error = fmt.Errorf("shader could not be found")
)

//go:embed kages
var kageShaders embed.FS

var (
	RocksShaders = subFS(kageShaders, "kages/rocks")
	FXShaders    = subFS(kageShaders, "kages/fx")
	DiceShaders  = subFS(kageShaders, "kages/dice")
)

func subFS(shaderFS embed.FS, dir string) fs.FS {
	categoryFS, err := fs.Sub(shaderFS, dir)
	if err != nil {
		panic(err)
	}
	return categoryFS
}

func loadShader(shaderFS fs.FS, path string) *ebiten.Shader {
	kageShader, err := fs.ReadFile(shaderFS, path)
	if err != nil {
		panic(err)
	}

	shader, err := ebiten.NewShader(kageShader)
	if err != nil {
		panic(err)
	}
	return shader
}

type ShaderKey uint16

const (
	DieShaderKey ShaderKey = iota
	RocksShaderKey
	ColorFilterShaderKey
	FXAAShaderKey
	ExplosionShaderKey
	BackgroundShaderKey
	ColorShaderKey
)

func LoadShaders() map[ShaderKey]*ebiten.Shader {
	var shaders = map[ShaderKey]*ebiten.Shader{}

	shaders[ExplosionShaderKey] = loadShader(RocksShaders, "explosion.kage")
	shaders[RocksShaderKey] = loadShader(RocksShaders, "moon_rock.kage")

	shaders[DieShaderKey] = loadShader(DiceShaders, "die.kage")

	shaders[ColorFilterShaderKey] = loadShader(FXShaders, "color_filter.kage")
	shaders[FXAAShaderKey] = loadShader(FXShaders, "fxaa.kage")
	shaders[BackgroundShaderKey] = loadShader(FXShaders, "background.kage")
	shaders[ColorShaderKey] = loadShader(FXShaders, "color.kage")

	return shaders
}
