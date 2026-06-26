package pomodoro

import (
	"testing"

	"github.com/Ippolid/focusbet/internal/domain"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name string
		cfg  domain.Pomodoro
		want Durations
	}{
		{
			name: "known preset wins over fields",
			cfg:  domain.Pomodoro{Preset: "classic", FocusMinutes: 99},
			want: Durations{Focus: 25, ShortBreak: 5, LongBreak: 20, CycleLen: 4},
		},
		{
			name: "deep preset",
			cfg:  domain.Pomodoro{Preset: "deep"},
			want: Durations{Focus: 50, ShortBreak: 10, LongBreak: 30, CycleLen: 3},
		},
		{
			name: "custom uses explicit fields",
			cfg:  domain.Pomodoro{Preset: "custom", FocusMinutes: 40, ShortBreakMinutes: 8, LongBreakMinutes: 25, CycleLength: 5},
			want: Durations{Focus: 40, ShortBreak: 8, LongBreak: 25, CycleLen: 5},
		},
		{
			name: "custom backfills missing fields from classic",
			cfg:  domain.Pomodoro{Preset: "custom", FocusMinutes: 40},
			want: Durations{Focus: 40, ShortBreak: 5, LongBreak: 20, CycleLen: 4},
		},
		{
			name: "unknown preset falls back to fields/classic",
			cfg:  domain.Pomodoro{Preset: "bogus"},
			want: Durations{Focus: 25, ShortBreak: 5, LongBreak: 20, CycleLen: 4},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Resolve(tc.cfg); got != tc.want {
				t.Errorf("Resolve = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestCycle_FullRotation(t *testing.T) {
	// CycleLen 2: focus1 -> short -> focus2 -> long -> focus1 ...
	c := NewCycle(Durations{Focus: 25, ShortBreak: 5, LongBreak: 20, CycleLen: 2})

	type step struct {
		phase Phase
		pos   int
	}
	want := []step{
		{PhaseFocus, 1},
		{PhaseShortBreak, 1},
		{PhaseFocus, 2},
		{PhaseLongBreak, 2},
		{PhaseFocus, 1}, // wraps back
		{PhaseShortBreak, 1},
	}

	for i, w := range want {
		if c.Phase() != w.phase || c.Position() != w.pos {
			t.Fatalf("step %d: phase=%v pos=%d, want phase=%v pos=%d",
				i, c.Phase(), c.Position(), w.phase, w.pos)
		}
		c.Advance()
	}
}

func TestCycle_PhaseMinutes(t *testing.T) {
	d := Durations{Focus: 25, ShortBreak: 5, LongBreak: 20, CycleLen: 1}
	c := NewCycle(d)

	if got := c.PhaseMinutes(); got != 25 {
		t.Errorf("focus minutes = %d, want 25", got)
	}
	c.Advance() // CycleLen 1 -> long break after first focus
	if c.Phase() != PhaseLongBreak {
		t.Fatalf("phase = %v, want long break", c.Phase())
	}
	if got := c.PhaseMinutes(); got != 20 {
		t.Errorf("long break minutes = %d, want 20", got)
	}
}

func TestTimer_RunPauseResumeComplete(t *testing.T) {
	var start int64 = 1_000_000
	tm := NewTimer(25) // 1500s

	if tm.State(start) != StateIdle {
		t.Fatalf("state = %v, want idle", tm.State(start))
	}

	tm.Start(start)
	at := start + 600 // 10 min in
	if done := tm.Tick(at); done {
		t.Fatal("completed early")
	}
	if got := tm.FocusedMinutes(at); got != 10 {
		t.Errorf("focused = %v, want 10", got)
	}
	if p := tm.Progress(at); p < 0.39 || p > 0.41 {
		t.Errorf("progress = %.3f, want ~0.40", p)
	}

	// Pause, jump the clock an hour, resume: elapsed must stay 600s.
	tm.Pause(at)
	resume := at + 3600
	tm.Start(resume)
	if got := tm.ElapsedSeconds(resume); got != 600 {
		t.Errorf("elapsed after pause = %d, want 600", got)
	}

	// Run the remaining 900s.
	end := resume + 900
	if done := tm.Tick(end); !done {
		t.Fatal("should be done")
	}
	if tm.State(end) != StateDone {
		t.Errorf("state = %v, want done", tm.State(end))
	}
	if got := tm.RemainingSeconds(end); got != 0 {
		t.Errorf("remaining = %d, want 0", got)
	}
}

func TestTimer_StateDoneWithoutTick(t *testing.T) {
	const start = 500
	tm := NewTimer(1) // 60s
	tm.Start(start)
	// Past the planned length but no Tick called: State must still report Done.
	if tm.State(start+120) != StateDone {
		t.Errorf("state = %v, want done", tm.State(start+120))
	}
}

func TestTimer_ClockGoesBackwards(t *testing.T) {
	const start = 1000
	tm := NewTimer(10)
	tm.Start(start)
	// now < startedAt must not produce negative elapsed.
	if got := tm.ElapsedSeconds(start - 50); got != 0 {
		t.Errorf("elapsed = %d, want 0 on backwards clock", got)
	}
}

func TestComputeReward(t *testing.T) {
	e := domain.Economy{}

	// Full session: 50 focus / 50 planned, 10-min break -> banks the whole break.
	r := ComputeReward(50, 50, 10, e)
	if r.Fair != 10 {
		t.Errorf("fair = %v, want 10", r.Fair)
	}
	if r.FocusedMinutes != 50 {
		t.Errorf("focused = %v, want 50", r.FocusedMinutes)
	}

	// Stopped at half the planned focus -> half the break.
	if h := ComputeReward(25, 50, 10, e); h.Fair != 5 {
		t.Errorf("half-session fair = %v, want 5", h.Fair)
	}

	// Zero break -> nothing earned.
	if z := ComputeReward(50, 50, 0, e); z.Fair != 0 {
		t.Errorf("fair = %v for zero break, want 0", z.Fair)
	}
}

func TestEngine_FocusCompletesThenBreak(t *testing.T) {
	cfg := domain.Pomodoro{Preset: "classic"}
	eng := NewEngine(cfg, domain.Economy{}, domain.Games{})
	if eng.Phase() != PhaseFocus {
		t.Fatalf("phase = %v, want focus", eng.Phase())
	}

	const start = 2_000_000
	eng.Start(start)

	// Mid-focus: no completion.
	if res := eng.Tick(start + 600); res.Completed {
		t.Fatal("completed at 10min, want not yet")
	}

	// Complete the 25-min focus.
	res := eng.Tick(start + 25*60)
	if !res.Completed || !res.WasFocus {
		t.Fatalf("res = %+v, want completed focus", res)
	}
	if res.FocusedMinutes != 25 {
		t.Errorf("focused = %v, want 25", res.FocusedMinutes)
	}
	if res.Reward.Fair != 5 { // classic break is 5 min
		t.Errorf("reward fair = %v, want 5", res.Reward.Fair)
	}
	if res.Next != PhaseShortBreak {
		t.Errorf("next = %v, want short break", res.Next)
	}
}

func TestEngine_StopEarly(t *testing.T) {
	cfg := domain.Pomodoro{Preset: "classic"}
	eco := domain.Economy{}
	eng := NewEngine(cfg, eco, domain.Games{BaseStakeMinutes: 5})

	const start = 3_000_000
	eng.Start(start)
	res := eng.StopEarly(start + 10*60) // stopped at 10 of 25 minutes
	if !res.Completed || !res.WasFocus {
		t.Fatalf("res = %+v, want completed focus", res)
	}
	if res.FocusedMinutes != 10 {
		t.Errorf("focused = %v, want 10", res.FocusedMinutes)
	}
	if res.Next != PhaseShortBreak {
		t.Errorf("next = %v, want short break", res.Next)
	}
}

func TestEngine_SnapshotRunningFocus(t *testing.T) {
	eng := NewEngine(domain.Pomodoro{Preset: "classic"}, domain.Economy{}, domain.Games{})
	const start = 4_000_000

	// Idle focus is not snapshotted.
	if _, ok := eng.Snapshot(start); ok {
		t.Error("snapshot ok on idle, want false")
	}

	eng.Start(start)
	s, ok := eng.Snapshot(start + 300)
	if !ok {
		t.Fatal("snapshot not ok on running focus")
	}
	if s.FocusSeconds != 25*60 {
		t.Errorf("focus seconds = %d, want 1500", s.FocusSeconds)
	}
	if s.StartedAt != start { // start = now - elapsed(300)
		t.Errorf("started at = %d, want %d", s.StartedAt, start)
	}
	if s.Preset != "classic" {
		t.Errorf("preset = %q, want classic", s.Preset)
	}
}

func TestUpdateStreak(t *testing.T) {
	// unix seconds for three known UTC days.
	day := func(unix int64) int64 { return unix }
	const d24 = 1_750_000_000 // some day
	const oneDay = 24 * 60 * 60

	var stats domain.Stats

	UpdateStreak(&stats, day(d24))
	if stats.StreakDays != 1 {
		t.Fatalf("streak = %d, want 1", stats.StreakDays)
	}

	// same day again: unchanged
	UpdateStreak(&stats, day(d24+100))
	if stats.StreakDays != 1 {
		t.Errorf("streak = %d, want 1 (same day)", stats.StreakDays)
	}

	// next day: +1
	UpdateStreak(&stats, day(d24+oneDay))
	if stats.StreakDays != 2 {
		t.Errorf("streak = %d, want 2", stats.StreakDays)
	}

	// skip a day: reset to 1
	UpdateStreak(&stats, day(d24+3*oneDay))
	if stats.StreakDays != 1 {
		t.Errorf("streak = %d, want 1 (gap)", stats.StreakDays)
	}
}

func TestKeptEarnings(t *testing.T) {
	eco := domain.Economy{InterruptKeepFraction: 0.5}
	eng := NewEngine(domain.Pomodoro{Preset: "classic"}, eco, domain.Games{})
	// classic: 25 focus / 5 break -> banked = whole fair break = 5.
	r := ComputeReward(25, 25, 5, eco)

	if got := eng.KeptEarnings(r, false); got != 5 {
		t.Errorf("full session kept = %v, want 5", got)
	}
	if got := eng.KeptEarnings(r, true); got != 2.5 { // 5 * 0.5
		t.Errorf("interrupted kept = %v, want 2.5", got)
	}
}
