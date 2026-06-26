package tui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ippolid/focusbet/internal/domain"
)

// errEmptyRest is shown when the player confirms a zero-minute rest.
var errEmptyRest = errors.New("choose at least 1 minute (→ to add)")

// restScreen lets the player spend banked minutes on extra rest beyond the free
// fair break: a simple left/right slider bounded by the bank.
type restScreen struct {
	root    *model
	payload resultPayload
	amount  domain.Minutes // minutes chosen to spend
	done    bool
	err     error
}

// restStep is how much one left/right press changes the amount.
const restStep = domain.Minutes(1)

func newRest(root *model, p resultPayload) *restScreen {
	// Start the slider at a small non-zero amount (capped to the bank) so Enter
	// does something immediately instead of silently doing nothing.
	amount := restStep
	if bank := root.core.Bank().Bank(); amount > bank {
		amount = bank
	}
	return &restScreen{root: root, payload: p, amount: amount}
}

func (r *restScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := r.root.keys
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case keyMatches(key, km.Quit):
			return r, r.root.quit()
		case keyMatches(key, km.Back):
			return r, switchTo(screenDashboard, nil)
		case keyMatches(key, km.Left):
			// Clamp to zero: a fractional starting amount (capped to a fractional
			// bank) minus a whole step could otherwise go negative and render as a
			// negative "-00:30" time.
			if r.amount > 0 {
				r.amount -= restStep
				if r.amount < 0 {
					r.amount = 0
				}
			}
		case keyMatches(key, km.Right):
			if r.amount+restStep <= r.root.core.Bank().Bank() {
				r.amount += restStep
			}
		case keyMatches(key, km.Enter):
			return r, r.confirm()
		}
	}
	return r, nil
}

func (r *restScreen) confirm() tea.Cmd {
	if r.amount <= 0 {
		r.err = errEmptyRest
		return nil
	}
	if err := r.root.core.Bank().SpendOnRest(r.root.wallNow(), r.amount); err != nil {
		r.err = err
		return nil
	}
	r.done = true
	// Spend committed; run a rest countdown for the chosen minutes.
	p := r.payload
	p.breakMinutes = r.amount
	p.isLongBreak = false
	return switchTo(screenRestTimer, p)
}

func (r *restScreen) View() string {
	st := r.root.styles
	var b strings.Builder

	b.WriteString(st.subtitle.Render("Spend banked minutes for a longer break.") + "\n\n")

	bank := r.root.core.Bank().Bank()
	b.WriteString("Bank available: " + st.bank.Render(bank.String()) + "\n")
	b.WriteString("Spend on rest:  " + st.good.Render(r.amount.String()) + "\n")
	b.WriteString(r.slider(bank))

	if r.err != nil {
		b.WriteString("\n\n" + st.bad.Render("error: "+r.err.Error()))
	}

	header := st.heading.Render("TOP UP REST")
	return st.frame(r.root.width, r.root.height, header, b.String(),
		"←/→ adjust • enter confirm • esc cancel • q quit")
}

// slider renders a simple bar of the chosen fraction of the bank.
func (r *restScreen) slider(bank domain.Minutes) string {
	const width = 24
	frac := 0.0
	if bank > 0 {
		frac = float64(r.amount) / float64(bank)
	}
	// Clamp so an amount above the bank (or rounding) can't drive filled out of
	// [0, width] and crash strings.Repeat with a negative count.
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * width)
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}
