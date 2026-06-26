package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// taskPayload carries the chosen task into the timer screen.
type taskPayload struct {
	task string
}

// taskInput asks the user what they will work on before the focus timer starts.
// It is the "выбор задачи" screen from the user flow.
type taskInput struct {
	root  *model
	input textinput.Model
}

func newTaskInput(root *model) *taskInput {
	ti := textinput.New()
	ti.Placeholder = "what are you working on?"
	ti.CharLimit = 60
	ti.Width = 40
	ti.Focus()
	return &taskInput{root: root, input: ti}
}

func (t *taskInput) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		// On a text-input screen only ctrl+c / esc / enter are control keys, so
		// the user can freely type letters like "q" into the task.
		switch key.Type {
		case tea.KeyCtrlC:
			return t, t.root.quit()
		case tea.KeyEsc:
			return t, switchTo(screenDashboard, nil)
		case tea.KeyEnter:
			// Ritual 3-2-1, then start the timer with whatever was typed.
			return t, startCountdown(t.root, screenTimer, taskPayload{task: t.input.Value()})
		}
	}

	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	return t, cmd
}

func (t *taskInput) View() string {
	st := t.root.styles

	header := st.heading.Render("WHAT'S THE FOCUS?")
	return st.frame(t.root.width, t.root.height, header, t.input.View(),
		"enter start • esc cancel • q quit")
}
