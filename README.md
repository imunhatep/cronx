# `cron` — a tiny, channel‑driven cron for Go (Timer‑style, bitmasks, TZ/DST‑aware)

A minimal, **clean**, and **fast** cron scheduler modeled after Go’s `time.Timer`/`time.Ticker` style:

*   Emits ticks via a **receive‑only channel** (`C <-chan time.Time`).
*   **Bitmask** schedule representation for speed and low memory.
*   **Option‑driven seconds**: minute resolution by default; opt‑in to per‑second schedules.
*   Correct **Vixie cron OR semantics** for `DOM` vs `DOW`.
*   **Timezone/DST** aware (uses `*time.Location` everywhere).
*   No dynamic resets: just `Stop()` and create a new `Cron` (simple, leak‑free runtime).

***

## ✨ Features

*   **Timer‑like API**: simple `C <-chan time.Time`, `Stop()` to terminate.
*   **Fast matching**: `uint64` bitmasks + `math/bits` to scan next allowed value.
*   **5‑field or 6‑field cron** (option‑controlled, see below).
*   **Month/weekday names** (`jan..dec`, `sun..sat`), ranges (`a-b`), steps (`*/n`), lists (`a,b,c`).
*   **Vixie semantics** for `day-of-month` vs `day-of-week` (match if either matches when both are set).
*   **TZ/DST aware** using Go’s `time` package.

***

## 📦 Install

```bash
go get github.com/your-org/cron
```

> Replace `github.com/your-org/cron` with your actual module path.

***

## 🚀 Quick Start

### Minute‑level (default, 5 fields)

```go
c, err := cron.New("*/5 * * * *") // every 5 minutes at second 0
if err != nil { panic(err) }
defer c.Stop()

for t := range c.C {
    fmt.Println("tick:", t)
}
```

### Per‑second (opt‑in, seconds **last**)

```go
// Every minute at :00 and :30 (seconds field is last)
c, err := cron.New("0,30 */1 * * * *", cron.WithSeconds())
if err != nil { panic(err) }
defer c.Stop()

for t := range c.C {
    fmt.Println("tick:", t)
}
```

### Timezone & buffering

```go
loc, _ := time.LoadLocation("America/New_York")
c, _ := cron.New(
    "0 9-17/2 * * mon-fri", // 09:00, 11:00, 13:00, 15:00, 17:00 (minute resolution)
    cron.WithLocation(loc),
    cron.WithBuffered(1),   // buffer one tick
)
defer c.Stop()
```

***

## 🧭 Seconds & Field Order (option‑driven)

This package uses a **single canonical 6‑field layout with seconds as the last field**:

    sec min hour dom mon dow

Behavior depends on the `WithSeconds()` option:

*   **Without `WithSeconds()` (default)**  
    Expect **5 fields** (`min hour dom mon dow`).  
    Seconds are fixed to **0** (ticks occur at `:00` only, i.e., minute resolution).

*   **With `WithSeconds()`**  
    Accept **5 or 6 fields**:
    *   **5 fields** → seconds default to `"*"` (every second).
    *   **6 fields** → the **first field is seconds** (`sec min hour dom mon dow`).

Examples:

| Spec               | WithSeconds | Normalized (internal) | Meaning                              |
|--------------------| ----------: |---------------------| ------------------------------------ |
| `*/5 * * * *`      |           ❌ | `* */5 * * * *`     | Every 5 minutes at `:00`             |
| `*/2 * * * *`      |           ✅ | `*/2 * * * * *`     | Every second during every 2nd minute |
| `0,30 */1 * * * *` |           ✅ | as-is               | Every minute at `:00` and `:30`      |

> **Why not infer by field count?** To keep the API unsurprising and explicit: **seconds are enabled only when you ask for it**.

***

## 🧩 Cron Syntax

*   **Fields (with seconds enabled)**: `sec min hour dom mon dow`
*   **Fields (default)**: `min hour dom mon dow`
*   **Ranges**: `a-b` (inclusive)
*   **Lists**: `a,b,c`
*   **Steps**: `*/n`, `a-b/n`
*   **Names** (case‑insensitive):
    *   Months: `jan..dec`
    *   Weekdays: `sun..sat`
*   **DOW `7`** is normalized to `0` (Sunday).
*   **`?`** is treated as `*` (wildcard).
*   **DOM vs DOW**: **Vixie OR semantics** — when both are **specific**, a date matches if **either** matches.

Examples:

*   `0 9-17/2 * * mon-fri` → 9:00, 11:00, 13:00, 15:00, 17:00 on weekdays
*   `*/10 * * * * *` (+`WithSeconds`) → every 10 seconds
*   `0 0 1 jan mon` → on Jan 1 **or** any Monday at 00:00 (Vixie OR)

***

## 📚 API Overview

```go
// Build & run a cron
func New(spec string, opts ...Option) (*Cron, error)

// Alternative: parse spec using option-driven seconds rule
func ParseSpec(spec string, withSeconds bool) (Schedule, error)

// Lower-level constructor if you want to re-use a Schedule
func NewSchedule(s Schedule, opts ...Option) (*Cron, error)

type Cron struct {
    C <-chan time.Time       // receive ticks here
}
func (c *Cron) Stop() bool   // stop the goroutine; returns true if it stopped
func (c *Cron) Next(from time.Time) time.Time // compute the next tick (helper)

// Options
func WithLocation(loc *time.Location) Option
func WithBuffered(n int) Option
func WithStartFrom(from time.Time) Option
func WithSeconds() Option // enable second-level scheduling & 6th field
```

**Runtime semantics**

*   `C` blocks like `time.Timer` if the receiver is slow (use `WithBuffered` to buffer).
*   `Stop()` does **not** close `C` (same semantics as `time.Timer`/`time.Ticker`).
*   **No dynamic Reset**; if you need a new schedule, `Stop()` and create a new `Cron`.

***

## 🧠 Design Notes

*   **Bitmasks**: Each field is a 64‑bit mask (e.g., minute 0..59) for O(1) checks; next matching value is found via `bits.TrailingZeros64`.
*   **Single parser path**: We normalize to the canonical 6‑field layout (seconds **first**) and parse once.
*   **TZ/DST**: All computations use your supplied `*time.Location`; DST gaps/overlaps are handled by Go’s `time` library.

***

## ⚠️ Edge Cases & Semantics

*   **DST transitions**: Ticks that would fall into a non‑existent local timestamp are normalized by `time.Date`. The next matching time is always computed **forward**.
*   **DOM/DOW OR**: Example — `dom=1` and `dow=mon` will fire on the 1st of the month **or** on any Monday.
*   **Whitespace**: Multiple spaces/tabs are fine (we use `strings.Fields`).
*   **Validation**: Out‑of‑range values (e.g., minute `60`) are rejected with clear errors.

***

## 🔍 FAQ

**Q: Why not support `Reset()`?**  
A: Keeping the runtime state immutable avoids cross‑goroutine synchronization complexity and potential leaks. Creating a new cron is cheap and clear.

**Q: Where are seconds in the spec? First or last?**  
A: **Last**. We chose `sec min hour dom mon dow` to make extending classic 5‑field cron intuitive. Seconds are **enabled explicitly** via `WithSeconds()`.

**Q: Can I preview upcoming runs without starting a goroutine?**  
A: Yes—use `ParseSpec` + `Schedule.NextFrom` (via `Cron.Next`) to enumerate future instants.

***

## ✅ Examples (snippets)

### Preview next 5 runs (minute resolution)

```go
s, _ := cron.ParseSpec("0 9-17/2 * * mon-fri", false) // no seconds
c, _ := cron.NewSchedule(s)
t := time.Now()
for i := 0; i < 5; i++ {
    t = c.Next(t)
    fmt.Println("Next:", t)
}
```

### Every 15 seconds between 10:00–10:10

```go
c, _ := cron.New("*/15 0-10 10 * * *", cron.WithSeconds())
defer c.Stop()
for t := range c.C { fmt.Println(t) }
```

***

## 🧪 Testing ideas (if you contribute)

*   Parsing: names, ranges, steps, DOW `7→0`.
*   `Next` across month/year rollovers.
*   DOM/DOW OR behavior.
*   DST “spring forward” / “fall back” transitions in various timezones.
*   Second vs minute resolution correctness.

***

## 📄 License

MIT — see `LICENSE` file.

***

## 🙌 Contributing

Issues and PRs welcome! Please include:

*   Clear reproduction cases or failing tests.
*   Bench/pprof data if proposing performance changes.
*   Rationale for user‑visible behavior changes (especially parsing/semantics).

***

## 🗺️ Roadmap (nice‑to‑have)

*   Optional **WithLogger** to log internals.
*   **Context** support (`WithContext`) for cooperative cancellation.
*   A tiny **CLI** to print next N runs for a spec and timezone.

***

If you want, I can also generate a small `examples/` folder and a `go doc` badge once you settle on the module path.
