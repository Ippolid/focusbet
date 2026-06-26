package game

import (
	"fmt"

	"github.com/Ippolid/focusbet/internal/domain"
)

// RouletteBet is the kind of wager placed on a roulette spin.
type RouletteBet int

const (
	// BetRed pays even money when the pocket is red.
	BetRed RouletteBet = iota
	// BetBlack pays even money when the pocket is black.
	BetBlack
	// BetEven pays even money when the pocket is a non-zero even number.
	BetEven
	// BetOdd pays even money when the pocket is odd.
	BetOdd
	// BetNumber pays 36× when the pocket equals Move.Number (a straight bet).
	BetNumber
)

// String implements fmt.Stringer for display.
func (b RouletteBet) String() string {
	switch b {
	case BetRed:
		return "red"
	case BetBlack:
		return "black"
	case BetEven:
		return "even"
	case BetOdd:
		return "odd"
	case BetNumber:
		return "number"
	default:
		return "?"
	}
}

// roulettePockets is the European single-zero wheel: 0..36.
const roulettePockets = 37

// evenMoneyMult is the payout multiplier for red/black/even/odd (1:1 + stake).
const evenMoneyMult = 2.0

// straightMult is the payout multiplier for a winning single number (35:1 + stake).
const straightMult = 36.0

// rouletteCeiling maps a straight-number win to game_fraction 1.0; smaller wins
// land proportionally lower in the rest corridor.
const rouletteCeiling = straightMult

// redPockets is the set of red numbers on a European wheel.
var redPockets = map[int]bool{
	1: true, 3: true, 5: true, 7: true, 9: true, 12: true, 14: true, 16: true,
	18: true, 19: true, 21: true, 23: true, 25: true, 27: true, 30: true,
	32: true, 34: true, 36: true,
}

// Roulette is a European single-zero roulette game. It is one-shot: Start, then a
// single Step (carrying the bet) resolves the spin.
type Roulette struct {
	rng    Rand
	stake  domain.Minutes
	result Outcome
	done   bool
}

// NewRoulette builds a roulette game drawing from rng.
func NewRoulette(rng Rand) *Roulette { return &Roulette{rng: rng} }

// Kind implements Game.
func (r *Roulette) Kind() domain.GameKind { return domain.GameRoulette }

// Start implements Game.
func (r *Roulette) Start(stake domain.Minutes) GameState {
	r.stake = stake
	r.done = false
	r.result = Outcome{}
	return GameState{Stake: stake}
}

// Step spins the wheel and settles the bet carried by move.
func (r *Roulette) Step(move Move) (GameState, bool) {
	pocket := r.rng.Intn(roulettePockets)

	mult := 0.0
	if rouletteWins(move, pocket) {
		if move.Bet == BetNumber {
			mult = straightMult
		} else {
			mult = evenMoneyMult
		}
	}

	symbols := []string{fmt.Sprintf("%d", pocket), pocketColor(pocket)}
	r.result = settle(r.stake, mult, rouletteCeiling, symbols)
	r.done = true
	return GameState{Stake: r.stake, Multiplier: mult, Symbols: symbols, Done: true}, true
}

// Result implements Game.
func (r *Roulette) Result() Outcome { return r.result }

// rouletteWins reports whether the bet wins for the given pocket. Zero loses all
// even-money bets, which is the house edge.
func rouletteWins(move Move, pocket int) bool {
	switch move.Bet {
	case BetRed:
		return redPockets[pocket]
	case BetBlack:
		return pocket != 0 && !redPockets[pocket]
	case BetEven:
		return pocket != 0 && pocket%2 == 0
	case BetOdd:
		return pocket%2 == 1
	case BetNumber:
		return pocket == move.Number
	default:
		return false
	}
}

// pocketColor returns the display colour name for a pocket.
func pocketColor(pocket int) string {
	switch {
	case pocket == 0:
		return "green"
	case redPockets[pocket]:
		return "red"
	default:
		return "black"
	}
}
