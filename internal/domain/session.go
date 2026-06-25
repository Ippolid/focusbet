package domain

// Session is one focus run. Stored by start time + length so it survives restarts.
type Session struct {
	StartedAt     int64  `json:"started_at"`     // unix seconds
	FocusSeconds  int64  `json:"focus_seconds"`  // planned length
	Preset        string `json:"preset"`         // "classic", "deep", ...
	Task          string `json:"task"`           // what the user is working on
	CyclePosition int    `json:"cycle_position"` // 1..4 within the pomodoro cycle
}
