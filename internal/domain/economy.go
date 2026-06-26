package domain

// Economy tunes the single-currency rest bank: the bank holds rest minutes,
// focusing earns them, resting and gambling spend them.
type Economy struct {
	// BankCapMinutes: hard ceiling on accumulated bank, so you can't hoard hours.
	BankCapMinutes int `json:"bank_cap_minutes"`

	// InterruptKeepFraction: fraction of a break kept when a session is stopped
	// early. 0 = lose everything on early stop, 1 = keep all. Discourages fake focus.
	InterruptKeepFraction float64 `json:"interrupt_keep_fraction"`
}
