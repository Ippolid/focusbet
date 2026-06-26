package game

import (
	"testing"

	"github.com/Ippolid/focusbet/internal/economy"
)

// fixedIntn is a Rand whose Intn always returns a chosen value, to force a known
// roulette pocket. (Not used for mines, which needs distinct draws.)
type fixedIntn struct{ v int }

func (f fixedIntn) Intn(int) int     { return f.v }
func (f fixedIntn) Float64() float64 { return 0 }

// seqIntn is a Rand whose Intn returns a fixed sequence, cycling. It lets mines
// tests place several distinct mines deterministically.
type seqIntn struct {
	seq []int
	i   int
}

func (s *seqIntn) Intn(n int) int {
	v := s.seq[s.i%len(s.seq)]
	s.i++
	return v % n
}
func (s *seqIntn) Float64() float64 { return 0 }

func TestRoulette_StraightWin(t *testing.T) {
	// Force pocket 7; bet the number 7 -> 36× payout.
	r := NewRoulette(fixedIntn{v: 7})
	r.Start(10)
	_, done := r.Step(Move{Bet: BetNumber, Number: 7})
	if !done {
		t.Fatal("spin should be done")
	}
	out := r.Result()
	if out.Multiplier != straightMult {
		t.Errorf("mult = %v, want %v", out.Multiplier, straightMult)
	}
	if out.Payout != 360 || !out.Win {
		t.Errorf("payout = %v win = %v, want 360/true", out.Payout, out.Win)
	}
	if out.Fraction != 1.0 { // straight win hits the ceiling
		t.Errorf("fraction = %v, want 1.0", out.Fraction)
	}
}

func TestRoulette_RedBlackEvenOdd(t *testing.T) {
	tests := []struct {
		name   string
		pocket int
		bet    RouletteBet
		win    bool
	}{
		{"red on red", 1, BetRed, true},
		{"red on black", 2, BetRed, false},
		{"black on black", 2, BetBlack, true},
		{"even on 4", 4, BetEven, true},
		{"even on zero loses", 0, BetEven, false},
		{"odd on 5", 5, BetOdd, true},
		{"red on zero loses", 0, BetRed, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRoulette(fixedIntn{v: tc.pocket})
			r.Start(10)
			r.Step(Move{Bet: tc.bet})
			out := r.Result()
			if out.Win != tc.win {
				t.Errorf("win = %v, want %v (pocket %d)", out.Win, tc.win, tc.pocket)
			}
		})
	}
}

// TestRoulette_RTP enumerates the wheel for each bet type and asserts the RTP is
// the expected single-zero value (~0.973), the same for every bet.
func TestRoulette_RTP(t *testing.T) {
	bets := []struct {
		bet    RouletteBet
		number int
	}{
		{BetRed, 0}, {BetBlack, 0}, {BetEven, 0}, {BetOdd, 0}, {BetNumber, 17},
	}
	for _, b := range bets {
		outcomes := make([]economy.Outcome, roulettePockets)
		for pocket := range roulettePockets {
			mult := 0.0
			if rouletteWins(Move{Bet: b.bet, Number: b.number}, pocket) {
				if b.bet == BetNumber {
					mult = straightMult
				} else {
					mult = evenMoneyMult
				}
			}
			outcomes[pocket] = economy.Outcome{Prob: 1.0 / roulettePockets, Multiplier: mult}
		}
		rtp, err := economy.RTP(outcomes)
		if err != nil {
			t.Fatalf("RTP(%v): %v", b.bet, err)
		}
		if !economy.WithinRTPCorridor(rtp, 0.97, 0.974) {
			t.Errorf("%v RTP = %.4f, want ~0.973", b.bet, rtp)
		}
	}
}

func TestMines_RevealThenCashOut(t *testing.T) {
	// 3 mines at cells 22,23,24; we reveal cell 0 (safe — the first reveal is
	// always guaranteed safe anyway).
	m := NewMines(&seqIntn{seq: []int{22, 23, 24}}, 3)
	m.Start(10)

	st, done := m.Step(Move{Cell: 0})
	if done {
		t.Fatal("first reveal should be safe and not end the round")
	}
	if st.Multiplier <= 1 {
		t.Errorf("multiplier after 1 pick = %v, want > 1", st.Multiplier)
	}

	// Cash out: payout = stake × multiplier, a win.
	cs, done := m.Step(Move{CashOut: true})
	if !done {
		t.Fatal("cash out should end the round")
	}
	out := m.Result()
	if !out.Win || out.Payout <= 10 {
		t.Errorf("cash-out outcome = %+v, want a win above stake", out)
	}
	if !cs.Done {
		t.Error("state should be Done after cash out")
	}
}

func TestMines_HitMineBusts(t *testing.T) {
	// Single mine at cell 5; first reveal cell 0 (safe), then reveal cell 5.
	m := NewMines(fixedIntn{v: 5}, 1)
	m.Start(10)
	if _, done := m.Step(Move{Cell: 0}); done {
		t.Fatal("first reveal must be safe")
	}
	_, done := m.Step(Move{Cell: 5})
	if !done {
		t.Fatal("hitting the mine should end the round")
	}
	out := m.Result()
	if out.Win || out.Payout != 0 {
		t.Errorf("busted outcome = %+v, want total loss", out)
	}
}

func TestMines_CashOutAtZeroPicksWinsNothing(t *testing.T) {
	m := NewMines(&seqIntn{seq: []int{22, 23, 24}}, 3)
	m.Start(10)
	_, done := m.Step(Move{CashOut: true})
	if !done {
		t.Fatal("cash out should end the round")
	}
	if out := m.Result(); out.Win || out.Payout != 0 {
		t.Errorf("cash out with no picks = %+v, want nothing", out)
	}
}

// TestMines_RTP checks that the per-pick fair-times-RTP multiplier yields the
// configured RTP at each cash-out point: EV = survival_prob × multiplier = RTP.
func TestMines_RTP(t *testing.T) {
	const mines = 3
	safe := minesTiles - mines
	for picks := 1; picks <= safe; picks++ {
		// Probability of surviving `picks` reveals on a fresh board.
		surv := 1.0
		for i := 0; i < picks; i++ {
			surv *= float64(safe-i) / float64(minesTiles-i)
		}
		ev := surv * minesMultiplier(mines, picks)
		if ev < minesRTP-1e-9 || ev > minesRTP+1e-9 {
			t.Errorf("picks=%d EV=%.4f, want RTP %.2f", picks, ev, minesRTP)
		}
	}
}

func TestMinesBoardString(t *testing.T) {
	syms := make([]string, minesTiles)
	for i := range syms {
		syms[i] = "·"
	}
	got := MinesBoardString(syms)
	// 5 rows, each ending in newline.
	if n := countByte(got, '\n'); n != 5 {
		t.Errorf("rows = %d, want 5\n%s", n, got)
	}
}

func countByte(s string, b byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			n++
		}
	}
	return n
}
