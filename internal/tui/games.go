package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ippolid/focusbet/internal/domain"
)

// gameChoice is a game in the lobby. available=false renders as "coming soon".
type gameChoice struct {
	kind      domain.GameKind
	label     string
	available bool
}

var gameChoices = []gameChoice{
	{domain.GameSlots, "Slots — 3-reel, weighted", true},
	{domain.GameRoulette, "Roulette — European wheel", true},
	{domain.GameMines, "Mines — 5×5 board", true},
}

// gameScreen maps a game kind to the screen that plays it.
func gameScreen(kind domain.GameKind) screen {
	switch kind {
	case domain.GameRoulette:
		return screenRoulette
	case domain.GameMines:
		return screenMines
	default:
		return screenPlay
	}
}

// playPayload tells the play screen which game to run.
type playPayload struct {
	kind domain.GameKind
}

// games is the lobby: pick a game to play.
type games struct {
	root   *model
	cursor int
}

func newGames(root *model) *games {
	return &games{root: root}
}

func (g *games) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := g.root.keys
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case keyMatches(key, km.Quit):
			return g, g.root.quit()
		case keyMatches(key, km.Back):
			return g, switchTo(screenDashboard, nil)
		case keyMatches(key, km.Up):
			if g.cursor > 0 {
				g.cursor--
			}
		case keyMatches(key, km.Down):
			if g.cursor < len(gameChoices)-1 {
				g.cursor++
			}
		case keyMatches(key, km.Enter):
			c := gameChoices[g.cursor]
			if c.available {
				return g, switchTo(gameScreen(c.kind), playPayload{kind: c.kind})
			}
		}
	}
	return g, nil
}

func (g *games) View() string {
	st := g.root.styles

	lines := make([]string, len(gameChoices))
	for i, c := range gameChoices {
		switch {
		case i == g.cursor && c.available:
			lines[i] = st.selected.Render(c.label)
		case !c.available:
			lines[i] = st.subtitle.Render(c.label)
		default:
			lines[i] = st.item.Render(c.label)
		}
	}

	header := st.heading.Render("GAMES")
	body := lipgloss.JoinVertical(lipgloss.Center,
		st.subtitle.Render("Bank "+st.bank.Render(g.root.core.Bank().Bank().String())+" rest"),
		"",
		lipgloss.JoinVertical(lipgloss.Center, lines...),
	)
	return st.frame(g.root.width, g.root.height, header, body, "↑/↓ move • enter play • esc menu • q quit")
}
