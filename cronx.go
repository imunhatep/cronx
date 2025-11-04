package cronx

import (
	"sync"
	"time"
)

// Cron emits matching instants over the channel C.
// There is no Reset(); Stop() the cron and create a new one for a different schedule.
type Cron struct {
	C <-chan time.Time

	// internal
	c           chan time.Time
	loc         *time.Location
	buf         int
	from        time.Time
	withSeconds bool
	sched       Schedule

	stop    chan struct{}
	mu      sync.Mutex
	stopped bool
}

// New constructs a Cron from the spec and options.
// IMPORTANT: Options are applied first to discover WithSeconds(), then the spec is parsed accordingly.
func New(spec string, opts ...Option) (*Cron, error) {
	// Pre-apply options on a temp to read withSeconds (and any other setup).
	tmp := &Cron{loc: time.Local, buf: 1}
	for _, opt := range opts {
		opt(tmp)
	}

	s, err := ParseSpec(spec, tmp.withSeconds)
	if err != nil {
		return nil, err
	}
	return NewSchedule(s, opts...)
}

// NewSchedule constructs a Cron from a pre-parsed Schedule and options.
// If WithSeconds() is not set, the engine will still run at minute resolution.
// (Use ParseSpec to ensure parsing rules consistent with your options.)
func NewSchedule(s Schedule, opts ...Option) (*Cron, error) {
	c := &Cron{
		loc:   time.Local,
		buf:   1,
		sched: s,
		stop:  make(chan struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}

	// Normalize start time after options (option order safe).
	if c.from.IsZero() {
		c.from = time.Now().In(c.loc)
	} else {
		c.from = c.from.In(c.loc)
	}

	// Channel
	ch := make(chan time.Time, c.buf)
	c.c = ch
	c.C = ch

	go c.run()
	return c, nil
}

func (c *Cron) run() {
	now := c.from.In(c.loc)
	next := c.sched.NextFrom(now, c.withSeconds)
	timer := time.NewTimer(time.Until(next))
	defer func() {
		stopTimer(timer)
		// Do not close c.c (match time.Timer/time.Ticker semantics).
	}()

	for {
		select {
		case <-timer.C:
			c.c <- next
			now = next
			next = c.sched.NextFrom(now, c.withSeconds)
			stopTimer(timer)
			timer = time.NewTimer(time.Until(next))
		case <-c.stop:
			return
		}
	}
}

func stopTimer(t *time.Timer) {
	if t == nil {
		return
	}
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

// Stop terminates the cron loop. Safe to call multiple times.
// Returns true if this call stopped a running cron; false if it was already stopped.
func (c *Cron) Stop() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return false
	}
	c.stopped = true
	close(c.stop)
	return true
}

// Next returns the next matching instant from 'from' using this Cron's resolution.
func (c *Cron) Next(from time.Time) time.Time {
	return c.sched.NextFrom(from.In(c.loc), c.withSeconds)
}
