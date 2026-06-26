package economy

import (
	"errors"
	"fmt"
	"math"
)

// ErrNotDistribution is returned when outcome probabilities don't sum to ~1.
var ErrNotDistribution = errors.New("economy: outcome probabilities must sum to 1")

// distTolerance is how far the probabilities may sum from 1 before RTP rejects
// them, absorbing ordinary float rounding.
const distTolerance = 1e-9

// Outcome is one possible result of a single play: the chance it happens and the
// payout multiplier applied to the stake (0 = total loss, 1 = stake returned,
// 2 = double, …).
type Outcome struct {
	Prob       float64
	Multiplier float64
}

// RTP (return-to-player) is the expected payout per unit staked, i.e.
// Σ prob × multiplier over all outcomes. An RTP below 1 means the bank trends
// down over time, pulling the player back to work — the design intent.
//
// It returns ErrNotDistribution if the probabilities don't sum to 1, which would
// make the result meaningless.
func RTP(outcomes []Outcome) (float64, error) {
	var sumP, ev float64
	for _, o := range outcomes {
		sumP += o.Prob
		ev += o.Prob * o.Multiplier
	}
	if math.Abs(sumP-1) > distTolerance {
		return 0, fmt.Errorf("%w: sum=%g", ErrNotDistribution, sumP)
	}
	return ev, nil
}

// HouseEdge is 1 − RTP: the fraction of each stake the bank keeps on average.
func HouseEdge(outcomes []Outcome) (float64, error) {
	rtp, err := RTP(outcomes)
	if err != nil {
		return 0, err
	}
	return 1 - rtp, nil
}

// ExpectedFraction is the average game_fraction a play yields, given the payout
// distribution and the multiplier→fraction mapping the game uses. It lets a test
// confirm the average outcome lands near config.Games.TargetFraction (~0.44),
// i.e. the average player gets roughly the fair break.
//
// toFraction maps a payout multiplier to a game_fraction in [0,1].
func ExpectedFraction(outcomes []Outcome, toFraction func(multiplier float64) float64) (float64, error) {
	var sumP, ef float64
	for _, o := range outcomes {
		sumP += o.Prob
		ef += o.Prob * toFraction(o.Multiplier)
	}
	if math.Abs(sumP-1) > distTolerance {
		return 0, fmt.Errorf("%w: sum=%g", ErrNotDistribution, sumP)
	}
	return ef, nil
}

// WithinRTPCorridor reports whether rtp falls within [lo, hi] inclusive. Games
// use it in tests to assert they sit in the intended corridor (≈0.88–0.92, with
// roulette allowed to be more generous).
func WithinRTPCorridor(rtp, lo, hi float64) bool {
	return rtp >= lo && rtp <= hi
}
