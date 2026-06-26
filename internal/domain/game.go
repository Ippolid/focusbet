package domain

// Games tunes the gambling layer shared by games
type Games struct {
	// RTP: target return-to-player (<1 so the bank trends down, pulling you back
	// to work). 0.90 = player loses ~10% per stake on average.
	RTP float64 `json:"rtp"`

	// BaseStakeMinutes: default wager size per play, in minutes of bank.
	BaseStakeMinutes int `json:"base_stake_minutes"`

	// Per-game knobs.
	MinesCount   int    `json:"mines_count"`   // mines on a 5x5 board (3 = default)
	RouletteType string `json:"roulette_type"` // "european" (single 0) | "american" (0 and 00)

	// ProvablyFair: enable verifiable RNG (commitment/seed/nonce). Off = plain RNG.
	ProvablyFair bool `json:"provably_fair"`
}
