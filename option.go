package cronx

import "time"

//
// ======================= RUNTIME =======================
//

// Option configures Cron.
type Option func(*Cron)

// WithLocation sets the timezone (default: time.Local).
func WithLocation(loc *time.Location) Option {
	return func(c *Cron) { c.loc = loc }
}

// WithBuffered sets the buffer size of C (default: 1).
func WithBuffered(n int) Option {
	return func(c *Cron) {
		if n < 0 {
			n = 0
		}
		c.buf = n
	}
}

// WithStartFrom overrides the base time for computing the first tick.
func WithStartFrom(from time.Time) Option {
	return func(c *Cron) { c.from = from }
}

// WithSeconds enables 6-field cron and second-level resolution.
// Parsing is option-driven:
//   - Without WithSeconds(): the spec must have 5 fields; sec=0.
//   - With WithSeconds(): the spec may have 5 fields (sec="*") or 6 fields (leading "sec").
func WithSeconds() Option {
	return func(c *Cron) { c.withSeconds = true }
}
