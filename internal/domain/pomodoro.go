package domain

// Pomodoro controls focus/break lengths and the cycle
type Pomodoro struct {
	// Preset selects a built-in timing. "custom" uses the explicit fields below
	Preset string `json:"preset"` // "classic" | "deep" | "desktime" | "short" | "custom"

	// These apply when Preset == "custom" (or override a preset if set)
	FocusMinutes      int `json:"focus_minutes"`
	ShortBreakMinutes int `json:"short_break_minutes"`
	LongBreakMinutes  int `json:"long_break_minutes"`

	// CycleLength is how many focus sessions before a long break (classic = 4)
	CycleLength int `json:"cycle_length"`
}
