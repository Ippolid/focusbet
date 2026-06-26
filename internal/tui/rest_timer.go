package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/pomodoro"
)

// restTimerScreen is the break countdown taken after a focus session: a timer on
// the break minutes, with a Play option that interrupts the rest to gamble the
// surplus. Shown automatically in flow mode (where it chains into the next focus)
// or by choosing "Rest" in the manual fork.
type restTimerScreen struct {
	root      *model
	payload   resultPayload
	timer     *pomodoro.Timer
	shownFrac float64 // eased bar fill, chases the real progress for smooth motion
	framing   bool    // whether the smooth-bar frame loop is currently running
}

func newRestTimer(root *model, p resultPayload) *restTimerScreen {
	// Time the break in whole seconds so a fractional earned break (e.g. 0.4 min)
	// is honored exactly instead of being inflated to a full minute.
	secs := p.breakMinutes.Seconds()
	if secs < 1 {
		secs = 1
	}
	tm := pomodoro.NewTimerSeconds(secs)
	tm.Start(root.now())
	return &restTimerScreen{root: root, payload: p, timer: tm}
}

func (r *restTimerScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := r.root.keys
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, km.Quit):
			return r, r.root.quit()
		case keyMatches(msg, km.Back), keyMatches(msg, km.Enter):
			// Skip the rest early. In flow mode this still chains into the next
			// focus (via the same onDone path) instead of dropping to the menu, so
			// the hands-free loop survives an early skip.
			return r, r.skip()
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "p":
			if r.root.core.Bank().Bank() > 0 {
				return r, switchTo(screenGames, nil)
			}
		}

	case tickMsg:
		if r.timer.Tick(r.root.now()) {
			return r, r.onDone()
		}
		if !r.framing {
			r.framing = true
			return r, frameTick()
		}

	case frameMsg:
		target := r.timer.Progress(r.root.now())
		next, more := easeToward(r.shownFrac, target)
		r.shownFrac = next
		if more || r.shownFrac < 1 {
			r.framing = true // own the loop so tickMsg won't start a second one
			return r, frameTick()
		}
		r.framing = false
	}
	return r, nil
}

// skip ends the rest early. In flow mode it still chains into the next focus
// (countdown then timer) so the loop continues hands-free; otherwise it drops to
// the dashboard. No "break over" alert here — the user chose to cut it short.
func (r *restTimerScreen) skip() tea.Cmd {
	if r.root.core.FlowMode() {
		return startCountdown(r.root, screenTimer, taskPayload{task: r.payload.task})
	}
	return switchTo(screenDashboard, nil)
}

// onDone fires the alert and routes: flow mode chains into the next focus
// session (same task); otherwise it returns to the dashboard.
func (r *restTimerScreen) onDone() tea.Cmd {
	if r.root.core.FlowMode() {
		return tea.Batch(
			alert("Break over ☕", "Back to focus — let's go."),
			startCountdown(r.root, screenTimer, taskPayload{task: r.payload.task}),
		)
	}
	return tea.Batch(
		alert("Break over ☕", "Ready when you are."),
		switchTo(screenDashboard, nil),
	)
}

func (r *restTimerScreen) View() string {
	st := r.root.styles
	now := r.root.now()

	remaining := domain.MinutesFromSeconds(r.timer.RemainingSeconds(now))
	header := st.art.Render(bigText(remaining.String(), digitFont))

	label := "SHORT BREAK"
	labelStyle := st.break_
	if r.payload.isLongBreak {
		label = "LONG BREAK"
		labelStyle = st.good // distinct colour so the long break is unmistakable
	}

	frac := r.shownFrac
	if !r.framing {
		frac = r.timer.Progress(now)
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		labelStyle.Render(label),
		"",
		smoothBar(frac, 32, labelStyle, st.help),
		"",
		st.subtitle.Render(r.root.cycleDots()),
	)

	footer := "esc skip • q quit"
	if r.root.core.Bank().Bank() > 0 {
		footer = "p play a game • esc skip • q quit"
	}
	return st.frame(r.root.width, r.root.height, header, body, footer)
}
