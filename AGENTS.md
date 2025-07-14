# AGENTS.md - Development Guide for dice-will-roll

## Build/Test Commands
- **Build**: `go build` or `make build-all` (cross-platform builds)
- **Run**: `go run main.go` or `make run`
- **Test**: `go test ./...` (all packages)
- **Single test**: `go test ./dice -run TestName`
- **Test with verbose**: `go test -v ./dice`
- **Clean**: `make clean`

## Project Structure
- Go game using Ebiten v2 game engine
- Main packages: `dice/` (game logic), `render/` (graphics), root (main game loop)
- Shaders in `render/shaders/` using Kage shader language

## Code Style Guidelines
- **Imports**: Standard library first, then third-party, then local packages
- **Naming**: PascalCase for exported, camelCase for unexported, ALL_CAPS for constants
- **Types**: Use descriptive names (e.g., `HandRank`, `Action`, `Modifier`)
- **Error handling**: Use `log.Fatal()` for unrecoverable errors, return errors otherwise
- **Comments**: Document exported functions/types, use `//` for single line
- **Testing**: Use table-driven tests with descriptive test names
- **Constants**: Group related constants with `iota` for enums
- **Structs**: Embed types when appropriate, use pointer receivers for methods that modify state

## Key Conventions
- Game state managed through `Action` enum (ROLLING, DRAG, HELD, etc.)
- Dice logic separated into `dice/` package with comprehensive test coverage
- Use `generateDiceValues()` helper in tests to create test dice
- Shader uniforms use column-major matrix order for Kage compatibility