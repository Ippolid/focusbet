package pomodoro

import "github.com/Ippolid/focusbet/internal/domain"

// TimerState is the run state of a single timed phase.
type TimerState int

const (
	// StateIdle means no timer is running and none has completed.
	StateIdle TimerState = iota
	// StateRunning means the timer is counting toward its planned length.
	StateRunning
	// StatePaused means the timer is frozen; banked elapsed time is preserved.
	StatePaused
	// StateDone means the timer reached its planned length.
	StateDone
)

// Timer is a pure, clock-driven countdown for one phase (focus or break). It
// never reads the wall clock itself: every method that needs "the current time"
// takes a unix-seconds now, so behaviour is fully deterministic and testable.
//
// Elapsed time is derived from now, not from how often Tick is called, so the
// timer is correct regardless of UI refresh rate and survives the process being
// suspended.
type Timer struct {
	plannedSeconds int64
	elapsed        int64 // banked seconds from completed running spells
	startedAt      int64 // unix seconds of the last transition into StateRunning
	state          TimerState
}

// NewTimer creates an idle timer for a phase of the given length in minutes.
func NewTimer(minutes int) *Timer {
	if minutes < 0 {
		minutes = 0
	}
	return &Timer{plannedSeconds: int64(minutes) * 60, state: StateIdle}
}

// NewTimerSeconds creates an idle timer for a phase of the given length in whole
// seconds, so a fractional-minute break is timed exactly instead of being
// rounded up to a full minute.
func NewTimerSeconds(seconds int64) *Timer {
	if seconds < 0 {
		seconds = 0
	}
	return &Timer{plannedSeconds: seconds, state: StateIdle}
}

// FromSession rebuilds a running timer from a persisted session, so a focus run
// survives a restart. The session's StartedAt and FocusSeconds anchor the timer.
func FromSession(s domain.Session) *Timer {
	return &Timer{
		plannedSeconds: s.FocusSeconds,
		startedAt:      s.StartedAt,
		state:          StateRunning,
	}
}

// State returns the current run state, evaluated at now: a running timer that
// has reached its planned length reports StateDone without needing a Tick.
func (t *Timer) State(now int64) TimerState {
	if t.state == StateRunning && t.elapsedAt(now) >= t.plannedSeconds {
		return StateDone
	}
	return t.state
}

// Start begins or resumes counting at now. It is a no-op if already running or
// already done.
func (t *Timer) Start(now int64) {
	if t.state == StateRunning || t.state == StateDone {
		return
	}
	t.state = StateRunning
	t.startedAt = now
}

// Pause freezes the timer at now, banking the elapsed running time.
func (t *Timer) Pause(now int64) {
	if t.state != StateRunning {
		return
	}
	t.elapsed += t.runningSpan(now)
	t.state = StatePaused
}

// Tick advances the timer to now and reports whether it just completed. Calling
// it at any cadence is safe; progress comes from now, not the call count.
func (t *Timer) Tick(now int64) (completed bool) {
	if t.state != StateRunning {
		return false
	}
	if t.elapsedAt(now) >= t.plannedSeconds {
		t.elapsed = t.plannedSeconds
		t.startedAt = 0
		t.state = StateDone
		return true
	}
	return false
}

// ElapsedSeconds returns total counted seconds at now, capped at the planned length.
func (t *Timer) ElapsedSeconds(now int64) int64 {
	e := t.elapsedAt(now)
	if e > t.plannedSeconds {
		return t.plannedSeconds
	}
	return e
}

// RemainingSeconds returns seconds left until the planned length at now.
func (t *Timer) RemainingSeconds(now int64) int64 {
	rem := t.plannedSeconds - t.ElapsedSeconds(now)
	if rem < 0 {
		return 0
	}
	return rem
}

// Progress returns completion in [0,1] at now. A zero-length phase is complete.
func (t *Timer) Progress(now int64) float64 {
	if t.plannedSeconds <= 0 {
		return 1
	}
	return float64(t.ElapsedSeconds(now)) / float64(t.plannedSeconds)
}

// FocusedMinutes returns whole counted minutes at now, used to compute earnings.
func (t *Timer) FocusedMinutes(now int64) domain.Minutes {
	return domain.Minutes(float64(t.ElapsedSeconds(now)) / 60)
}

// elapsedAt is the raw (uncapped) elapsed seconds at now.
func (t *Timer) elapsedAt(now int64) int64 {
	return t.elapsed + t.runningSpan(now)
}

// runningSpan is the seconds counted in the current running spell, or 0 if not
// running. A clock that goes backwards (now < startedAt) yields 0, never negative.
func (t *Timer) runningSpan(now int64) int64 {
	if t.state != StateRunning {
		return 0
	}
	if now <= t.startedAt {
		return 0
	}
	return now - t.startedAt
}
