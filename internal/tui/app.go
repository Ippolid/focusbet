// Package tui is the terminal UI: a bubbletea root model that runs a screen
// state machine over the app core. The root owns shared state (the app.App, the
// clock, styles, keys) and delegates Update/View to whichever screen is active.
// Screens never touch the bank directly — they call app.App, which routes every
// currency change through the balance core.
package tui

import (
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ippolid/focusbet/internal/app"
)

// screen identifies the active sub-model, mirroring the user-flow diagram.
type screen int

const (
	screenOnboarding screen = iota
	screenDashboard
	screenTaskInput
	screenTimer
	screenResult
	screenRest
	screenRestTimer
	screenCountdown
	screenGames
	screenPlay
	screenRoulette
	screenMines
	screenStats
	screenSettings
)

// clock returns the current unix-seconds time. It is a field on the model so
// tests can drive time deterministically.
type clock func() int64

// newClock returns the wall-clock, optionally sped up for testing. Setting
// FOCUSBET_TIME_SCALE to an integer N > 1 makes time run N× faster, so a 25-min
// session finishes in ~25/N minutes. Anchored at the first real second so it
// stays monotonic. Invalid or unset values fall back to real time.
func newClock() clock {
	scale, err := strconv.Atoi(os.Getenv("FOCUSBET_TIME_SCALE"))
	if err != nil || scale <= 1 {
		return func() int64 { return time.Now().Unix() }
	}
	t0 := time.Now().Unix()
	return func() int64 {
		return t0 + (time.Now().Unix()-t0)*int64(scale)
	}
}

// screenModel is the contract every screen satisfies. It mirrors tea.Model but
// returns the concrete type so the root can hold screens uniformly while each
// keeps its own Update signature simple.
type screenModel interface {
	Update(msg tea.Msg) (screenModel, tea.Cmd)
	View() string
}

// switchMsg asks the root to change the active screen. Screens emit it (wrapped
// in a tea.Cmd) instead of mutating root state, keeping transitions in one place.
type switchMsg struct {
	to      screen
	payload any // optional data handed to the destination screen
}

func switchTo(to screen, payload any) tea.Cmd {
	return func() tea.Msg { return switchMsg{to: to, payload: payload} }
}

// tickMsg is the once-per-second timer pulse.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// model is the bubbletea root model.
type model struct {
	core    *app.App
	now     clock
	styles  styles
	keys    keyMap
	width   int
	height  int
	current screen
	screen  screenModel

	// cycle tracks pomodoro progress across screens (each screen is recreated, so
	// the count lives here). focusDone is completed focus sessions in the current
	// cycle; a long break fires once it reaches the preset cycle length.
	focusDone int
}

// cycleLen is the number of focus sessions before a long break. It goes through
// the app's resolved timings (not the raw config field), so a built-in preset
// like "deep" reports its own cycle length instead of a stale stored value.
func (m *model) cycleLen() int {
	n := m.core.CycleLength()
	if n < 1 {
		n = 4
	}
	return n
}

// onFocusComplete advances the cycle and reports whether the break that follows
// is the long one (cycle just completed).
func (m *model) onFocusComplete() (isLong bool) {
	m.focusDone++
	if m.focusDone >= m.cycleLen() {
		m.focusDone = 0
		return true
	}
	return false
}

// cycleDots renders ●●●○ progress for the current cycle (filled = focus done).
func (m *model) cycleDots() string {
	n := m.cycleLen()
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i < m.focusDone {
			b.WriteString("● ")
		} else {
			b.WriteString("○ ")
		}
	}
	return strings.TrimSpace(b.String())
}

// New builds the root model over the app core. It returns a *model so there is a
// single stable instance: screens hold a pointer back to it, so updates to shared
// state (terminal size) are visible to every screen.
func New(core *app.App) *model {
	m := &model{
		core:   core,
		now:    newClock(),
		styles: newStyles(),
		keys:   defaultKeys(),
	}
	if core.IsFirstRun() {
		m.current = screenOnboarding
		m.screen = newOnboarding(m)
	} else {
		m.current = screenDashboard
		m.screen = newDashboard(m)
	}
	return m
}

// wallNow returns the real wall-clock unix time, ignoring FOCUSBET_TIME_SCALE.
// Persisted timestamps (money log, state, streak day) must use real time so a
// sped-up test session doesn't write future-dated entries or skew the streak;
// only the on-screen timers/countdowns run on the scaled clock (m.now).
func (m *model) wallNow() int64 { return time.Now().Unix() }

// Init starts the timer pulse and one animation frame so the first screen's
// entrance animation plays immediately instead of waiting for the first switch.
func (m *model) Init() tea.Cmd { return tea.Batch(tick(), frameTick()) }

// Update routes messages: global quit and screen switches are handled here; the
// timer pulse and everything else are forwarded to the active screen.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// fall through so screens can react to size too

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, m.quit()
		}

	case switchMsg:
		m.current = msg.to
		m.screen = m.build(msg.to, msg.payload)
		// Kick a frame so screens that animate their entrance (reveal) start
		// immediately instead of waiting up to a second for the first pulse.
		return m, frameTick()

	case tickMsg:
		// Re-arm the pulse and let the active screen advance.
		next := tick()
		sm, cmd := m.screen.Update(msg)
		m.screen = sm
		return m, tea.Batch(next, cmd)

	case quitMsg:
		return m, tea.Quit
	}

	sm, cmd := m.screen.Update(msg)
	m.screen = sm
	return m, cmd
}

// View delegates to the active screen.
func (m *model) View() string { return m.screen.View() }

// quitMsg signals the program to terminate after a graceful save.
type quitMsg struct{}

// quit saves state then asks bubbletea to exit.
func (m model) quit() tea.Cmd {
	_ = m.core.Save(m.wallNow())
	return func() tea.Msg { return quitMsg{} }
}

// build constructs the destination screen, passing the payload where relevant.
func (m *model) build(to screen, payload any) screenModel {
	switch to {
	case screenOnboarding:
		return newOnboarding(m)
	case screenDashboard:
		return newDashboard(m)
	case screenTaskInput:
		return newTaskInput(m)
	case screenTimer:
		tp, _ := payload.(taskPayload)
		return newTimer(m, tp)
	case screenResult:
		r, _ := payload.(resultPayload)
		return newResult(m, r)
	case screenRest:
		r, _ := payload.(resultPayload)
		return newRest(m, r)
	case screenRestTimer:
		r, _ := payload.(resultPayload)
		return newRestTimer(m, r)
	case screenCountdown:
		c, _ := payload.(countdownPayload)
		return newCountdown(m, c)
	case screenGames:
		return newGames(m)
	case screenPlay:
		k, _ := payload.(playPayload)
		return newPlay(m, k)
	case screenRoulette:
		return newRoulette(m)
	case screenMines:
		return newMines(m)
	case screenStats:
		return newStats(m)
	case screenSettings:
		return newSettings(m)
	default:
		return newDashboard(m)
	}
}
