package cronx

import (
	"errors"
	"fmt"
	"math/bits"
	"strings"
	"time"
)

//
// ======================= SCHEDULE & PARSING =======================
//

// Schedule is a parsed cron mask supporting 5- or 6-field syntax depending on config.
// If WithSeconds() is enabled:
//   - 6 fields: "sec min hour dom mon dow"
//   - 5 fields: "min hour dom mon dow"  (seconds default to "*")
//
// If WithSeconds() is disabled:
//   - exactly 5 fields: "min hour dom mon dow" (seconds fixed to 0)
//
// Bit field domains:
//
//	Second:  0..59
//	Minute:  0..59
//	Hour: 0..23
//	Dom:  1..31
//	Month:  1..12
//	Dow:  0..6 (Sun=0; raw "7" is normalized to 0)
type Schedule struct {
	Second uint64
	Minute uint64
	Hour   uint64
	Dom    uint64
	Month  uint64
	Dow    uint64

	domStar bool
	dowStar bool
}

func mustMask(min, max int) uint64 {
	var m uint64
	for v := min; v <= max; v++ {
		m |= 1 << uint(v-min)
	}
	return m
}

// ParseSpec parses a cron spec according to whether seconds are enabled.
//   - withSeconds=false: expect exactly 5 fields (sec fixed to 0)
//   - withSeconds=true:
//   - 6 fields: "sec min hour dom mon dow"
//   - 5 fields: "min hour dom mon dow" (seconds default to "*")

func ParseSpec(spec string, withSeconds bool) (Schedule, error) {
	fields := strings.Fields(spec)

	if !withSeconds && len(fields) != 5 {
		return Schedule{}, fmt.Errorf("expected 5 fields without seconds, got %d", len(fields))
	}

	// transform 5-field to 6-field by appending "*" for seconds
	if withSeconds && len(fields) == 5 {
		fields = append(fields, "*")
	}

	// transform 5-field to 6-field by prepending "*" for seconds
	if !withSeconds && len(fields) == 5 {
		fields = append([]string{"*"}, fields...)
	}

	if len(fields) != 6 {
		return Schedule{}, fmt.Errorf("expected 5 or 6 fields with seconds enabled, got %d", len(fields))
	}

	return parseWithSeconds(fields)
}

// parseWithSeconds parses: "sec min hour dom mon dow"
func parseWithSeconds(f []string) (Schedule, error) {
	var s Schedule
	var err error
	var star bool

	s.Second, _, err = parseField(f[0], 0, 59, nil)
	if err != nil {
		return Schedule{}, fmt.Errorf("sec: %w", err)
	}
	s.Minute, _, err = parseField(f[1], 0, 59, nil)
	if err != nil {
		return Schedule{}, fmt.Errorf("minute: %w", err)
	}
	s.Hour, _, err = parseField(f[2], 0, 23, nil)
	if err != nil {
		return Schedule{}, fmt.Errorf("hour: %w", err)
	}
	s.Dom, star, err = parseField(f[3], 1, 31, nil)
	if err != nil {
		return Schedule{}, fmt.Errorf("dom: %w", err)
	}
	s.domStar = star

	monNames := map[string]int{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
		"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}
	s.Month, _, err = parseField(f[4], 1, 12, monNames)
	if err != nil {
		return Schedule{}, fmt.Errorf("month: %w", err)
	}

	dowNames := map[string]int{
		"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
	}
	raw := f[5]
	if strings.Contains(raw, "7") {
		raw = strings.ReplaceAll(raw, "7", "0")
	}
	s.Dow, star, err = parseField(raw, 0, 6, dowNames)
	if err != nil {
		return Schedule{}, fmt.Errorf("dow: %w", err)
	}
	s.dowStar = star

	return s, nil
}

func parseField(field string, min, max int, names map[string]int) (mask uint64, star bool, err error) {
	field = strings.TrimSpace(field)
	if field == "" {
		return 0, false, errors.New("empty field")
	}
	// '?' == '*'
	if field == "?" {
		field = "*"
	}
	// "*"
	if field == "*" {
		return mustMask(min, max), true, nil
	}
	// "*/n"
	if strings.HasPrefix(field, "*/") {
		step, err := parseNumber(field[2:], names)
		if err != nil || step <= 0 {
			return 0, false, fmt.Errorf("invalid step %q", field)
		}
		var m uint64
		for v := min; v <= max; v += step {
			m |= 1 << uint(v-min)
		}
		return m, false, nil
	}

	// Comma-separated values and ranges (with optional /step)
	var m uint64
	parts := strings.Split(field, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return 0, false, fmt.Errorf("empty list atom in %q", field)
		}

		rng, step := p, 1
		if slash := strings.IndexByte(p, '/'); slash >= 0 {
			rng = p[:slash]
			st, err := parseNumber(p[slash+1:], names)
			if err != nil || st <= 0 {
				return 0, false, fmt.Errorf("invalid step in %q", p)
			}
			step = st
		}

		var lo, hi int
		if dash := strings.IndexByte(rng, '-'); dash >= 0 {
			v1, err := parseNumber(rng[:dash], names)
			if err != nil {
				return 0, false, err
			}
			v2, err := parseNumber(rng[dash+1:], names)
			if err != nil {
				return 0, false, err
			}
			lo, hi = v1, v2
		} else {
			v, err := parseNumber(rng, names)
			if err != nil {
				return 0, false, err
			}
			lo, hi = v, v
		}

		if lo < min || hi > max || lo > hi {
			return 0, false, fmt.Errorf("range/value out of bounds [%d..%d]: %d-%d", min, max, lo, hi)
		}

		for v := lo; v <= hi; v += step {
			m |= 1 << uint(v-min)
		}
	}
	return m, false, nil
}

func parseNumber(s string, names map[string]int) (int, error) {
	s = strings.TrimSpace(s)
	if names != nil {
		if v, ok := names[strings.ToLower(s)]; ok {
			return v, nil
		}
	}
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	var n int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

//
// Matching and Next()
//

func (s Schedule) dayMatches(t time.Time) bool {
	domOk := ((s.Dom >> uint(t.Day()-1)) & 1) != 0
	dowOk := ((s.Dow >> uint(t.Weekday())) & 1) != 0

	// Vixie OR semantics
	if !s.domStar && !s.dowStar {
		return domOk || dowOk
	}
	if !s.domStar && !domOk {
		return false
	}
	if !s.dowStar && !dowOk {
		return false
	}
	return true
}

// NextFrom returns the next matching instant strictly after t.
// If withSeconds=false: minute resolution, fires at second==0.
// If withSeconds=true: second resolution.
func (s Schedule) NextFrom(t time.Time, withSeconds bool) time.Time {
	loc := t.Location()
	var ts time.Time
	if withSeconds {
		ts = t.In(loc).Truncate(time.Second).Add(time.Second)
	} else {
		ts = t.In(loc).Truncate(time.Minute).Add(time.Minute)
	}

	for {
		y, m, d := ts.Date()
		mon := int(m)

		// Month
		if ((s.Month >> uint(mon-1)) & 1) == 0 {
			nextMon, wrap := nextAllowed(s.Month, mon, 1, 12)
			if wrap {
				y++
			}
			mon = nextMon
			ts = time.Date(y, time.Month(mon), 1, 0, 0, 0, 0, loc)
			if withSeconds {
				ts = ts.Add(time.Second)
			} else {
				ts = ts.Add(time.Minute)
			}
			continue
		}

		// Day
		if !s.dayMatches(ts) {
			ts = time.Date(y, m, d, 0, 0, 0, 0, loc).AddDate(0, 0, 1)
			if withSeconds {
				ts = ts.Add(time.Second)
			} else {
				ts = ts.Add(time.Minute)
			}
			continue
		}

		// Hour
		h := ts.Hour()
		if ((s.Hour >> uint(h)) & 1) == 0 {
			nextH, wrapH := nextAllowed(s.Hour, h, 0, 23)
			if wrapH {
				ts = time.Date(y, m, d, 0, 0, 0, 0, loc).AddDate(0, 0, 1)
			} else {
				ts = time.Date(y, m, d, nextH, 0, 0, 0, loc)
			}
			if withSeconds {
				ts = ts.Add(time.Second)
			} else {
				ts = ts.Add(time.Minute)
			}
			continue
		}

		// Minute
		fieldMin := ts.Minute()
		if ((s.Minute >> uint(fieldMin)) & 1) == 0 {
			nextM, wrapM := nextAllowed(s.Minute, fieldMin, 0, 59)
			if wrapM {
				ts = time.Date(y, m, d, h, 0, 0, 0, loc).Add(time.Hour)
			} else {
				ts = time.Date(y, m, d, h, nextM, 0, 0, loc)
			}
			if withSeconds {
				ts = ts.Add(time.Second)
			} else {
				// second=0 at minute resolution
				return ts
			}
			continue
		}

		// Second (only when enabled)
		if withSeconds {
			sec := ts.Second()
			if ((s.Second >> uint(sec)) & 1) == 0 {
				nextS, wrapS := nextAllowed(s.Second, sec, 0, 59)
				if wrapS {
					ts = time.Date(y, m, d, h, fieldMin, 0, 0, loc).Add(time.Minute)
					continue
				}
				return time.Date(y, m, d, h, fieldMin, nextS, 0, loc)
			}
			return ts
		}

		// All matched at minute resolution
		return time.Date(y, m, d, h, fieldMin, 0, 0, loc)
	}
}

func nextAllowed(mask uint64, cur, min, max int) (val int, wrap bool) {
	if mask == 0 {
		return min, true
	}
	if cur < min {
		cur = min
	}
	rel := cur - min
	shifted := mask >> uint(rel)
	if shifted != 0 {
		off := bits.TrailingZeros64(shifted)
		return cur + off, false
	}
	off := bits.TrailingZeros64(mask)
	return min + off, true
}
