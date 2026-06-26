package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestMinesCashOutBlockedAtZeroPicks verifies the cash-out key is ignored until
// at least one gem is revealed, so a player can't settle a total loss while the
// button would otherwise promise the stake back.
func TestMinesCashOutBlockedAtZeroPicks(t *testing.T) {
	m := newTestModel(t)
	ms := &minesScreen{root: m, phase: minesPlaying} // empty state -> 0 gems found

	// Pressing "c" with no gems revealed must not end the round.
	updated, _ := ms.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	got := updated.(*minesScreen)
	if got.phase != minesPlaying {
		t.Errorf("phase = %v after cash-out at 0 picks, want still playing", got.phase)
	}

	// The button must read as disabled, not a winnable cash-out.
	if v := got.View(); !strings.Contains(v, "reveal a gem first") {
		t.Errorf("view should show disabled cash-out hint:\n%s", v)
	}
}
