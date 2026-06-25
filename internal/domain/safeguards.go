package domain

// Safeguards are the anti-addiction limits, on by default by design.
type Safeguards struct {
	// DailyPlayCap: max number of plays/spins per day, regardless of bank.
	DailyPlayCap int `json:"daily_play_cap"`
	// CooldownSeconds: forced pause between plays, breaks mindless spamming.
	CooldownSeconds int `json:"cooldown_seconds"`
	// LossMeansWait: if true, after a loss you must wait instead of replaying.
	LossMeansWait bool `json:"loss_means_wait"`
	// MaxDailyLossMinutes: stop play once this much rest has been lost today (0 = off).
	MaxDailyLossMinutes int `json:"max_daily_loss_minutes"`
}
