package pomodoro

import (
	"time"

	"github.com/Ippolid/focusbet/internal/domain"
)

// Engine drives a full pomodoro run: it owns the cycle position and the timer
// for the current phase, and resolves what happens when a phase completes. It
// is pure with respect to time — every method that needs the clock takes a
// unix-seconds now — and it never touches persistence or the bank directly.
type Engine struct {
	cfg     domain.Pomodoro
	economy domain.Economy
	games   domain.Games

	cycle *Cycle
	timer *Timer
}

// NewEngine builds an idle engine from the user's config. The first phase is the
// first focus session of a fresh cycle; call Start to begin counting.
func NewEngine(cfg domain.Pomodoro, economy domain.Economy, games domain.Games) *Engine {
	d := Resolve(cfg)
	cycle := NewCycle(d)
	return &Engine{
		cfg:     cfg,
		economy: economy,
		games:   games,
		cycle:   cycle,
		timer:   NewTimer(cycle.PhaseMinutes()),
	}
}

// Phase returns the current phase.
func (e *Engine) Phase() Phase { return e.cycle.Phase() }

// Position returns the 1-based focus-session index within the cycle.
func (e *Engine) Position() int { return e.cycle.Position() }

// Timer exposes the current phase's timer for progress/remaining queries.
func (e *Engine) Timer() *Timer { return e.timer }

// Start begins (or resumes) the current phase at now.
func (e *Engine) Start(now int64) { e.timer.Start(now) }

// Pause freezes the current phase at now.
func (e *Engine) Pause(now int64) { e.timer.Pause(now) }

// PhaseResult reports what completed and what the user can do next.
type PhaseResult struct {
	// Completed is true when the current phase reached its planned length at now.
	Completed bool
	// WasFocus is true when the completed phase was a focus session.
	WasFocus bool
	// FocusedMinutes is the minutes focused, set only when WasFocus is true.
	FocusedMinutes domain.Minutes
	// Reward is the base/fair/max breakdown, set only when WasFocus is true.
	Reward Reward
	// Next is the phase the engine moved to after completion.
	Next Phase
}

// Tick advances the timer to now. If the phase just completed it auto-advances
// the cycle, arms the timer for the next phase, and — for a finished focus
// session — computes the reward that feeds the gambling layer.
//
// The engine does not move any currency. The caller settles the result: bank
// Reward.EarnMinutes via balance.Earn.
func (e *Engine) Tick(now int64) PhaseResult {
	wasFocus := e.cycle.Phase() == PhaseFocus
	if !e.timer.Tick(now) {
		return PhaseResult{Completed: false, WasFocus: wasFocus, Next: e.cycle.Phase()}
	}

	res := PhaseResult{Completed: true, WasFocus: wasFocus}
	if wasFocus {
		res.FocusedMinutes = e.timer.FocusedMinutes(now)
		res.Reward = ComputeReward(res.FocusedMinutes, e.plannedFocus(), e.breakMinutes(), e.economy)
	}
	res.Next = e.advance()
	return res
}

// plannedFocus and breakMinutes read the cycle's resolved durations.
func (e *Engine) plannedFocus() domain.Minutes {
	return domain.Minutes(e.cycle.Durations().Focus)
}

func (e *Engine) breakMinutes() domain.Minutes {
	return domain.Minutes(e.cycle.Durations().ShortBreak)
}

// StopEarly ends the current focus session before its planned length at now,
// returning the reward for the focused minutes so far. Applying the early-stop
// penalty (InterruptKeepFraction) is the caller's job; the engine just reports
// the raw focused time. It then advances to the break.
func (e *Engine) StopEarly(now int64) PhaseResult {
	if e.cycle.Phase() != PhaseFocus {
		return PhaseResult{Completed: false, WasFocus: false, Next: e.cycle.Phase()}
	}
	focused := e.timer.FocusedMinutes(now)
	res := PhaseResult{
		Completed:      true,
		WasFocus:       true,
		FocusedMinutes: focused,
		Reward:         ComputeReward(focused, e.plannedFocus(), e.breakMinutes(), e.economy),
	}
	res.Next = e.advance()
	return res
}

// KeptEarnings applies the early-stop penalty to a reward's bankable emission:
// a full (non-interrupted) session keeps everything; an interrupted one keeps
// only InterruptKeepFraction of the fair-minus-base earnings. This is the amount
// the caller should pass to balance.Earn.
func (e *Engine) KeptEarnings(r Reward, interrupted bool) domain.Minutes {
	earn := r.EarnMinutes()
	if !interrupted {
		return earn
	}
	keep := e.economy.InterruptKeepFraction
	if keep <= 0 {
		return 0
	}
	if keep >= 1 {
		return earn
	}
	return domain.Minutes(float64(earn) * keep)
}

// Snapshot captures the current focus session for persistence, so a run survives
// a restart. It returns ok=false when the current phase is not a running focus
// session.
func (e *Engine) Snapshot(now int64) (domain.Session, bool) {
	if e.cycle.Phase() != PhaseFocus || e.timer.State(now) != StateRunning {
		return domain.Session{}, false
	}
	return domain.Session{
		StartedAt:     now - e.timer.ElapsedSeconds(now),
		FocusSeconds:  int64(e.cycle.PhaseMinutes()) * 60,
		Preset:        e.cfg.Preset,
		CyclePosition: e.cycle.Position(),
	}, true
}

// advance moves the cycle to the next phase and re-arms the timer for it.
func (e *Engine) advance() Phase {
	next := e.cycle.Advance()
	e.timer = NewTimer(e.cycle.PhaseMinutes())
	return next
}

// UpdateStreak advances the daily focus streak in stats based on the calendar
// day of now in the user's local time zone. Same-day completions don't change
// the streak; a consecutive day increments it; a gap resets it to 1. It is pure:
// pass now, no clock is read (the local zone, not wall-clock, decides the day).
func UpdateStreak(stats *domain.Stats, now int64) {
	today := time.Unix(now, 0).Local().Format("2006-01-02")
	yesterday := time.Unix(now, 0).Local().AddDate(0, 0, -1).Format("2006-01-02")

	switch stats.LastActiveDate {
	case today:
		// already counted today
	case yesterday:
		stats.StreakDays++
	default:
		stats.StreakDays = 1
	}
	stats.LastActiveDate = today
}
