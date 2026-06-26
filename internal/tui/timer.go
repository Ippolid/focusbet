package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/pomodoro"
)

// timerScreen runs a focus (or break) phase via the pomodoro engine. It owns the
// engine for this run and advances it on each one-second pulse.
type timerScreen struct {
	root      *model
	engine    *pomodoro.Engine
	task      string
	paused    bool
	shownFrac float64 // eased bar fill, chases the real progress for smooth motion
	framing   bool    // whether the smooth-bar frame loop is currently running
}

func newTimer(root *model, p taskPayload) *timerScreen {
	eng := root.core.NewEngine()
	eng.Start(root.now())
	return &timerScreen{
		root:   root,
		engine: eng,
		task:   p.task,
	}
}

func (t *timerScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := t.root.keys
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, km.Quit):
			return t, t.root.quit()
		case keyMatches(msg, km.Back):
			return t, switchTo(screenDashboard, nil)
		case keyMatches(msg, km.Pause):
			t.togglePause()
		case keyMatches(msg, km.Stop):
			return t, t.finish(true)
		}

	case tickMsg:
		if t.paused {
			return t, nil
		}
		res := t.engine.Tick(t.root.now())
		if res.Completed && res.WasFocus {
			// Focus finished: fire the multi-channel alert and route by flow mode.
			return t, tea.Batch(
				alert("Focus done 🎯", "Nice work — time for a break."),
				t.toResult(res.FocusedMinutes, false),
			)
		}
		// Kick the smooth-bar frame loop if it isn't already running.
		if !t.framing {
			t.framing = true
			return t, frameTick()
		}

	case frameMsg:
		if t.paused {
			t.framing = false
			return t, nil // resume the loop on the next tick after un-pause
		}
		target := t.engine.Timer().Progress(t.root.now())
		next, more := easeToward(t.shownFrac, target)
		t.shownFrac = next
		if more || t.shownFrac < 1 {
			t.framing = true // own the loop so tickMsg won't start a second one
			return t, frameTick()
		}
		t.framing = false
	}
	return t, nil
}

func (t *timerScreen) View() string {
	st := t.root.styles
	now := t.root.now()
	tm := t.engine.Timer()

	// Big ASCII-art time, a slim bar under it, the task below — matching the mockup.
	remaining := domain.MinutesFromSeconds(tm.RemainingSeconds(now))
	header := st.art.Render(bigText(remaining.String(), digitFont))

	taskLine := t.task
	if taskLine == "" {
		taskLine = "focus"
	}

	// Use the eased fill once the frame loop has warmed up; on the very first
	// render (before any frame) fall back to the exact progress so the bar isn't
	// empty for a beat.
	frac := t.shownFrac
	if !t.framing {
		frac = tm.Progress(now)
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		smoothBar(frac, 32, st.good, st.help),
		"",
		st.subtitle.Render(taskLine),
		"",
		st.subtitle.Render(t.root.cycleDots()),
	)
	return st.frame(t.root.width, t.root.height, header, body, t.statusLine())
}

// statusLine is the dim footer with phase, pause state and key hints.
func (t *timerScreen) statusLine() string {
	phase := "focus"
	if t.engine.Phase().IsBreak() {
		phase = "break"
	}
	if t.paused {
		phase += " (paused)"
	}
	return phase + " • p pause • s stop • esc menu • q quit"
}

func (t *timerScreen) togglePause() {
	now := t.root.now()
	if t.paused {
		t.engine.Start(now)
	} else {
		t.engine.Pause(now)
	}
	t.paused = !t.paused
}

// finish ends the focus session early (or via stop) and routes to the result.
func (t *timerScreen) finish(interrupted bool) tea.Cmd {
	res := t.engine.StopEarly(t.root.now())
	if !res.WasFocus {
		return switchTo(screenDashboard, nil)
	}
	return t.toResult(res.FocusedMinutes, interrupted)
}

// toResult settles the completed focus session through the app core, then routes
// by flow mode: on -> straight into the auto-rest countdown; off -> the fork that
// lets the user confirm what to do next.
func (t *timerScreen) toResult(focused domain.Minutes, interrupted bool) tea.Cmd {
	// Settlement persists stats/streak, so stamp it with real wall-clock time.
	now := t.root.wallNow()

	// Advance the cycle first so settlement knows whether the long break follows;
	// a completed cycle earns (and rests) the long break. An interrupted session
	// does not advance the cycle.
	isLong := false
	if !interrupted {
		isLong = t.root.onFocusComplete()
	}

	reward, banked, err := t.root.core.CompleteFocus(now, focused, interrupted, isLong)

	breakMin := reward.Fair // the rest the user is about to take (short or long)
	if isLong {
		breakMin = domain.Minutes(t.root.core.LongBreakMinutes())
	}

	payload := resultPayload{
		focused:      focused,
		reward:       reward,
		banked:       banked,
		interrupted:  interrupted,
		err:          err,
		task:         t.task,
		breakMinutes: breakMin,
		isLongBreak:  isLong,
	}
	if t.root.core.FlowMode() && !interrupted {
		return switchTo(screenRestTimer, payload)
	}
	return switchTo(screenResult, payload)
}
