package rating

import (
	"testing"
	"time"

	"api/internal/domain"
)

func TestCategoryForEveryPreset(t *testing.T) {
	cases := []struct {
		initial, increment time.Duration
		want               domain.RatingCategory
	}{
		{1 * time.Minute, 0, domain.CategoryBullet},
		{1 * time.Minute, 1 * time.Second, domain.CategoryBullet},
		{2 * time.Minute, 1 * time.Second, domain.CategoryBullet},
		{3 * time.Minute, 0, domain.CategoryBlitz},
		{3 * time.Minute, 2 * time.Second, domain.CategoryBlitz},
		{5 * time.Minute, 0, domain.CategoryBlitz},
		{10 * time.Minute, 0, domain.CategoryRapid},
		{10 * time.Minute, 5 * time.Second, domain.CategoryRapid},
		{15 * time.Minute, 10 * time.Second, domain.CategoryRapid},
	}
	for _, c := range cases {
		tc := domain.TimeControl{Initial: c.initial, Increment: c.increment}
		if got := CategoryFor(tc); got != c.want {
			t.Errorf("CategoryFor(%v+%v) = %s, want %s", c.initial, c.increment, got, c.want)
		}
	}
}

func TestKFactorDropsAfterTheProvisionalGames(t *testing.T) {
	if got := KFactor(0); got != 40 {
		t.Errorf("KFactor(0) = %d, want 40", got)
	}
	if got := KFactor(29); got != 40 {
		t.Errorf("KFactor(29) = %d, want 40", got)
	}
	if got := KFactor(30); got != 20 {
		t.Errorf("KFactor(30) = %d, want 20", got)
	}
}

func TestDelta(t *testing.T) {
	cases := []struct {
		name           string
		rating, opp, k int32
		score          float64
		want           int32
	}{
		{"equal, win", 1200, 1200, 40, 1, 20},
		{"equal, draw", 1200, 1200, 40, 0.5, 0},
		{"equal, loss", 1200, 1200, 40, 0, -20},
		// 400 points up: expected score 10/11.
		{"favourite wins", 1600, 1200, 20, 1, 2},
		{"favourite draws", 1600, 1200, 20, 0.5, -8},
		{"favourite loses", 1600, 1200, 20, 0, -18},
		{"underdog wins", 1200, 1600, 20, 1, 18},
		{"underdog draws", 1200, 1600, 20, 0.5, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Delta(c.rating, c.opp, c.k, c.score); got != c.want {
				t.Errorf("Delta = %d, want %d", got, c.want)
			}
		})
	}
}

func TestDeltaNeverGoesBelowTheFloor(t *testing.T) {
	if got := Delta(110, 110, 40, 0); got != -10 {
		t.Errorf("Delta at 110 = %d, want -10 (stops at %d)", got, Floor)
	}
	if got := Delta(100, 100, 40, 0); got != 0 {
		t.Errorf("Delta at the floor = %d, want 0", got)
	}
	// Winning near the floor is unaffected.
	if got := Delta(100, 100, 40, 1); got != 20 {
		t.Errorf("Delta winning at the floor = %d, want 20", got)
	}
}

func TestStakesFor(t *testing.T) {
	got := StakesFor(1200, 1600, 20)
	want := Stakes{Win: 18, Draw: 8, Loss: -2}
	if got != want {
		t.Errorf("StakesFor = %+v, want %+v", got, want)
	}
}
