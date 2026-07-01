package music

import (
	"testing"
	"time"
)

type fakePlayer struct {
	position time.Duration
}

func (p *fakePlayer) Position() time.Duration {
	return p.position
}

func (p *fakePlayer) Play() {}

func (p *fakePlayer) SetPosition(position time.Duration) error {
	p.position = position
	return nil
}

func updateLaneIndexes(n *NowPlaying, count int) {
	for range count {
		n.UpdateLaneIndexes()
	}
}

func TestLoadEmbeddedManifest(t *testing.T) {
	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("LoadEmbeddedManifest() error = %v", err)
	}

	track, ok := manifest.Track("placeholder")
	if !ok {
		t.Fatal("placeholder track not found")
	}

	if track.File != "tracks/files/placeholder.mp3" {
		t.Fatalf("track.File = %q; want %q", track.File, "tracks/files/placeholder.mp3")
	}
	if len(track.Hooks) != 2 {
		t.Fatalf("len(track.Hooks) = %d; want 2", len(track.Hooks))
	}
}

func TestHookAdvancesLaneIndex(t *testing.T) {
	track := Track{
		Name: "test",
		File: "test.mp3",
		Hooks: [][]int64{
			{100, 200, 300},
			{150, 250, 350},
		},
	}
	player := &fakePlayer{position: 225 * time.Millisecond}
	nowPlaying := NewNowPlaying(track, player)

	if nowPlaying.LaneIndexes == nil {
		t.Fatal("LaneIndexes = nil")
	}
	updateLaneIndexes(nowPlaying, 2)

	if !nowPlaying.Hook(LandingLane) {
		t.Fatal("Hook(LandingLane) = false; want true")
	}
	if !nowPlaying.Hook(LandingLane) {
		t.Fatal("second Hook(LandingLane) = false; want true")
	}
	if nowPlaying.Hook(LandingLane) {
		t.Fatal("third Hook(LandingLane) = true; want false")
	}
	if !nowPlaying.Hook(LaneOne) {
		t.Fatal("Hook(LaneOne) = false; want true")
	}

	if got := nowPlaying.LaneIndexes[LandingLane]; got != 2 {
		t.Fatalf("LaneIndexes[LandingLane] = %d; want 2", got)
	}
	if got := nowPlaying.LaneIndexes[LaneOne]; got != 1 {
		t.Fatalf("LaneIndexes[LaneOne] = %d; want 1", got)
	}
	if got := nowPlaying.LaneIndexes[LaneTwo]; got != 0 {
		t.Fatalf("LaneIndexes[LaneTwo] = %d; want 0", got)
	}
	if track.Hooks[0][0] != 100 || track.Hooks[1][0] != 150 {
		t.Fatal("Hook mutated track hooks")
	}
}

func TestHooksIsAtomic(t *testing.T) {
	player := &fakePlayer{position: 250 * time.Millisecond}
	nowPlaying := NewNowPlaying(Track{
		Name: "test",
		File: "test.mp3",
		Hooks: [][]int64{
			{100, 200, 300},
		},
	}, player)
	updateLaneIndexes(nowPlaying, 2)

	if nowPlaying.Hooks(LandingLane, 3) {
		t.Fatal("Hooks(LandingLane, 3) = true; want false")
	}
	if got := nowPlaying.HookIndexes[LandingLane]; got != 0 {
		t.Fatalf("HookIndexes[LandingLane] = %d; want 0", got)
	}
	if !nowPlaying.Hooks(LandingLane, 2) {
		t.Fatal("Hooks(LandingLane, 2) = false; want true")
	}
	if got := nowPlaying.HookIndexes[LandingLane]; got != 2 {
		t.Fatalf("HookIndexes[LandingLane] = %d; want 2", got)
	}
}

func TestResetHooksIgnoresAlreadyPassedHooks(t *testing.T) {
	player := &fakePlayer{position: 250 * time.Millisecond}
	nowPlaying := NewNowPlaying(Track{
		Name: "test",
		File: "test.mp3",
		Hooks: [][]int64{
			{100, 200, 300},
		},
	}, player)
	nowPlaying.UpdateLaneIndexes()
	nowPlaying.ResetHooks(LandingLane)

	if nowPlaying.Hook(LandingLane) {
		t.Fatal("Hook(LandingLane) = true after ResetHooks; want false")
	}

	player.position = 325 * time.Millisecond
	nowPlaying.UpdateLaneIndexes()
	if !nowPlaying.Hook(LandingLane) {
		t.Fatal("Hook(LandingLane) = false after next hook; want true")
	}
}
