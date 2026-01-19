# AGENTS.md - Coding Agent Guidelines

This document provides essential information for coding agents working on the dice-will-roll project, a 2D dice game built with Ebiten.

## Build, Lint, and Test Commands

### Building
```bash
go build                                    # Build main executable
GOOS=linux GOARCH=amd64 go build            # Cross-compile for Linux
GOOS=windows GOARCH=amd64 go build          # Cross-compile for Windows
```

### Testing
```bash
go test ./...                               # Run all tests
go test ./dice                              # Run tests for specific package
go test -run TestDetermineHandRank ./dice   # Run a single test
go test -v ./dice                           # Verbose output
go test -race ./dice                        # With race detection
```

### Linting and Code Quality
```bash
gofmt -d .                                  # Check formatting (diff)
gofmt -w .                                  # Apply formatting
go vet ./...                                # Check for issues
go mod tidy                                 # Clean dependencies
```

## Code Style Guidelines

### Imports
```go
import (
    "sort"
    "slices"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/ninesl/dice-will-roll/dice"
    "github.com/ninesl/dice-will-roll/render"
)
```
- Standard library first, blank line, then third-party/internal packages
- Use `_ "embed"` for embedding files (see `render/shaders/load.go`)

### Naming Conventions
- **Variables/Functions**: `camelCase` for unexported, `PascalCase` for exported
- **Constants**: `ALL_CAPS` with iota for enums
- **Structs**: `PascalCase` fields, group related fields together
- Prefer descriptive names: `diceDataBuffer` over `buf`

```go
const (
    NONE Action = iota
    ROLLING
    DRAG
    HELD
)
```

### Error Handling
```go
s, err := shaders.LoadColorFilter()
if err != nil {
    log.Fatal(err)  // For initialization errors
}
// Or return: fmt.Errorf("failed to load shader: %w", err)
```
- Check errors immediately after they occur
- Use `log.Fatal()` for startup errors, return errors otherwise
- Use `panic()` only for truly unrecoverable shader/graphics errors

### Function Structure
- Keep functions under 50 lines when possible
- Use early returns to reduce nesting
- Group related operations together

### Testing Patterns
```go
func TestDetermineHandRank(t *testing.T) {
    tests := []struct {
        name       string
        diceValues []int
        maxPips    int
        expected   HandRank
    }{
        {"HIGH_DIE with single die", []int{6}, 6, HIGH_DIE},
        {"ONE_PAIR with two identical", []int{3, 3}, 6, ONE_PAIR},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            dice := generateDiceValues(tc.diceValues, tc.maxPips)
            got := DetermineHandRank(dice)
            if got != tc.expected {
                t.Errorf("DetermineHandRank(%v) = %s; want %s",
                    tc.diceValues, got.String(), tc.expected.String())
            }
        })
    }
}
```
- Use table-driven tests with descriptive names
- Use `t.Helper()` for helper functions

## Project Structure
```
dice-will-roll/
├── main.go              # Entry point, Game struct, game loop
├── die.go               # Die struct and mechanics
├── controls.go          # Input handling
├── update.go            # Game state updates
├── draw.go              # Rendering logic
├── ui.go                # UI components
├── levels.go            # Level management
├── dice/                # Hand evaluation (has tests)
│   ├── dice.go, hand.go, score.go
│   └── *_test.go
├── render/              # Rendering system
│   ├── types.go, colors.go, dice.go, zones.go
│   └── shaders/         # Kage shaders (.kage files)
├── rocks/               # Rock physics and rendering
│   ├── rocks.go, movement.go, collisions.go
│   └── rocks_renderer.go
└── assets/              # Game assets (images)
```

## Architecture Notes

- **Ebiten v2**: 2D game framework, implements `Game` interface (Update/Draw/Layout)
- **Kage Shaders**: Custom shaders in `render/shaders/*.kage`, loaded via `//go:embed`
- **State Machine**: `Action` type controls game modes (ROLLING, DRAG, HELD, SCORING)
- **Rendering**: Hybrid SDF-based system for rocks, shader-based for dice

### Performance Considerations
- Pre-allocate slices/buffers (see `diceDataBuffer`, `diceVelocityBuffer`)
- Use `float32` for graphics, minimize allocations in Update/Draw loops
- Rock system uses sprite batching with interleave layers

### Key Types
```go
type Game struct { ... }      // Main game state (main.go)
type Die struct { ... }       // Player die (die.go)
type Action uint16            // Game state enum (main.go)
type HandRank uint8           // Poker-style hand rankings (dice/hand.go)
```

## Commit Message Style
```
feat: add new dice hand evaluation algorithm
fix: resolve collision detection bug in rock physics
refactor: simplify die rendering pipeline
test: add comprehensive hand ranking tests
docs: update AGENTS.md with new guidelines
```
