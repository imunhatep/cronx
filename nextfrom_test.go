package cronx

import (
	"testing"
	"time"
)

// nextFrom parses spec and returns the next instant after from, in UTC.
func nextFrom(t *testing.T, spec string, withSeconds bool, from time.Time) time.Time {
	t.Helper()

	sched, err := ParseSpec(spec, withSeconds)
	if err != nil {
		t.Fatalf("ParseSpec(%q): %v", spec, err)
	}

	return sched.NextFrom(from, withSeconds)
}

// A schedule whose minute is 0 used to be pushed a full hour late: advancing to the
// target hour left the minute at 1, which then wrapped into the following hour.
func TestNextFrom_MinuteResolution(t *testing.T) {
	// 2026-08-26 is a Wednesday
	wed0915 := time.Date(2026, 8, 26, 9, 15, 0, 0, time.UTC)

	tests := []struct {
		name string
		spec string
		from time.Time
		want time.Time
	}{
		{
			name: "weekly on monday at 07:00",
			spec: "0 7 * * 1",
			from: wed0915,
			want: time.Date(2026, 8, 31, 7, 0, 0, 0, time.UTC),
		},
		{
			name: "daily at midnight",
			spec: "0 0 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "daily at 07:00",
			spec: "0 7 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 27, 7, 0, 0, 0, time.UTC),
		},
		{
			name: "daily at 07:30",
			spec: "30 7 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 27, 7, 30, 0, 0, time.UTC),
		},
		{
			name: "same minute tomorrow, not the next hour today",
			spec: "15 9 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 27, 9, 15, 0, 0, time.UTC),
		},
		{
			name: "later today",
			spec: "0 12 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "every minute",
			spec: "* * * * *",
			from: wed0915,
			want: time.Date(2026, 8, 26, 9, 16, 0, 0, time.UTC),
		},
		{
			name: "next allowed minute within the same hour",
			spec: "20,40 9 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 26, 9, 20, 0, 0, time.UTC),
		},
		{
			name: "minute wraps into the next allowed hour",
			spec: "10 9,14 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 26, 14, 10, 0, 0, time.UTC),
		},
		{
			name: "rolls into the next month",
			spec: "0 0 1 * *",
			from: wed0915,
			want: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "rolls into the next year",
			spec: "0 0 1 1 *",
			from: wed0915,
			want: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "strictly after t, never t itself",
			spec: "0 7 * * *",
			from: time.Date(2026, 8, 26, 7, 0, 0, 0, time.UTC),
			want: time.Date(2026, 8, 27, 7, 0, 0, 0, time.UTC),
		},
		{
			name: "leap day",
			spec: "0 0 29 2 *",
			from: time.Date(2026, 8, 26, 9, 15, 0, 0, time.UTC),
			want: time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nextFrom(t, tc.spec, false, tc.from)
			if !got.Equal(tc.want) {
				t.Errorf("NextFrom(%q, %s) = %s, want %s",
					tc.spec, tc.from.Format(time.RFC3339), got.Format(time.RFC3339), tc.want.Format(time.RFC3339))
			}
		})
	}
}

// The same off-by-one applied at second resolution: advancing a coarser field left the
// second at 1, so a schedule firing on second 0 was carried into the next minute.
func TestNextFrom_SecondResolution(t *testing.T) {
	wed0915 := time.Date(2026, 8, 26, 9, 15, 30, 0, time.UTC)

	tests := []struct {
		name string
		spec string
		from time.Time
		want time.Time
	}{
		{
			name: "daily at 07:00:00",
			spec: "0 0 7 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 27, 7, 0, 0, 0, time.UTC),
		},
		{
			name: "every second",
			spec: "* * * * * *",
			from: wed0915,
			want: time.Date(2026, 8, 26, 9, 15, 31, 0, time.UTC),
		},
		{
			name: "next allowed second in the same minute",
			spec: "45 15 9 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 26, 9, 15, 45, 0, time.UTC),
		},
		{
			name: "second wraps into the next allowed minute",
			spec: "5 15,20 9 * * *",
			from: wed0915,
			want: time.Date(2026, 8, 26, 9, 20, 5, 0, time.UTC),
		},
		{
			name: "weekly on monday at 07:00:00",
			spec: "0 0 7 * * 1",
			from: wed0915,
			want: time.Date(2026, 8, 31, 7, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nextFrom(t, tc.spec, true, tc.from)
			if !got.Equal(tc.want) {
				t.Errorf("NextFrom(%q, %s) = %s, want %s",
					tc.spec, tc.from.Format(time.RFC3339), got.Format(time.RFC3339), tc.want.Format(time.RFC3339))
			}
		})
	}
}

// An impossible date must terminate the search instead of looping forever.
func TestNextFrom_ImpossibleScheduleReturnsZero(t *testing.T) {
	from := time.Date(2026, 8, 26, 9, 15, 0, 0, time.UTC)

	for _, spec := range []string{"0 0 30 2 *", "0 0 31 2 *"} {
		t.Run(spec, func(t *testing.T) {
			done := make(chan time.Time, 1)
			go func() { done <- nextFrom(t, spec, false, from) }()

			select {
			case got := <-done:
				if !got.IsZero() {
					t.Errorf("NextFrom(%q) = %s, want zero time", spec, got.Format(time.RFC3339))
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("NextFrom(%q) did not terminate", spec)
			}
		})
	}
}

// A Cron built on an impossible schedule must not spin, and must never deliver a tick.
func TestCron_ImpossibleScheduleDoesNotFire(t *testing.T) {
	c, err := New("0 0 30 2 *", WithLocation(time.UTC))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Stop()

	select {
	case tick := <-c.C:
		t.Fatalf("unexpected tick at %s", tick.Format(time.RFC3339))
	case <-time.After(200 * time.Millisecond):
	}
}

// Successive calls must advance, and every returned instant must satisfy the schedule.
func TestNextFrom_IsMonotonicAndMatches(t *testing.T) {
	specs := []string{"0 7 * * 1", "0 0 * * *", "*/15 * * * *", "0 7 * * 2", "30 6 1 * *"}

	for _, spec := range specs {
		t.Run(spec, func(t *testing.T) {
			sched, err := ParseSpec(spec, false)
			if err != nil {
				t.Fatalf("ParseSpec: %v", err)
			}

			ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			for i := 0; i < 200; i++ {
				next := sched.NextFrom(ts, false)
				if next.IsZero() {
					t.Fatalf("iteration %d: schedule stopped matching", i)
				}
				if !next.After(ts) {
					t.Fatalf("iteration %d: %s is not after %s", i, next, ts)
				}
				if next.Second() != 0 {
					t.Errorf("iteration %d: %s has non-zero seconds", i, next)
				}
				if ((sched.Minute>>uint(next.Minute()))&1) == 0 || ((sched.Hour>>uint(next.Hour()))&1) == 0 {
					t.Fatalf("iteration %d: %s does not match %q", i, next.Format(time.RFC3339), spec)
				}
				if !sched.dayMatches(next) {
					t.Fatalf("iteration %d: %s day does not match %q", i, next.Format(time.RFC3339), spec)
				}
				ts = next
			}
		})
	}
}
