package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/pomodoro"
)

// resultPayload carries a completed focus session into the result/rest screens.
// The earnings were already settled by app.CompleteFocus; this is for display
// and to drive the fork.
type resultPayload struct {
	focused     domain.Minutes
	reward      pomodoro.Reward
	banked      domain.Minutes
	interrupted bool
	err         error

	// task is the focus task, carried so flow mode can resume it after the break.
	task string
	// breakMinutes is how long the upcoming break is; isLongBreak marks the long
	// break that follows a full cycle.
	breakMinutes domain.Minutes
	isLongBreak  bool
}

// resultChoice is one branch of the post-focus fork. payload is what gets handed
// to the destination screen; desc is a one-line explanation shown under it.
type resultChoice struct {
	label   string
	desc    string
	to      screen
	payload any
}

// result is the prominent fork shown when a focus session ends: continue with
// another focus, take the rest, or gamble the surplus. The rest option is the
// default (cursor starts on it) — the safe, expected choice.
type result struct {
	root    *model
	payload resultPayload
	cursor  int
	choices []resultChoice
	intro   reveal
}

func newResult(root *model, p resultPayload) *result {
	choices := []resultChoice{
		{
			label: "Continue focus",
			desc:  "skip the break and start another session",
			to:    screenTaskInput,
		},
		{
			label:   "Rest now",
			desc:    "take a break, spending banked minutes",
			to:      screenRest,
			payload: p,
		},
	}
	// Offer play only when there are banked minutes to wager.
	if root.core.Bank().Bank() > 0 {
		choices = append(choices, resultChoice{
			label: "Play a game",
			desc:  "risk banked minutes in a game (the house edge means you usually lose)",
			to:    screenGames,
		})
	}
	// Default the cursor to "Rest now".
	return &result{root: root, payload: p, choices: choices, cursor: 1}
}

func (r *result) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := r.root.keys
	if _, ok := msg.(frameMsg); ok {
		if r.intro.step() {
			return r, frameTick()
		}
		return r, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case keyMatches(key, km.Quit):
			return r, r.root.quit()
		case keyMatches(key, km.Back):
			return r, switchTo(screenDashboard, nil)
		case keyMatches(key, km.Up):
			if r.cursor > 0 {
				r.cursor--
			}
		case keyMatches(key, km.Down):
			if r.cursor < len(r.choices)-1 {
				r.cursor++
			}
		case keyMatches(key, km.Enter):
			c := r.choices[r.cursor]
			return r, switchTo(c.to, c.payload)
		}
	}
	return r, nil
}

func (r *result) View() string {
	st := r.root.styles

	head := "FOCUS DONE"
	if r.payload.interrupted {
		head = "STOPPED EARLY"
	}
	header := st.heading.Render(r.intro.header(head)) // typewriter on entrance

	// One-line summary of what this session produced.
	summary := st.subtitle.Render(fmt.Sprintf(
		"You focused %s and earned a %s break.  Bank: %s.",
		r.payload.focused.String(),
		r.payload.reward.Fair.String(),
		r.root.core.Bank().Bank().String(),
	))

	parts := []string{summary, ""}
	if r.payload.err != nil {
		parts = append(parts, st.bad.Render("error settling: "+r.payload.err.Error()), "")
	}

	// Each choice: the (highlightable) label, then a dim one-line explanation.
	for i, c := range r.choices {
		label := st.item.Render(c.label)
		if i == r.cursor {
			label = st.selected.Render(c.label)
		}
		parts = append(parts, label, st.subtitle.Render(c.desc), "")
	}

	body := r.intro.body(lipgloss.JoinVertical(lipgloss.Center, parts...))
	return st.frame(r.root.width, r.root.height, header, body, "↑/↓ move • enter select • esc menu • q quit")
}
