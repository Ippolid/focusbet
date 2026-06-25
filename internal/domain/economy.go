package domain

// Economy holds the rest-bank math. All ratios are fractions of focus time.
// Invariant: 0 < base_ratio < fair_ratio < max_ratio <= max_ratio_ceiling.
type Economy struct {
	// BaseRatio: guaranteed rest, can never be gambled away.
	BaseRatio float64 `json:"base_ratio"`
	// FairRatio: rest you get if you DON'T play — the standard pomodoro break.
	FairRatio float64 `json:"fair_ratio"`
	// MaxRatio: ceiling on rest if you play and win big
	MaxRatio float64 `json:"max_ratio"`

	// DailySpendMultiplier: daily spend cap = daily earnings * this. >1 gives a
	// real "profit" from playing/banking rather than just shuffling your break
	DailySpendMultiplier float64 `json:"daily_spend_multiplier"`

	// BankCapMinutes: hard ceiling on accumulated bank, so you can't hoard hours
	BankCapMinutes int `json:"bank_cap_minutes"`

	// InterruptPenalty: fraction of earnings kept when a session is stopped early
	// 0 = lose everything on early stop, 1 = keep all. Discourages fake focus
	InterruptKeepFraction float64 `json:"interrupt_keep_fraction"`
}
