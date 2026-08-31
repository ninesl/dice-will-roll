# SIMD Rock Layout

The updater uses Go's portable `simd` package. Runtime vector width determines
the batch size; the code contains no architecture-specific paths.

## Persistent State

Each rock occupies eight bytes across four contiguous `uint16` streams:

```text
PosX:    [velocityX:4][positionX:12]
PosY:    [velocityY:4][positionY:12]
Slope:   [slopeX:4][slopeY:4][slopeZ:4][sizeScore:4]
Stepping: [stepY:3][stepX:3][stepZ:4][stepTick:4][ForceStepping:1][Stepping:1]
```

Drawing reads `PosX`, `PosY`, and `Slope`, totaling six bytes per rock. It does
not read `Animate`.

## Update Flow

One update batch loads one `Uint16s` vector from each stream. Position,
velocity, collision, animation, damping, and repacking all stay in 16-bit SIMD
lanes. There is no byte-lane transpose, 32-bit expansion, or scratch buffer.
When a group is active, stepping executes as SIMD and inactive lanes are
selected by the packed `Stepping` state mask.

Before the full update, each group loads only `PosX`, `PosY`, and `Stepping`
and builds a SIMD activity vector from velocity bits, stepping state, and the
mouse broad phase. Groups with no active lanes skip the `Slope` load, full
update, and all stores. The existing `Stepping` slice temporarily receives the
activity vector for its portable horizontal reduction; active groups overwrite
it with the completed update and inactive groups only write the same zero state.

Collision and hover radii use amount-scale-specific integer affine formulas.
They exactly reproduce all atlas radius values for size scores 1 through 15,
without scanning 15 broadcast lookup vectors for each batch.

Mouse hover first applies an amount-scale X/Y broad phase. Its maximum radius
is 90 pixels, which bounds `dx*dx + dy*dy` to 16,200 and permits an exact
16-bit circular narrow phase against each rock's hover radius. Mouse-down uses
the broad phase directly. Walls clamp and reflect in signed 16-bit lanes.

Mouse force components use independent seven-band axis distances. Left pushes
away with stronger force farther from the pointer, right and hover push away
with stronger force near the pointer, and both buttons pull toward it with the
near-pointer magnitude. Button-hit lanes retain their assigned direction at
walls while the wall continues driving stepping animation.

Every hover or button hit queues `Stepping` independently from wall hits.
Repeated contact does not reset active `stepZ`; it only keeps the next full
step queued.

Each update checks mouse and projected wall collisions first. Collision lanes
keep their full assigned or reflected velocity; other finite-step lanes damp
once. The resulting velocity then moves the rock, animation advances, collision
animation is scheduled, and the four fields are repacked.

The disabled-mouse path skips mouse distance, falloff, and impulse work. All
full and partial batches remain allocation-free.
