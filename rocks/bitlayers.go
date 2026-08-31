package rocks

import "github.com/hajimehoshi/ebiten/v2"

type rockLayerLayout struct {
	groupCount     int
	groupsPerLayer int
	layerCount     int
}

func calculateRockLayerLayout(rockCount int) rockLayerLayout {
	groupCount := (rockCount + RocksPerPackedVector - 1) / RocksPerPackedVector
	groupsPerLayer := 10
	for groupCount > groupsPerLayer*10 {
		groupsPerLayer *= 10
	}
	return rockLayerLayout{
		groupCount:     groupCount,
		groupsPerLayer: groupsPerLayer,
		layerCount:     (groupCount + groupsPerLayer - 1) / groupsPerLayer,
	}
}

func prepareRockLayerDirtyState(rocks *Rocks) rockLayerLayout {
	layout := calculateRockLayerLayout(rocks.Len())
	if rocks.groupsPerLayer != layout.groupsPerLayer || len(rocks.dirtyLayers) != layout.layerCount {
		rocks.groupsPerLayer = layout.groupsPerLayer
		rocks.dirtyLayers = make([]bool, layout.layerCount)
		rocks.redraw = true
	} else {
		clear(rocks.dirtyLayers)
	}
	rocks.anyDirtyLayer = false
	return layout
}

func (rocks *Rocks) markGroupLayerDirty(group int) {
	rocks.dirtyLayers[group/rocks.groupsPerLayer] = true
	rocks.anyDirtyLayer = true
}

// InvalidateLayers requests a complete rebuild after render state is changed outside UpdateRocks.
func (rocks *Rocks) InvalidateLayers() {
	rocks.redraw = true
}

type rockLayerState struct {
	initialized    bool
	amountScale    int
	colorScale     ebiten.ColorScale
	compositeMode  ebiten.CompositeMode
	blend          ebiten.Blend
	filter         ebiten.Filter
	disableMipmaps bool
}

// RockLayers owns the cached Z layers and the one image presented by Draw.
type RockLayers struct {
	images         []*ebiten.Image
	composite      *ebiten.Image
	width          int
	height         int
	groupsPerLayer int
	state          rockLayerState
}

func NewRockLayers(width, height int) *RockLayers {
	return &RockLayers{
		composite: ebiten.NewImage(width, height),
		width:     width,
		height:    height,
	}
}

func (layers *RockLayers) Composite() *ebiten.Image {
	return layers.composite
}

func (layers *RockLayers) ensureImages(layout rockLayerLayout) bool {
	if layers.groupsPerLayer == layout.groupsPerLayer && len(layers.images) == layout.layerCount {
		return false
	}
	for _, image := range layers.images {
		image.Deallocate()
	}
	layers.images = make([]*ebiten.Image, layout.layerCount)
	for i := range layers.images {
		layers.images[i] = ebiten.NewImage(layers.width, layers.height)
	}
	layers.groupsPerLayer = layout.groupsPerLayer
	return true
}

// Update rebuilds dirty cached images. It is intended to run from Game.Update.
func (layers *RockLayers) Update(
	rocks *Rocks,
	atlas *RockSpriteAtlas,
	amountScale int,
	drawOptions ebiten.DrawImageOptions,
) {
	layout := calculateRockLayerLayout(rocks.Len())
	rebuildAll := layers.ensureImages(layout) || rocks.redraw || !layers.state.initialized ||
		layers.state.amountScale != amountScale || layers.state.colorScale != drawOptions.ColorScale ||
		layers.state.compositeMode != drawOptions.CompositeMode || layers.state.blend != drawOptions.Blend ||
		layers.state.filter != drawOptions.Filter ||
		layers.state.disableMipmaps != drawOptions.DisableMipmaps
	if !rebuildAll && !rocks.anyDirtyLayer {
		return
	}

	for layer, image := range layers.images {
		if !rebuildAll && !rocks.dirtyLayers[layer] {
			continue
		}
		image.Clear()
		startGroup := layer * layout.groupsPerLayer
		endGroup := min(startGroup+layout.groupsPerLayer, layout.groupCount)
		start := startGroup * RocksPerPackedVector
		end := min(endGroup*RocksPerPackedVector, rocks.Len())
		drawRockRange(rocks, atlas, amountScale, image, drawOptions, start, end)
	}

	layers.composite.Clear()
	for _, image := range layers.images {
		layers.composite.DrawImage(image, nil)
	}
	rocks.redraw = false
	rocks.anyDirtyLayer = false
	clear(rocks.dirtyLayers)
	layers.state = rockLayerState{
		initialized:    true,
		amountScale:    amountScale,
		colorScale:     drawOptions.ColorScale,
		compositeMode:  drawOptions.CompositeMode,
		blend:          drawOptions.Blend,
		filter:         drawOptions.Filter,
		disableMipmaps: drawOptions.DisableMipmaps,
	}
}
