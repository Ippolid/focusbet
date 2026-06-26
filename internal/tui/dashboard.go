package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// dashboardItem is a menu entry on the main screen.
type dashboardItem struct {
	label string
	to    screen
}

var dashboardItems = []dashboardItem{
	{"Start session", screenTaskInput},
	{"Games", screenGames},
	{"Stats", screenStats},
	{"Settings", screenSettings},
}

// dashboard is the hub screen: bank, streak, cycle dots and the main menu.
type dashboard struct {
	root   *model
	cursor int
	intro  reveal
}

func newDashboard(root *model) *dashboard {
	return &dashboard{root: root}
}

func (d *dashboard) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := d.root.keys
	switch msg := msg.(type) {
	case frameMsg:
		if d.intro.step() {
			return d, frameTick()
		}
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, km.Quit):
			return d, d.root.quit()
		case keyMatches(msg, km.Up):
			if d.cursor > 0 {
				d.cursor--
			}
		case keyMatches(msg, km.Down):
			if d.cursor < len(dashboardItems)-1 {
				d.cursor++
			}
		case keyMatches(msg, km.Enter):
			return d, switchTo(dashboardItems[d.cursor].to, nil)
		}
	}
	return d, nil
}

func (d *dashboard) View() string {
	st := d.root.styles

	labels := make([]string, len(dashboardItems))
	for i, it := range dashboardItems {
		labels[i] = it.label
	}

	header := st.art.Render(d.intro.body(bigText("FOCUSBET", brandFont)))
	stats := d.root.core.Stats()
	bank := d.root.core.Bank().Bank()
	body := lipgloss.JoinVertical(lipgloss.Center,
		st.menu(labels, d.cursor),
		"",
		st.subtitle.Render(fmt.Sprintf("Bank %s rest  •  Streak %d  •  Sessions %d",
			bank.String(), stats.StreakDays, stats.TimerCount)),
	)
	footer := "↑/↓ move • enter select • q quit"
	return st.frame(d.root.width, d.root.height, header, body, footer)
}
