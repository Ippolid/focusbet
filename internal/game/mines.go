package game

import (
	"strings"

	"github.com/Ippolid/focusbet/internal/domain"
)

// minesTiles is the board size: a 5×5 grid.
const minesTiles = 25

// minesRTP is the target return-to-player. Multipliers are the fair payout
// scaled by this, so the bank trends down at the configured edge.
const minesRTP = 0.90

// minesCeiling maps a multiplier to game_fraction 1.0. Clearing the board is
// astronomically unlikely, so we treat a healthy mid-game cash-out as a full
// win for rest purposes rather than the theoretical maximum.
const minesCeiling = 10.0

// Mines is a 5×5 minesweeper-style game. It is multi-step: each Step reveals a
// tile (Move.Cell) and grows the multiplier, or cashes out (Move.CashOut).
// Hitting a mine busts the round for nothing.
type Mines struct {
	rng      Rand
	mines    int // number of mines on the board
	stake    domain.Minutes
	isMine   [minesTiles]bool
	revealed [minesTiles]bool
	picks    int // safe tiles revealed so far
	busted   bool
	done     bool
	result   Outcome
	laidOut  bool // mines are placed lazily on the first reveal
}

// NewMines builds a mines game with the given mine count (clamped to a sane
// range), drawing from rng.
func NewMines(rng Rand, mines int) *Mines {
	if mines < 1 {
		mines = 1
	}
	if mines > minesTiles-1 {
		mines = minesTiles - 1
	}
	return &Mines{rng: rng, mines: mines}
}

// Kind implements Game.
func (m *Mines) Kind() domain.GameKind { return domain.GameMines }

// Start implements Game.
func (m *Mines) Start(stake domain.Minutes) GameState {
	m.stake = stake
	m.picks = 0
	m.busted = false
	m.done = false
	m.laidOut = false
	m.result = Outcome{}
	m.isMine = [minesTiles]bool{}
	m.revealed = [minesTiles]bool{}
	return GameState{Stake: stake, Multiplier: 1, Symbols: m.render()}
}

// Step reveals Move.Cell or cashes out. It returns the new state and whether the
// round is over. Revealing a mine busts; cashing out settles at the current
// multiplier.
func (m *Mines) Step(move Move) (GameState, bool) {
	if m.done {
		return m.state(), true
	}
	if move.CashOut {
		return m.cashOut(), true
	}

	cell := move.Cell
	if cell < 0 || cell >= minesTiles || m.revealed[cell] {
		// Ignore an out-of-range or repeated pick; the round continues.
		return m.state(), false
	}

	// Lay mines on the first reveal, guaranteeing the first pick is safe so a
	// round never ends on move one.
	if !m.laidOut {
		m.layMines(cell)
		m.laidOut = true
	}

	m.revealed[cell] = true
	if m.isMine[cell] {
		m.busted = true
		m.done = true
		m.result = settle(m.stake, 0, minesCeiling, m.render())
		return m.state(), true
	}

	m.picks++
	if m.picks >= minesTiles-m.mines {
		// Board cleared: forced cash-out at the top multiplier.
		return m.cashOut(), true
	}
	return m.state(), false
}

// Result implements Game.
func (m *Mines) Result() Outcome { return m.result }

// Multiplier returns the current cash-out multiplier given picks made so far.
func (m *Mines) Multiplier() float64 { return minesMultiplier(m.mines, m.picks) }

// cashOut settles the round at the current multiplier.
func (m *Mines) cashOut() GameState {
	m.done = true
	mult := minesMultiplier(m.mines, m.picks)
	if m.picks == 0 {
		mult = 0 // nothing revealed, nothing won
	}
	m.result = settle(m.stake, mult, minesCeiling, m.render())
	return m.state()
}

// layMines places m.mines mines uniformly at random, skipping the safe cell.
// Rejection sampling can stall on a pathological RNG that keeps returning the
// same cell, so after a bounded number of attempts it deterministically fills
// the next free cell. A real RNG effectively never reaches the fallback.
func (m *Mines) layMines(safe int) {
	placed := 0
	maxAttempts := minesTiles * 20
	for placed < m.mines && maxAttempts > 0 {
		maxAttempts--
		c := m.rng.Intn(minesTiles)
		if c == safe || m.isMine[c] {
			continue
		}
		m.isMine[c] = true
		placed++
	}
	// Fallback: fill remaining mines into the first free, non-safe cells.
	for i := 0; placed < m.mines && i < minesTiles; i++ {
		if i == safe || m.isMine[i] {
			continue
		}
		m.isMine[i] = true
		placed++
	}
}

// minesMultiplier is the fair cash-out multiplier after revealing `picks` safe
// tiles, scaled by the target RTP. The fair multiplier is the inverse of the
// probability of having survived this far:
//
//	fair = ∏_{i=0}^{picks-1} (tiles - i) / (safe - i)
//
// where safe = tiles - mines. Multiplying by minesRTP applies the house edge.
func minesMultiplier(mines, picks int) float64 {
	if picks <= 0 {
		return 1
	}
	safe := minesTiles - mines
	fair := 1.0
	for i := range picks {
		fair *= float64(minesTiles-i) / float64(safe-i)
	}
	return fair * minesRTP
}

// state builds the current GameState snapshot.
func (m *Mines) state() GameState {
	mult := minesMultiplier(m.mines, m.picks)
	if m.busted {
		mult = 0
	}
	return GameState{
		Stake:      m.stake,
		Multiplier: mult,
		Symbols:    m.render(),
		Done:       m.done,
	}
}

// render returns a per-tile symbol for the UI: "·" hidden, "✓" safe-revealed,
// "✗" the mine that was hit (only shown once busted).
func (m *Mines) render() []string {
	out := make([]string, minesTiles)
	for i := range minesTiles {
		switch {
		case m.revealed[i] && m.isMine[i]:
			out[i] = "💥"
		case m.revealed[i]:
			out[i] = "💎"
		default:
			out[i] = "🟦"
		}
	}
	return out
}

// MinesBoardString renders a board snapshot as a 5×5 text grid.
func MinesBoardString(symbols []string) string {
	var b strings.Builder
	for i, sym := range symbols {
		b.WriteString(sym)
		if (i+1)%5 == 0 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	return b.String()
}
