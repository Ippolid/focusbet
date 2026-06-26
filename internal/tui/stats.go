package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ippolid/focusbet/internal/domain"
)

// stats shows the three slices from the user flow: bank/money, work sessions,
// and game results — formatted for readability (hours/minutes, percentages).
type statsScreen struct {
	root *model
}

func newStats(root *model) *statsScreen { return &statsScreen{root: root} }

func (s *statsScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := s.root.keys
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case keyMatches(key, km.Quit):
			return s, s.root.quit()
		case keyMatches(key, km.Back), keyMatches(key, km.Enter):
			return s, switchTo(screenDashboard, nil)
		}
	}
	return s, nil
}

// statLabelWidth aligns the value column across every row on the screen.
const statLabelWidth = 16

func (s *statsScreen) View() string {
	st := s.root.styles
	d := s.root.core.Stats()
	bank := s.root.core.Bank().Bank()

	var b strings.Builder

	// 1) Bank / money.
	b.WriteString(s.section(st.bank.Render("◆ Bank")))
	b.WriteString(s.row("Current bank", st.bank.Render(bank.Human())))
	b.WriteString("\n")

	// 2) Work sessions.
	b.WriteString(s.section(st.good.Render("◆ Work")))
	b.WriteString(s.row("Sessions done", fmt.Sprintf("%d", d.TimerCount)))
	b.WriteString(s.row("Focused total", domain.MinutesFromSeconds(d.WorkSeconds).Human()))
	b.WriteString(s.row("Rest taken", domain.MinutesFromSeconds(d.RestSeconds).Human()))
	b.WriteString(s.row("Streak", fmt.Sprintf("%d %s", d.StreakDays, plural(int64(d.StreakDays), "day", "days"))))
	b.WriteString("\n")

	// 3) Games.
	b.WriteString(s.section(st.break_.Render("◆ Games")))
	b.WriteString(s.row("Played", fmt.Sprintf("%d", d.GameCount)))
	b.WriteString(s.row("Win rate", winRate(d.Wins, d.GameCount)))
	b.WriteString(s.row("Wins / losses", fmt.Sprintf("%d / %d", d.Wins, d.Loses)))
	b.WriteString(s.row("Minutes won", domain.MinutesFromSeconds(d.WonSeconds).Human()))
	b.WriteString(s.row("Best multiplier", bestMultiplier(d.BestMultiplier)))

	header := st.heading.Render("STATISTICS")
	return st.frame(s.root.width, s.root.height, header, b.String(), "esc back • q quit")
}

// section renders a group heading with spacing above it.
func (s *statsScreen) section(title string) string {
	return "\n" + title + "\n"
}

// row renders one "label ........ value" line with the value column aligned.
func (s *statsScreen) row(label, value string) string {
	return fmt.Sprintf("  %-*s %s\n", statLabelWidth, label, value)
}

// winRate renders wins as a percentage of games played, "—" when none played.
func winRate(wins, played int64) string {
	if played <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", float64(wins)/float64(played)*100)
}

// bestMultiplier renders the best game multiplier, "—" when no game was won.
func bestMultiplier(m float64) string {
	if m <= 0 {
		return "—"
	}
	return fmt.Sprintf("×%.2f", m)
}

// plural picks the singular or plural form for a count.
func plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
