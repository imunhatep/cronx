package cronx

import (
	"testing"
	"time"
)

func TestParser_Parse_Valid(t *testing.T) {
	sched, err := ParseSpec("0 12 15 6 *", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sched.Minute != 1<<0 {
		t.Errorf("expected minute bitmask 1, got %d", sched.Minute)
	}
	if sched.Hour != 1<<12 {
		t.Errorf("expected hour bitmask 4096, got %d", sched.Hour)
	}
	if sched.Dom != 1<<14 {
		t.Errorf("expected dom bitmask 16384, got %d", sched.Dom)
	}
	if sched.Month != 1<<5 {
		t.Errorf("expected month bitmask 32, got %d", sched.Month)
	}
}

func TestParser_Parse_Invalid(t *testing.T) {
	_, err := ParseSpec("bad cron expression", true)
	if err == nil {
		t.Error("expected error for invalid cron expression")
	}
}

func TestParser_Parse_Timezone(t *testing.T) {
	// Timezone is not parsed in Parse, but you can test location handling in Cron
	loc, _ := time.LoadLocation("UTC")
	c, err := NewSchedule(Schedule{
		Minute: 1 << 0,
		Hour:   1 << 0,
		Dom:    1 << 0,
		Month:  1 << 0,
		Dow:    1 << 0,
	}, WithLocation(loc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.loc.String() != "UTC" {
		t.Errorf("expected UTC location, got %v", c.loc)
	}
}

func TestParser_Parse_OptionalSeconds(t *testing.T) {
	sched, err := ParseSpec("0 0 1 1 *", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sched.Second != 1<<0 {
		t.Errorf("expected second bitmask 1, got %d", sched.Second)
	}
}

func TestOption_WithSeconds(t *testing.T) {
	// Schedule with seconds, should fail without WithSeconds
	sched, err := ParseSpec("5 0 12 15 6 *", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = NewSchedule(sched)
	if err == nil {
		t.Error("expected error when seconds are set but WithSeconds is not provided")
	}

	// Should succeed with WithSeconds
	_, err = NewSchedule(sched, WithSeconds())
	if err != nil {
		t.Errorf("unexpected error with WithSeconds: %v", err)
	}
}

func TestSchedule_Matches(t *testing.T) {
	sched, err := ParseSpec("0 12 15 6 *", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should match: 2024-06-15 12:00
	matchTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.Local)
	if !sched.dayMatches(matchTime) || ((sched.Minute>>uint(matchTime.Minute()))&1) == 0 || ((sched.Hour>>uint(matchTime.Hour()))&1) == 0 || ((sched.Month>>uint(int(matchTime.Month())-1))&1) == 0 {
		t.Errorf("expected match for %v", matchTime)
	}

	// Should not match: wrong minute
	noMatchTime := time.Date(2024, 6, 15, 12, 1, 0, 0, time.Local)
	if sched.dayMatches(noMatchTime) && ((sched.Minute>>uint(noMatchTime.Minute()))&1) != 0 && ((sched.Hour>>uint(noMatchTime.Hour()))&1) != 0 && ((sched.Month>>uint(int(noMatchTime.Month())-1))&1) != 0 {
		t.Errorf("unexpected match for %v", noMatchTime)
	}

	// Should not match: wrong hour
	noMatchTime = time.Date(2024, 6, 15, 13, 0, 0, 0, time.Local)
	if sched.dayMatches(noMatchTime) && ((sched.Minute>>uint(noMatchTime.Minute()))&1) != 0 && ((sched.Hour>>uint(noMatchTime.Hour()))&1) != 0 && ((sched.Month>>uint(int(noMatchTime.Month())-1))&1) != 0 {
		t.Errorf("unexpected match for %v", noMatchTime)
	}

	// Should not match: wrong day
	noMatchTime = time.Date(2024, 6, 16, 12, 0, 0, 0, time.Local)
	if sched.dayMatches(noMatchTime) && ((sched.Minute>>uint(noMatchTime.Minute()))&1) != 0 && ((sched.Hour>>uint(noMatchTime.Hour()))&1) != 0 && ((sched.Month>>uint(int(noMatchTime.Month())-1))&1) != 0 {
		t.Errorf("unexpected match for %v", noMatchTime)
	}
}

// A 5-field spec must parse to the same masks whether or not seconds are
// enabled: the seconds default is prepended, so no field may shift position.
func TestParser_Parse_FiveFieldsNoFieldShift(t *testing.T) {
	const spec = "30 8 * * 1" // 08:30 on Mondays

	base, err := ParseSpec(spec, false)
	if err != nil {
		t.Fatalf("withSeconds=false: unexpected error: %v", err)
	}
	withSec, err := ParseSpec(spec, true)
	if err != nil {
		t.Fatalf("withSeconds=true: unexpected error: %v", err)
	}
	if withSec != base {
		t.Errorf("5-field spec %q parsed differently with seconds enabled:\n without: %+v\n with:    %+v", spec, base, withSec)
	}
	if base.Second != 1<<0 {
		t.Errorf("expected seconds to default to bit 0, got %#x", base.Second)
	}
	if base.Minute != 1<<30 {
		t.Errorf("expected minute 30, got %#x", base.Minute)
	}
	if base.Hour != 1<<8 {
		t.Errorf("expected hour 8, got %#x", base.Hour)
	}
	if base.Dow != 1<<1 {
		t.Errorf("expected dow Monday, got %#x", base.Dow)
	}

	// Named day-of-week must stay in the dow field rather than shifting into month.
	if _, err := ParseSpec("0 9-17/2 * * mon-fri", true); err != nil {
		t.Errorf("named dow in 5-field spec with seconds enabled: %v", err)
	}
}
