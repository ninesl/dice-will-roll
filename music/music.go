package music

import (
	"time"
)

type Player interface {
	Position() time.Duration
	Play()
	SetPosition(time.Duration) error
}

type NowPlaying struct {
	Track       Track
	Player      Player
	LaneIndexes *[10]uint8
	HookIndexes *[10]uint8
	DurationMS  int64
}

func NewNowPlaying(track Track, player Player) *NowPlaying {
	n := &NowPlaying{
		Track:       track,
		Player:      player,
		LaneIndexes: &[10]uint8{},
		HookIndexes: &[10]uint8{},
	}

	return n
}

func (n *NowPlaying) MS() int64 {
	if n == nil || n.Player == nil {
		return 0
	}

	return n.Player.Position().Milliseconds() / 30 * 30
}

func (n *NowPlaying) Play() {
	if n == nil || n.Player == nil {
		return
	}

	n.Player.Play()
}

func (n *NowPlaying) Tick() {
	if n == nil || n.Player == nil {
		return
	}
	if n.DurationMS > 0 && n.MS() >= n.DurationMS {
		_ = n.Player.SetPosition(0)
	}
	n.UpdateLaneIndexes()
	if n.Player != nil {
		n.Player.Play()
	}
}

func (n *NowPlaying) UpdateLaneIndexes() {
	if n == nil || n.LaneIndexes == nil {
		return
	}

	ms := n.MS()
	laneCount := min(len(n.Track.Hooks), len(n.LaneIndexes))
	for lanePos := 0; lanePos < laneCount; lanePos++ {
		lane := n.Track.Hooks[lanePos]
		if len(lane) == 0 {
			continue
		}

		if int(n.LaneIndexes[lanePos]) >= len(lane) {
			n.LaneIndexes[lanePos] = 0
		}
		for ms >= lane[n.LaneIndexes[lanePos]] {
			n.LaneIndexes[lanePos]++
			if int(n.LaneIndexes[lanePos]) >= len(lane) {
				n.LaneIndexes[lanePos] = 0
				break
			}
		}
	}
}

func (n *NowPlaying) Hook(lane HookLane) bool {
	return n.Hooks(lane, 1)
}

func (n *NowPlaying) ResetHooks(lane HookLane) {
	if n == nil || n.LaneIndexes == nil || n.HookIndexes == nil {
		return
	}
	if lane < 0 || int(lane) >= len(n.Track.Hooks) || int(lane) >= len(n.LaneIndexes) {
		return
	}

	n.HookIndexes[lane] = n.LaneIndexes[lane]
}

func (n *NowPlaying) Hooks(lane HookLane, count uint8) bool {
	if count == 0 {
		count = 1
	}
	if n == nil || n.LaneIndexes == nil || n.HookIndexes == nil {
		return false
	}
	if lane < 0 || int(lane) >= len(n.Track.Hooks) || int(lane) >= len(n.LaneIndexes) {
		return false
	}

	i := int(lane)
	hooks := n.Track.Hooks[i]
	if len(hooks) == 0 {
		return false
	}

	available := n.LaneIndexes[i] - n.HookIndexes[i]
	if n.LaneIndexes[i] < n.HookIndexes[i] {
		available = uint8(len(hooks)) - n.HookIndexes[i] + n.LaneIndexes[i]
	}
	if available < count {
		return false
	}

	n.HookIndexes[i] += count
	if int(n.HookIndexes[i]) >= len(hooks) {
		n.HookIndexes[i] %= uint8(len(hooks))
	}
	return true
}

func (n *NowPlaying) UpcomingMS(lane HookLane) int64 {
	if n == nil || n.LaneIndexes == nil {
		return 0
	}
	if lane < 0 || int(lane) >= len(n.Track.Hooks) || int(lane) >= len(n.LaneIndexes) {
		return 0
	}

	i := int(lane)
	idx := int(n.LaneIndexes[i])
	if len(n.Track.Hooks[i]) == 0 || idx >= len(n.Track.Hooks[i]) {
		return 0
	}

	return n.Track.Hooks[i][idx]
}
