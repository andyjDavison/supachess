package matchmaking

import (
	"errors"
	"testing"
	"time"

	"api/internal/domain"
)

// Pinned against the preset list in PLAN.md. A client sends only the name;
// the server owns the durations, so a wrong mapping here silently changes
// what people are playing.
func TestTimeControlForPreset(t *testing.T) {
	want := map[TimeControlPreset]domain.TimeControl{
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

	if len(presetTimeControls) != len(want) {
		t.Errorf("there are %d presets but %d expectations - update this test alongside preset.go",
			len(presetTimeControls), len(want))
	}

	for preset, expected := range want {
		got, err := TimeControlForPreset(preset)
		if err != nil {
			t.Errorf("TimeControlForPreset(%q) returned error %v", preset, err)
			continue
		}
		if got != expected {
			t.Errorf("preset %q = %+v, want %+v", preset, got, expected)
		}
	}
}

func TestTimeControlForUnknownPreset(t *testing.T) {
	_, err := TimeControlForPreset("9+9")
	if !errors.Is(err, domain.ErrUnknownPreset) {
		t.Errorf("unknown preset gave %v, want ErrUnknownPreset", err)
	}
}
