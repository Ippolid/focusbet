package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Palette. A small, calm set: focus is green, money/play is gold, breaks are
// blue, warnings red. Kept as named vars so screens share one look.
var (
	colorFocus  = lipgloss.Color("42")  // green
	colorGold   = lipgloss.Color("220") // bank / play
	colorBreak  = lipgloss.Color("39")  // blue
	colorMuted  = lipgloss.Color("245") // secondary text
	colorDanger = lipgloss.Color("203") // red
	colorText   = lipgloss.Color("231") // bright foreground
	colorBg     = lipgloss.Color("236") // subtle panel bg
)

// styles bundles the reusable lipgloss styles. One value is built at startup and
// passed down so screens don't each re-declare them.
type styles struct {
	title    lipgloss.Style
	art      lipgloss.Style // big ASCII-art brand / timer
	heading  lipgloss.Style // per-screen heading
	subtitle lipgloss.Style
	bank     lipgloss.Style
	help     lipgloss.Style
	selected lipgloss.Style
	item     lipgloss.Style
	good     lipgloss.Style
	bad      lipgloss.Style
	break_   lipgloss.Style
}

// newStyles builds the shared style set.
func newStyles() styles {
	return styles{
		title: lipgloss.NewStyle().
			Bold(true).Foreground(colorFocus).MarginBottom(1),
		art: lipgloss.NewStyle().
			Bold(true).Foreground(colorFocus),
		heading: lipgloss.NewStyle().
			Bold(true).Foreground(colorText).
			Padding(0, 2).MarginBottom(1),
		subtitle: lipgloss.NewStyle().
			Foreground(colorMuted),
		bank: lipgloss.NewStyle().
			Bold(true).Foreground(colorGold),
		help: lipgloss.NewStyle().
			Foreground(colorMuted),
		// selected: bright pill on a focus-green background — the single, loud
		// selection cue every screen uses, so navigation feels identical.
		selected: lipgloss.NewStyle().
			Bold(true).Foreground(lipgloss.Color("16")).Background(colorFocus).
			Padding(0, 2),
		item: lipgloss.NewStyle().
			Foreground(colorText).Padding(0, 2),
		good:   lipgloss.NewStyle().Bold(true).Foreground(colorFocus),
		bad:    lipgloss.NewStyle().Bold(true).Foreground(colorDanger),
		break_: lipgloss.NewStyle().Bold(true).Foreground(colorBreak),
	}
}

// frame is the one layout every screen renders through, so the whole app shares
// a structure: optional header art/heading, a body, and a dim footer — all
// stacked centered and placed in the middle of the terminal. Passing "" for any
// part omits it.
func (s styles) frame(w, h int, header, body, footer string) string {
	parts := make([]string, 0, 5)
	if header != "" {
		parts = append(parts, header, "")
	}
	if body != "" {
		parts = append(parts, body)
	}
	if footer != "" {
		parts = append(parts, "", s.help.Render(footer))
	}
	content := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return center(w, h, content)
}

// menu renders a vertical list of choices with the cursor-th one highlighted,
// centered as a block. It is the shared widget for every selectable list.
func (s styles) menu(labels []string, cursor int) string {
	lines := make([]string, len(labels))
	for i, label := range labels {
		if i == cursor {
			lines[i] = s.selected.Render(label)
		} else {
			lines[i] = s.item.Render(label)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

// center places content in the middle of a w×h area. When the terminal size is
// unknown yet (0), it returns the content unchanged so the first frame still
// renders.
func center(w, h int, content string) string {
	if w <= 0 || h <= 0 {
		return content
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}
