package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// keyMatches reports whether a key message matches a binding.
func keyMatches(msg tea.KeyMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}

// keyMap holds every binding the app uses. Screens read the ones they need; the
// help line renders from the same source so hints never drift from behaviour.
type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
	Pause key.Binding
	Stop  key.Binding
	Help  key.Binding
}

// defaultKeys returns the standard bindings.
func defaultKeys() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down: key.NewBinding(
			key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left: key.NewBinding(
			key.WithKeys("left", "h"), key.WithHelp("←/h", "less")),
		Right: key.NewBinding(
			key.WithKeys("right", "l"), key.WithHelp("→/l", "more")),
		Enter: key.NewBinding(
			key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Back: key.NewBinding(
			key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
		Pause: key.NewBinding(
			key.WithKeys("p", " "), key.WithHelp("p", "pause")),
		Stop: key.NewBinding(
			key.WithKeys("s"), key.WithHelp("s", "stop")),
		Help: key.NewBinding(
			key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}
