package music

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

type Manifest struct {
	Tracks []Track `json:"tracks"`
}

type Track struct {
	Name  string    `json:"name"`
	File  string    `json:"file"`
	Hooks [][]int64 `json:"hooks"`
}

func LoadManifest(fsys fs.FS, path string) (Manifest, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}

	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func (m Manifest) Track(name string) (Track, bool) {
	for _, track := range m.Tracks {
		if track.Name == name {
			return track, true
		}
	}

	return Track{}, false
}

func validateManifest(manifest Manifest) error {
	if len(manifest.Tracks) == 0 {
		return fmt.Errorf("music manifest has no tracks")
	}

	for i, track := range manifest.Tracks {
		if track.Name == "" {
			return fmt.Errorf("track %d has no name", i)
		}
		if track.File == "" {
			return fmt.Errorf("track %q has no file", track.Name)
		}
		if len(track.Hooks) == 0 {
			return fmt.Errorf("track %q has no hooks", track.Name)
		}

		for lane, hooks := range track.Hooks {
			var previous int64 = -1
			for hookIndex, hook := range hooks {
				if hook < 0 {
					return fmt.Errorf("track %q hook lane %d index %d is negative", track.Name, lane, hookIndex)
				}
				if hook < previous {
					return fmt.Errorf("track %q hook lane %d is not ascending", track.Name, lane)
				}
				previous = hook
			}
		}
	}

	return nil
}
