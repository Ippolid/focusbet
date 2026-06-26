package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// countdownPayload tells the countdown screen what to launch once it reaches 0.
type countdownPayload struct {
	dest    screen
	payload any
}

// startCountdown switches to the 3-2-1 countdown, which then launches dest with
// payload. It is the small ritual before a focus or break starts, so the moment
// of beginning is deliberate rather than abrupt.
func startCountdown(_ *model, dest screen, payload any) tea.Cmd {
	return switchTo(screenCountdown, countdownPayload{dest: dest, payload: payload})
}

// countdownScreen shows a big 3 → 2 → 1 before launching the destination screen.
// It advances on the global one-second tickMsg the root already pumps, so no
// separate timer is needed.
type countdownScreen struct {
	root *model
	dest screen
	pay  any
	n    int
}

func newCountdown(root *model, p countdownPayload) *countdownScreen {
	dest := p.dest
	if dest == 0 {
		dest = screenTimer
	}
	return &countdownScreen{root: root, dest: dest, pay: p.payload, n: 3}
}

func (c *countdownScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := c.root.keys
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, km.Quit):
			return c, c.root.quit()
		case keyMatches(msg, km.Back):
			return c, switchTo(screenDashboard, nil)
		case keyMatches(msg, km.Enter):
			// Skip the countdown and start immediately.
			return c, switchTo(c.dest, c.pay)
		}

	case tickMsg:
		c.n--
		if c.n <= 0 {
			return c, switchTo(c.dest, c.pay)
		}
	}
	return c, nil
}

func (c *countdownScreen) View() string {
	st := c.root.styles

	header := st.art.Render(bigText(itoa(c.n), digitFont))
	body := lipgloss.JoinVertical(lipgloss.Center,
		st.subtitle.Render("starting…"),
	)
	return st.frame(c.root.width, c.root.height, header, body, "enter skip • esc cancel")
}

// itoa is a tiny int→string for the single countdown digit.
func itoa(n int) string {
	if n < 0 {
		n = 0
	}
	return string(rune('0' + n))
}
