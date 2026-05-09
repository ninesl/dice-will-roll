package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// parse command line flag for shader path
	shaderPath := flag.String("shader", "background.kage", "Path to shader file (relative to shaders/ directory)")
	flag.Parse()

	// read shader file
	shaderBytes, err := os.ReadFile(filepath.Join("..", *shaderPath))
	if err != nil {
		panic(err)
	}

	// compile the shader
	shader, err := ebiten.NewShader(shaderBytes)
	if err != nil {
		panic(err)
	}

	// create game struct
	game := &Game{shader: shader}

	// configure window and run game
	ebiten.SetWindowTitle("Shader Test: " + *shaderPath)
	ebiten.SetFullscreen(true)
	err = ebiten.RunGame(game)
	if err != nil {
		panic(err)
	}
}

// Struct implementing the ebiten.Game interface.
type Game struct {
	shader   *ebiten.Shader
	vertices [4]ebiten.Vertex
	time     float32
}

// Use the actual screen size for layout
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// Update time for animated shaders.
func (g *Game) Update() error {
	g.time += 1.0 / 60.0 // increment time at 60fps
	return nil
}

// Core drawing function from where we call DrawTrianglesShader.
func (g *Game) Draw(screen *ebiten.Image) {
	// map the vertices to the target image
	bounds := screen.Bounds()
	g.vertices[0].DstX = float32(bounds.Min.X) // top-left
	g.vertices[0].DstY = float32(bounds.Min.Y) // top-left
	g.vertices[1].DstX = float32(bounds.Max.X) // top-right
	g.vertices[1].DstY = float32(bounds.Min.Y) // top-right
	g.vertices[2].DstX = float32(bounds.Min.X) // bottom-left
	g.vertices[2].DstY = float32(bounds.Max.Y) // bottom-left
	g.vertices[3].DstX = float32(bounds.Max.X) // bottom-right
	g.vertices[3].DstY = float32(bounds.Max.Y) // bottom-right

	// triangle shader options with uniforms
	shaderOpts := ebiten.DrawTrianglesShaderOptions{
		Uniforms: map[string]any{
			"Time":        g.time,
			"AspectRatio": float32(bounds.Max.X) / float32(bounds.Max.Y),
		},
	}

	// draw shader
	indices := []uint16{0, 1, 2, 2, 1, 3} // map vertices to triangles
	screen.DrawTrianglesShader(g.vertices[:], indices, g.shader, &shaderOpts)
}
