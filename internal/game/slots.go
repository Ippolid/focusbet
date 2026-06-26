package game

import "github.com/Ippolid/focusbet/internal/domain"

// slotReels is the number of reels spun per round.
const slotReels = 3

// symbol is one reel face with its weight and three-of-a-kind payout.
type symbol struct {
	face       string
	weight     int     // relative frequency on a reel
	tripleMult float64 // payout multiplier for three of this symbol
}

// slotStrip defines the reel: rarer symbols pay more. Weights and payouts are
// tuned so the overall RTP sits near 0.91 (verified analytically in the test).
var slotStrip = []symbol{
	{"🍒", 8, 5},
	{"🍋", 6, 9},
	{"🔔", 4, 16},
	{"⭐", 3, 32},
	{"💎", 2, 64},
	{"7️⃣", 1, 160},
}

// pairMult is paid when exactly two reels match (any symbol). Below 1.0 so a
// pair softens a loss rather than turning a profit — keeps the RTP in corridor.
const pairMult = 0.8

// slotCeiling is the multiplier mapped to game_fraction 1.0 — the jackpot.
const slotCeiling = 160

var slotTotalWeight = func() int {
	w := 0
	for _, s := range slotStrip {
		w += s.weight
	}
	return w
}()

// Slots is a three-reel weighted slot machine. It is one-shot: Start then a
// single Step resolves the spin.
type Slots struct {
	rng    Rand
	stake  domain.Minutes
	result Outcome
	done   bool
}

// NewSlots builds a slot machine drawing from rng.
func NewSlots(rng Rand) *Slots { return &Slots{rng: rng} }

// Kind implements Game.
func (s *Slots) Kind() domain.GameKind { return domain.GameSlots }

// Start implements Game.
func (s *Slots) Start(stake domain.Minutes) GameState {
	s.stake = stake
	s.done = false
	s.result = Outcome{}
	return GameState{Stake: stake}
}

// Step implements Game: it spins all reels and settles the outcome. The move is
// ignored — slots resolve in a single step.
func (s *Slots) Step(_ Move) (GameState, bool) {
	reels := make([]int, slotReels)
	faces := make([]string, slotReels)
	counts := make(map[int]int, slotReels)
	for i := range reels {
		reels[i] = s.spinReel()
		faces[i] = slotStrip[reels[i]].face
		counts[reels[i]]++
	}

	bestSym, bestCount := -1, 0
	for sym, c := range counts {
		if c > bestCount {
			bestSym, bestCount = sym, c
		}
	}

	mult := 0.0
	switch bestCount {
	case slotReels:
		mult = slotStrip[bestSym].tripleMult
	case 2:
		mult = pairMult
	}

	s.result = settle(s.stake, mult, slotCeiling, faces)
	s.done = true
	return GameState{Stake: s.stake, Multiplier: mult, Symbols: faces, Done: true}, true
}

// Result implements Game.
func (s *Slots) Result() Outcome { return s.result }

// ReelDrawForSymbol returns the Rand.Intn value that lands the reel on symbol
// index idx (0 = lowest payout … len-1 = jackpot). It lets callers build a
// deterministic Rand that forces a chosen outcome, for tests and headless runs.
func ReelDrawForSymbol(idx int) int {
	start := 0
	for i := 0; i < idx && i < len(slotStrip); i++ {
		start += slotStrip[i].weight
	}
	return start
}

// SymbolCount is the number of distinct reel symbols.
func SymbolCount() int { return len(slotStrip) }

// SlotSymbols returns the reel faces in strip order, for UI animation.
func SlotSymbols() []string {
	faces := make([]string, len(slotStrip))
	for i, s := range slotStrip {
		faces[i] = s.face
	}
	return faces
}

// spinReel picks a symbol index by weight.
func (s *Slots) spinReel() int {
	r := s.rng.Intn(slotTotalWeight)
	for i, sym := range slotStrip {
		if r < sym.weight {
			return i
		}
		r -= sym.weight
	}
	return len(slotStrip) - 1 // unreachable; guards against rounding
}

// settle builds an Outcome from a stake and multiplier.
func settle(stake domain.Minutes, multiplier, ceiling float64, symbols []string) Outcome {
	payout := domain.Minutes(float64(stake) * multiplier)
	return Outcome{
		Stake:      stake,
		Multiplier: multiplier,
		Payout:     payout,
		Fraction:   fractionForMultiplier(multiplier, ceiling),
		Win:        payout > stake,
	}
}
