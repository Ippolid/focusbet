package game

import "github.com/Ippolid/focusbet/internal/domain"

// Move is a player action within a game. Slots ignore it; roulette reads Bet
// (and Number for a straight bet); mines uses Cell and CashOut. It is a small
// tagged value so the single Game interface fits every game without per-game
// method sets.
type Move struct {
	// Cell is the tile index a multi-step game should reveal (mines).
	Cell int
	// CashOut, when true, ends a multi-step game and banks the current multiplier.
	CashOut bool
	// Bet selects the wager type for roulette (red/black/even/odd/number).
	Bet RouletteBet
	// Number is the chosen pocket for a straight roulette bet (0..36).
	Number int
}

// GameState is the public snapshot of a game in progress, for the UI to render.
type GameState struct {
	// Stake is the wager in minutes.
	Stake domain.Minutes
	// Multiplier is the current payout multiplier (0 while undecided / on a bust).
	Multiplier float64
	// Symbols renders the visible outcome (reels, pocket, revealed tiles).
	Symbols []string
	// Done reports whether the game has finished.
	Done bool
}

// Outcome is the settled result of a game, in the terms the rest of the app
// settles on. It is deliberately bank-agnostic: the caller converts Fraction
// into rest minutes via economy and moves currency via the balance core.
type Outcome struct {
	// Stake is the wager that was placed.
	Stake domain.Minutes
	// Multiplier is the final payout multiplier (0 = total loss).
	Multiplier float64
	// Payout is the winnings credited back, in minutes (Stake × Multiplier).
	Payout domain.Minutes
	// Fraction is the game_fraction in [0,1] economy uses to place the result in
	// the [base, max] rest corridor.
	Fraction float64
	// Win reports whether the player finished ahead of the stake.
	Win bool
}

// Game is a playable game of chance. The same interface serves one-shot games
// (Start then a single Step) and multi-step games (Start then repeated Step
// until it reports done).
type Game interface {
	// Kind identifies the game.
	Kind() domain.GameKind
	// Start begins a round with the given stake and returns the initial state.
	Start(stake domain.Minutes) GameState
	// Step applies a move and returns the new state and whether the round is over.
	// One-shot games resolve on the first Step regardless of the move.
	Step(move Move) (GameState, bool)
	// Result returns the settled outcome. It is valid only once the round is done.
	Result() Outcome
}

// maxFraction is the multiplier that maps to game_fraction 1.0 (full win). A game
// converts its multiplier to a fraction by scaling against this ceiling, so the
// busiest win lands the player at the rest Max and a total loss lands at Base.
const maxFraction = 1.0

// fractionForMultiplier maps a payout multiplier to a game_fraction in [0,1].
// 0× → 0 (keep base), and the multiplier that returns the whole stakeable zone
// maps to 1. ceiling is the multiplier treated as a full win.
func fractionForMultiplier(multiplier, ceiling float64) float64 {
	if ceiling <= 0 {
		return 0
	}
	f := multiplier / ceiling
	if f < 0 {
		return 0
	}
	if f > maxFraction {
		return maxFraction
	}
	return f
}
