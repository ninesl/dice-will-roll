# SIMD Rock Layout

The updater uses Go's portable `simd` package. Runtime vector width determines
the batch size; the code contains no architecture-specific paths.

## Persistent State

Each rock occupies eight bytes across four contiguous `uint16` streams:

```text
PosX:    [velocityX:4][positionX:12]
PosY:    [velocityY:4][positionY:12]
Slope:   [slopeX:4][slopeY:4][slopeZ:4][sizeScore:4]
Animate: [stepY:3][stepX:3][stepZ:4][stepTick:4][permaSpin:1][spinAgain:1]
```

Drawing reads `PosX`, `PosY`, and `Slope`, totaling six bytes per rock. It does
not read `Animate`.

## Update Flow

One update batch loads one `Uint16s` vector from each stream. Position,
velocity, collision, animation, damping, and repacking all stay in 16-bit SIMD
lanes. There is no byte-lane transpose, 32-bit expansion, or scratch buffer.

Mouse hover first applies an amount-scale X/Y broad phase. Its maximum radius
is 180 pixels, which bounds `dx*dx + dy*dy` to 64,800 and permits an exact
16-bit circular narrow phase against each rock's hover radius. Mouse-down uses
the broad phase directly. Walls clamp and reflect in signed 16-bit lanes.

Each update checks mouse and projected wall collisions first. Collision lanes
keep their full assigned or reflected velocity; other finite-spin lanes damp
once. The resulting velocity then moves the rock, animation advances, collision
animation is scheduled, and the four fields are repacked.

The disabled-mouse path skips mouse distance, falloff, and impulse work. All
full and partial batches remain allocation-free.
