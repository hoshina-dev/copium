package clock_test

import (
	"testing"
	"time"

	"github.com/hoshina-dev/copium/internal/clock"
)

func TestSystem_NowIsRecent(t *testing.T) {
	c := clock.System{}
	before := time.Now()
	got := c.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("System.Now()=%v not within [%v,%v]", got, before, after)
	}
}

func TestFake_AdvanceAndSet(t *testing.T) {
	start := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	f := clock.NewFake(start)
	if !f.Now().Equal(start) {
		t.Fatalf("Now=%v want %v", f.Now(), start)
	}
	f.Advance(2 * time.Hour)
	want := start.Add(2 * time.Hour)
	if !f.Now().Equal(want) {
		t.Fatalf("after Advance Now=%v want %v", f.Now(), want)
	}
	other := time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
	f.Set(other)
	if !f.Now().Equal(other) {
		t.Fatalf("after Set Now=%v want %v", f.Now(), other)
	}
}

func TestFake_ConcurrentSafe(t *testing.T) {
	f := clock.NewFake(time.Now())
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			f.Advance(time.Millisecond)
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		_ = f.Now()
	}
	<-done
}
