// Package pomodoro implements the focus-timer state machine: presets, the
// focus/break cycle, a pure clock-driven timer, and the reward hook that runs
// after a focus session completes and feeds the gambling layer.
package pomodoro

import "github.com/Ippolid/focusbet/internal/domain"

// Durations is the resolved set of timings for one pomodoro configuration, in
// whole minutes. It is what the cycle and timer actually run on.
type Durations struct {
	Focus      int // focus-session length
	ShortBreak int // break after a normal focus session
	LongBreak  int // break after a full cycle
	CycleLen   int // focus sessions per cycle before a long break
}

// builtinPresets maps a preset name to its timings. "custom" is handled
// separately by reading the explicit fields off domain.Pomodoro.
var builtinPresets = map[string]Durations{
	"classic":  {Focus: 25, ShortBreak: 5, LongBreak: 20, CycleLen: 4},
	"deep":     {Focus: 50, ShortBreak: 10, LongBreak: 30, CycleLen: 3},
	"desktime": {Focus: 52, ShortBreak: 17, LongBreak: 17, CycleLen: 4},
	"short":    {Focus: 15, ShortBreak: 3, LongBreak: 15, CycleLen: 4},
}

// Resolve turns a domain.Pomodoro config into concrete Durations.
//
// For a known built-in preset it returns that preset's timings. For "custom"
// (or an unknown preset) it falls back to the explicit fields on the config,
// and any non-positive field is backfilled from the classic preset so the timer
// never runs on a zero-length phase.
func Resolve(p domain.Pomodoro) Durations {
	if d, ok := builtinPresets[p.Preset]; ok {
		return d
	}

	classic := builtinPresets["classic"]
	d := Durations{
		Focus:      p.FocusMinutes,
		ShortBreak: p.ShortBreakMinutes,
		LongBreak:  p.LongBreakMinutes,
		CycleLen:   p.CycleLength,
	}
	if d.Focus <= 0 {
		d.Focus = classic.Focus
	}
	if d.ShortBreak <= 0 {
		d.ShortBreak = classic.ShortBreak
	}
	if d.LongBreak <= 0 {
		d.LongBreak = classic.LongBreak
	}
	if d.CycleLen <= 0 {
		d.CycleLen = classic.CycleLen
	}
	return d
}
