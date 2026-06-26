package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// presetChoice is a selectable preset on the onboarding screen.
type presetChoice struct {
	key   string
	label string
}

var presetChoices = []presetChoice{
	{"classic", "Classic — 25 min focus / 5 min break"},
	{"deep", "Deep — 50 min focus / 10 min break"},
	{"desktime", "DeskTime — 52 min focus / 17 min break"},
	{"short", "Short — 15 min focus / 3 min break"},
	{"custom", "Custom — set your own focus / break"},
}

// onboarding is the first-run screen: it explains the idea and picks a preset.
type onboarding struct {
	root   *model
	cursor int
}

func newOnboarding(root *model) *onboarding {
	// Start the cursor on the currently configured preset, if it matches.
	cur := 0
	for i, p := range presetChoices {
		if p.key == root.core.Config().Pomodoro.Preset {
			cur = i
			break
		}
	}
	return &onboarding{root: root, cursor: cur}
}

func (o *onboarding) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := o.root.keys
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case keyMatches(key, km.Quit):
			return o, o.root.quit()
		case keyMatches(key, km.Up):
			if o.cursor > 0 {
				o.cursor--
			}
		case keyMatches(key, km.Down):
			if o.cursor < len(presetChoices)-1 {
				o.cursor++
			}
		case keyMatches(key, km.Enter):
			_ = o.root.core.SetPreset(presetChoices[o.cursor].key)
			return o, switchTo(screenDashboard, nil)
		}
	}
	return o, nil
}

func (o *onboarding) View() string {
	st := o.root.styles

	header := st.art.Render(bigText("FOCUSBET", brandFont))

	labels := make([]string, len(presetChoices))
	for i, p := range presetChoices {
		labels[i] = p.label
	}
	body := lipgloss.JoinVertical(lipgloss.Center,
		st.subtitle.Render(
			"Focus to earn rest. Bank the fair break — or gamble the surplus,\n"+
				"never risking your guaranteed minimum. Pick a rhythm to start:"),
		"",
		st.menu(labels, o.cursor),
	)
	return st.frame(o.root.width, o.root.height, header, body, "↑/↓ choose • enter confirm • q quit")
}
