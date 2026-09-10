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
	PresetBullet TimeControlPreset = "bullet"
	PresetBlitz  TimeControlPreset = "blitz"
	PresetRapid  TimeControlPreset = "rapid"
)

// Placeholder values, per your call to lock in real numbers later.
var presetTimeControls = map[TimeControlPreset]domain.TimeControl{
	PresetBullet: {Initial: 1 * time.Minute, Increment: 0},
	PresetBlitz:  {Initial: 5 * time.Minute, Increment: 3 * time.Second},
	PresetRapid:  {Initial: 10 * time.Minute, Increment: 0},
}

func timeControlForPreset(p TimeControlPreset) (domain.TimeControl, error) {
	tc, ok := presetTimeControls[p]
	if !ok {
		return domain.TimeControl{}, domain.ErrUnknownPreset
	}
	return tc, nil
}