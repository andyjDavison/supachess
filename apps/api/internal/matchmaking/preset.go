package matchmaking

import (
	"time"

	"api/internal/domain"
)

// TimeControlPreset identifies one of a small, fixed set of supported
// game modes. Clients select by name; the server owns the mapping to an
// actual domain.TimeControl — never trusting client-supplied durations
// directly, same principle applied everywhere else (passwords, player
// identity, move legality).
type TimeControlPreset string

const (
	Preset1_0   TimeControlPreset = "1+0"
	Preset1_1   TimeControlPreset = "1+1"
	Preset2_1   TimeControlPreset = "2+1"
	Preset3_0   TimeControlPreset = "3+0"
	Preset3_2   TimeControlPreset = "3+2"
	Preset5_0   TimeControlPreset = "5+0"
	Preset10_0  TimeControlPreset = "10+0"
	Preset10_5  TimeControlPreset = "10+5"
	Preset15_10 TimeControlPreset = "15+10"
)

var presetTimeControls = map[TimeControlPreset]domain.TimeControl{
	Preset1_0:   {Initial: 1 * time.Minute, Increment: 0},
	Preset1_1:   {Initial: 1 * time.Minute, Increment: 1 * time.Second},
	Preset2_1:   {Initial: 2 * time.Minute, Increment: 1 * time.Second},
	Preset3_0:   {Initial: 3 * time.Minute, Increment: 0},
	Preset3_2:   {Initial: 3 * time.Minute, Increment: 2 * time.Second},
	Preset5_0:   {Initial: 5 * time.Minute, Increment: 0},
	Preset10_0:  {Initial: 10 * time.Minute, Increment: 0},
	Preset10_5:  {Initial: 10 * time.Minute, Increment: 5 * time.Second},
	Preset15_10: {Initial: 15 * time.Minute, Increment: 10 * time.Second},
}

func timeControlForPreset(p TimeControlPreset) (domain.TimeControl, error) {
	tc, ok := presetTimeControls[p]
	if !ok {
		return domain.TimeControl{}, domain.ErrUnknownPreset
	}
	return tc, nil
}