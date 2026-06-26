package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/game"
)

// minesCols is the board width and minesTilesCount the total, matching the
// engine's 5×5 board.
const (
	minesCols       = 5
	minesTilesCount = minesCols * minesCols
)

// minesPhase is where the mines screen is in its lifecycle.
type minesPhase int

const (
	minesBetting minesPhase = iota // choosing a stake
	minesPlaying                   // revealing tiles
	minesOver                      // busted or cashed out
)

// minesScreen plays mines: pick a stake, then reveal tiles on a 5×5 grid with a
// cursor, cashing out at any time. It drives the multi-step game engine.
type minesScreen struct {
	root   *model
	phase  minesPhase
	stake  domain.Minutes
	cursor int // 0..24, current grid cell
	g      game.Game
	state  game.GameState
	out    game.Outcome
	err    error
	confettiHost
}

func newMines(root *model) *minesScreen {
	return &minesScreen{root: root, stake: defaultStake(root)}
}

func (m *minesScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := m.root.keys
	if _, ok := msg.(frameMsg); ok {
		return m, m.advanceConfetti()
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case keyMatches(key, km.Quit):
		return m, m.root.quit()
	case keyMatches(key, km.Back):
		return m, switchTo(screenGames, nil)
	}

	switch m.phase {
	case minesBetting:
		return m, m.updateBetting(key)
	case minesPlaying:
		return m, m.updatePlaying(key)
	case minesOver:
		// Enter starts a fresh board; the player stays in mines.
		if keyMatches(key, km.Enter) {
			m.phase = minesBetting
			m.err = nil
			m.cursor = 0
			m.g = nil
			m.state = game.GameState{}
			if bank := m.root.core.Bank().Bank(); m.stake > bank {
				m.stake = bank
			}
			if m.stake < stakeStep {
				m.stake = stakeStep
			}
		}
	}
	return m, nil
}

func (m *minesScreen) updateBetting(key tea.KeyMsg) tea.Cmd {
	km := m.root.keys
	switch {
	case keyMatches(key, km.Left):
		m.stake = adjustStake(m.root, m.stake, false)
	case keyMatches(key, km.Right):
		m.stake = adjustStake(m.root, m.stake, true)
	case keyMatches(key, km.Enter):
		if ok, reason := m.root.core.Bank().CanSpend(m.stake); !ok {
			m.err = fmt.Errorf("%s", reason)
			return nil
		}
		if err := m.root.core.Stake(m.root.wallNow(), domain.GameMines, m.stake); err != nil {
			m.err = err
			return nil
		}
		m.g = m.root.core.NewGame(domain.GameMines, game.NewCryptoRand())
		m.state = m.g.Start(m.stake)
		m.phase = minesPlaying
	}
	return nil
}

func (m *minesScreen) updatePlaying(key tea.KeyMsg) tea.Cmd {
	km := m.root.keys
	switch {
	case keyMatches(key, km.Up):
		if m.cursor >= minesCols {
			m.cursor -= minesCols
		}
	case keyMatches(key, km.Down):
		if m.cursor < minesTilesCount-minesCols {
			m.cursor += minesCols
		}
	case keyMatches(key, km.Left):
		if m.cursor%minesCols > 0 {
			m.cursor--
		}
	case keyMatches(key, km.Right):
		if m.cursor%minesCols < minesCols-1 {
			m.cursor++
		}
	case key.Type == tea.KeyRunes && string(key.Runes) == "c":
		// Cashing out before revealing any gem pays nothing, so the engine would
		// settle a total loss. Ignore the key until at least one gem is found.
		if m.minesPicks() == 0 {
			return nil
		}
		return m.finish(game.Move{CashOut: true})
	case keyMatches(key, km.Enter):
		return m.finish(game.Move{Cell: m.cursor})
	}
	return nil
}

// finish applies a move; if the round ends, it settles through the app core.
func (m *minesScreen) finish(move game.Move) tea.Cmd {
	state, done := m.g.Step(move)
	m.state = state
	if !done {
		return nil
	}
	m.out = m.g.Result()
	m.phase = minesOver
	if err := m.root.core.Settle(m.root.wallNow(), domain.GameMines, m.out); err != nil {
		m.err = err
	}
	if m.err == nil && m.out.Win {
		return m.celebrate(m.root.width) // confetti on a successful cash-out
	}
	return nil
}

func (m *minesScreen) View() string {
	st := m.root.styles
	header := st.heading.Render("MINES")

	lines := []string{
		st.subtitle.Render("Bank " + st.bank.Render(m.root.core.Bank().Bank().String()) + " rest"),
		"",
	}

	switch m.phase {
	case minesBetting:
		lines = append(lines,
			st.subtitle.Render(fmt.Sprintf("%d mines on a 5×5 board", m.root.core.Config().Games.MinesCount)),
			"",
			"Stake "+st.good.Render(m.stake.String()),
		)
		if m.err != nil {
			lines = append(lines, st.bad.Render(m.err.Error()))
		}
		body := lipgloss.JoinVertical(lipgloss.Center, lines...)
		return st.frame(m.root.width, m.root.height, header, body,
			"←/→ stake • enter start • esc back • q quit")

	case minesPlaying:
		picks := m.minesPicks()
		var cashBtn, footer string
		if picks == 0 {
			// No gem revealed yet: cashing out would pay nothing, so show the
			// button disabled (dim) and drop "c" from the help line.
			cashBtn = st.subtitle.Render("  CASH OUT  (reveal a gem first)  ")
			footer = "↑↓←→ move • enter reveal • esc back"
		} else {
			cashValue := domain.Minutes(float64(m.stake) * m.state.Multiplier)
			cashBtn = st.selected.Render(fmt.Sprintf("  CASH OUT  ×%.2f = %s  ",
				m.state.Multiplier, cashValue.String()))
			footer = "↑↓←→ move • enter reveal • c cash out • esc back"
		}
		lines = append(lines,
			m.grid(true),
			"",
			st.subtitle.Render(fmt.Sprintf("%d gems found", picks)),
			"",
			cashBtn,
		)
		body := lipgloss.JoinVertical(lipgloss.Center, lines...)
		return st.frame(m.root.width, m.root.height, header, body, footer)

	default: // minesOver
		lines = append(lines, m.grid(false), "")
		switch {
		case m.err != nil:
			lines = append(lines, st.bad.Render("error: "+m.err.Error()))
		case m.out.Win:
			lines = append(lines, st.good.Render(fmt.Sprintf("WIN ×%.2f  +%s",
				m.out.Multiplier, m.out.Payout.String())))
		default:
			lines = append(lines, st.bad.Render("BOOM — stake lost"))
		}
		body := lipgloss.JoinVertical(lipgloss.Center, lines...)
		out := st.frame(m.root.width, m.root.height, header, body, "enter play again • esc games • q quit")
		return m.renderConfetti(out, m.root.width, m.root.height)
	}
}

// minesPicks counts the revealed gems on the board.
func (m *minesScreen) minesPicks() int {
	n := 0
	for _, sym := range m.state.Symbols {
		if sym == "💎" {
			n++
		}
	}
	return n
}

// grid renders the 5×5 board. When showCursor is true the current cell is framed
// with brackets so it stands out around the emoji tile.
func (m *minesScreen) grid(showCursor bool) string {
	syms := m.state.Symbols
	var b strings.Builder
	for i, sym := range syms {
		cell := " " + sym + " "
		if showCursor && i == m.cursor {
			cell = m.root.styles.good.Render("[" + sym + "]")
		}
		b.WriteString(cell)
		if (i+1)%minesCols == 0 {
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
