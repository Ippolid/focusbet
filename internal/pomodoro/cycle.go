package pomodoro

// Phase is one segment of the pomodoro cycle.
type Phase int

const (
	// PhaseFocus is a focus/work segment.
	PhaseFocus Phase = iota
	// PhaseShortBreak is the break after a normal focus session.
	PhaseShortBreak
	// PhaseLongBreak is the break after a full cycle of focus sessions.
	PhaseLongBreak
)

// String implements fmt.Stringer.
func (p Phase) String() string {
	switch p {
	case PhaseFocus:
		return "focus"
	case PhaseShortBreak:
		return "short_break"
	case PhaseLongBreak:
		return "long_break"
	default:
		return "unknown"
	}
}

// IsBreak reports whether the phase is any kind of break.
func (p Phase) IsBreak() bool { return p == PhaseShortBreak || p == PhaseLongBreak }

// Cycle tracks position within the focus/break rotation. It is pure state: it
// holds no timer and no wall-clock, it only answers "what phase are we in and
// what comes next".
//
// Position is the 1-based index of the current focus session within the cycle
// (1..CycleLen). A long break fires after the CycleLen-th focus session and
// resets the position back to 1.
type Cycle struct {
	durations Durations
	phase     Phase
	position  int // 1..CycleLen, the focus session we are on
}

// NewCycle starts a fresh cycle on its first focus session.
func NewCycle(d Durations) *Cycle {
	return &Cycle{durations: d, phase: PhaseFocus, position: 1}
}

// Phase returns the current phase.
func (c *Cycle) Phase() Phase { return c.phase }

// Position returns the 1-based focus-session index within the cycle.
func (c *Cycle) Position() int { return c.position }

// Durations returns the resolved timings driving the cycle.
func (c *Cycle) Durations() Durations { return c.durations }

// PhaseMinutes returns the planned length of the current phase in minutes.
func (c *Cycle) PhaseMinutes() int {
	switch c.phase {
	case PhaseShortBreak:
		return c.durations.ShortBreak
	case PhaseLongBreak:
		return c.durations.LongBreak
	default:
		return c.durations.Focus
	}
}

// Advance moves to the next phase and returns it.
//
// Transitions: a completed focus session goes to a break — a long break when
// the session was the last in the cycle, otherwise a short break. A completed
// break goes back to focus, advancing the position (and wrapping to 1 after a
// long break).
func (c *Cycle) Advance() Phase {
	switch c.phase {
	case PhaseFocus:
		if c.position >= c.durations.CycleLen {
			c.phase = PhaseLongBreak
		} else {
			c.phase = PhaseShortBreak
		}
	case PhaseShortBreak:
		c.phase = PhaseFocus
		c.position++
	case PhaseLongBreak:
		c.phase = PhaseFocus
		c.position = 1
	}
	return c.phase
}
