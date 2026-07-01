package music

import "embed"

//go:embed tracks/tracks.json tracks/files/iommiwatts.mp3 tracks/json/track_iommiwatts.json
var TracksFS embed.FS

const TracksManifestPath = "tracks/tracks.json"

func LoadEmbeddedManifest() (Manifest, error) {
	return LoadManifest(TracksFS, TracksManifestPath)
}
