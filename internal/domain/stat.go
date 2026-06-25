package domain

// Stats is lifetime/aggregate counters shown on the dashboard.
type Stats struct {
	StreakDays     int     `json:"streak_days"`
	LastActiveDate string  `json:"last_active_date"` // "2006-01-02"
	WorkingMinutes int64   `json:"working_minutes"`
	RestMinutes    int64   `json:"rest_minutes"`
	TimerCount     int64   `json:"timer_count"` // completed sessions
	GameCount      int64   `json:"game_count"`
	MinutesWon     int64   `json:"minutes_won"`
	Wins           int64   `json:"wins"`
	Loses          int64   `json:"loses"`
	BestMultiplier float64 `json:"best_multiplier"`
}
