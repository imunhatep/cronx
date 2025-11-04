package cronx

import (
	"testing"
	"time"
)

func TestCron_Tick_MinuteResolution(t *testing.T) {
	cron, err := New("0 12 15 6 *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cron.Stop()

	// Set base time to just before the match
	base := time.Date(2024, 6, 15, 11, 59, 0, 0, time.Local)
	cron2, err := New("0 12 15 6 *", WithStartFrom(base))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cron2.Stop()

	next := <-cron2.C
	expected := time.Date(2024, 6, 15, 12, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestCron_Tick_SecondResolution(t *testing.T) {
	cron, err := New("5 0 12 15 6 *", WithSeconds(), WithStartFrom(time.Date(2024, 6, 15, 12, 0, 4, 0, time.Local)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cron.Stop()

	next := <-cron.C
	expected := time.Date(2024, 6, 15, 12, 0, 5, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestCron_Location(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	cron, err := New("0 0 1 1 *", WithLocation(loc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cron.Stop()

	if cron.loc.String() != "UTC" {
		t.Errorf("expected UTC location, got %v", cron.loc)
	}
}

func TestCron_Stop(t *testing.T) {
	cron, err := New("0 0 1 1 *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stopped := cron.Stop()
	if !stopped {
		t.Error("expected first Stop() to return true")
	}
	stoppedAgain := cron.Stop()
	if stoppedAgain {
		t.Error("expected second Stop() to return false")
	}
}
