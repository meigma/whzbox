package clock_test

import (
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/core/clock"
)

func TestReal_Now(t *testing.T) {
	var c clock.Clock = clock.Real{}
	before := time.Now()
	got := c.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("Real.Now = %v, expected between %v and %v", got, before, after)
	}
}

func TestFake_Now(t *testing.T) {
	fixed := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	f := &clock.Fake{T: fixed}

	var c clock.Clock = f
	if got := c.Now(); !got.Equal(fixed) {
		t.Errorf("Fake.Now = %v, want %v", got, fixed)
	}
}

func TestFake_Advance(t *testing.T) {
	start := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	f := &clock.Fake{T: start}

	f.Advance(5 * time.Minute)
	want := start.Add(5 * time.Minute)
	if got := f.Now(); !got.Equal(want) {
		t.Errorf("after Advance(5m), Now = %v, want %v", got, want)
	}

	f.Advance(-10 * time.Minute)
	want = start.Add(-5 * time.Minute)
	if got := f.Now(); !got.Equal(want) {
		t.Errorf("after Advance(-10m), Now = %v, want %v", got, want)
	}
}
