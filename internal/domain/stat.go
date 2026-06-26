package domain

// Stats is lifetime/aggregate counters shown on the dashboard. The time totals
// are stored in whole seconds (not minutes) so fractional-minute focus sessions
// and sub-minute game stakes accumulate without truncation.
type Stats struct {
	StreakDays     int     `json:"streak_days"`
	LastActiveDate string  `json:"last_active_date"` // "2006-01-02"
	WorkSeconds    int64   `json:"work_seconds"`     // total focused time
	RestSeconds    int64   `json:"rest_seconds"`     // total minutes spent on rest
	TimerCount     int64   `json:"timer_count"`      // completed sessions
	GameCount      int64   `json:"game_count"`
	WonSeconds     int64   `json:"won_seconds"` // net minutes won across games
	Wins           int64   `json:"wins"`
	Loses          int64   `json:"loses"`
	BestMultiplier float64 `json:"best_multiplier"`
}
