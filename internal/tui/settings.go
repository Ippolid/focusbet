package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// settingsScreen lets the user change the session preset, set custom focus/break
// lengths when the custom preset is active, and review the economy config.
type settingsScreen struct {
	root   *model
	cursor int
	err    error // last persist error, surfaced so a failed save isn't silent
}

func newSettings(root *model) *settingsScreen { return &settingsScreen{root: root} }

// settingsItem is one navigable row, with the action its arrow keys perform.
type settingsItem struct {
	label  string
	value  string
	adjust func(s *settingsScreen, up bool) error // nil for read-only rows
}

// rows builds the current row set. The focus/break rows appear only for the
// custom preset; otherwise the preset's fixed timing is shown inline.
func (s *settingsScreen) rows() []settingsItem {
	cfg := s.root.core.Config()
	focus, brk := s.root.core.Timings()
	isCustom := cfg.Pomodoro.Preset == "custom"

	presetValue := cfg.Pomodoro.Preset
	if !isCustom {
		presetValue = fmt.Sprintf("%s  (%d/%d min)", cfg.Pomodoro.Preset, focus, brk)
	}

	items := []settingsItem{
		{"Session preset", presetValue, (*settingsScreen).cyclePreset},
	}
	if isCustom {
		items = append(items,
			settingsItem{"  Focus length", fmt.Sprintf("%d min", focus), (*settingsScreen).adjustFocus},
			settingsItem{"  Break length", fmt.Sprintf("%d min", brk), (*settingsScreen).adjustBreak},
			settingsItem{"  Sessions to long break", fmt.Sprintf("%d", s.root.core.CycleLength()), (*settingsScreen).adjustCycle},
			settingsItem{"  Long break length", fmt.Sprintf("%d min", s.root.core.LongBreakMinutes()), (*settingsScreen).adjustLongBreak},
		)
	}

	flowValue := "off (ask after focus)"
	if s.root.core.FlowMode() {
		flowValue = "on (auto-rest)"
	}
	items = append(items,
		settingsItem{"Flow mode", flowValue, (*settingsScreen).toggleFlow},
		settingsItem{"Import statistics", "coming soon", nil},
	)
	return items
}

// toggleFlow flips flow mode regardless of arrow direction.
func (s *settingsScreen) toggleFlow(bool) error {
	return s.root.core.SetFlowMode(!s.root.core.FlowMode())
}

func (s *settingsScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := s.root.keys
	if key, ok := msg.(tea.KeyMsg); ok {
		rows := s.rows()
		switch {
		case keyMatches(key, km.Quit):
			return s, s.root.quit()
		case keyMatches(key, km.Back):
			return s, switchTo(screenDashboard, nil)
		case keyMatches(key, km.Up):
			if s.cursor > 0 {
				s.cursor--
			}
		case keyMatches(key, km.Down):
			if s.cursor < len(rows)-1 {
				s.cursor++
			}
		case keyMatches(key, km.Left):
			if f := rows[s.cursor].adjust; f != nil {
				s.err = f(s, false)
			}
		case keyMatches(key, km.Right), keyMatches(key, km.Enter):
			if f := rows[s.cursor].adjust; f != nil {
				s.err = f(s, true)
			}
		}
		// Clamp the cursor in case the row count shrank (e.g. left custom).
		if s.cursor >= len(s.rows()) {
			s.cursor = len(s.rows()) - 1
		}
	}
	return s, nil
}

func (s *settingsScreen) cyclePreset(up bool) error {
	cur := s.root.core.Config().Pomodoro.Preset
	idx := 0
	for i, p := range presetChoices {
		if p.key == cur {
			idx = i
			break
		}
	}
	if up {
		idx = (idx + 1) % len(presetChoices)
	} else {
		idx = (idx - 1 + len(presetChoices)) % len(presetChoices)
	}
	return s.root.core.SetPreset(presetChoices[idx].key)
}

func (s *settingsScreen) adjustFocus(up bool) error {
	f, _ := s.root.core.Timings()
	return s.root.core.SetCustomFocus(f + step(up))
}

func (s *settingsScreen) adjustBreak(up bool) error {
	_, b := s.root.core.Timings()
	return s.root.core.SetCustomBreak(b + step(up))
}

func (s *settingsScreen) adjustCycle(up bool) error {
	return s.root.core.SetCustomCycleLength(s.root.core.CycleLength() + step(up))
}

func (s *settingsScreen) adjustLongBreak(up bool) error {
	return s.root.core.SetCustomLongBreak(s.root.core.LongBreakMinutes() + step(up))
}

func step(up bool) int {
	if up {
		return 1
	}
	return -1
}

func (s *settingsScreen) View() string {
	st := s.root.styles
	cfg := s.root.core.Config()
	var b strings.Builder

	for i, r := range s.rows() {
		label := st.item.Render(r.label)
		if i == s.cursor {
			label = st.selected.Render(r.label)
		}
		b.WriteString(label + "   " + st.good.Render(r.value) + "\n")
	}

	b.WriteString("\n" + st.subtitle.Render(fmt.Sprintf(
		"Bank cap %d min  •  games RTP %.0f%% (the house edge)",
		cfg.Economy.BankCapMinutes, cfg.Games.RTP*100,
	)))

	if s.err != nil {
		b.WriteString("\n" + st.bad.Render("could not save: "+s.err.Error()))
	}

	header := st.heading.Render("SETTINGS")
	return st.frame(s.root.width, s.root.height, header, b.String(),
		"↑/↓ row • ←/→ adjust • esc back • q quit")
}
