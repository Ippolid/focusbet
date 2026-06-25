package repository

import "github.com/Ippolid/focusbet/internal/domain"

// StateSchemaVersion is bumped when the State shape changes, so old saves can be migrated.
const StateSchemaVersion = 1

// ConfigSchemaVersion is bumped when the Config shape changes, independently of State.
const ConfigSchemaVersion = 1

// State is the single source of truth that gets persisted to state.json.
// It is loaded whole at startup and written whole on every meaningful change.
type State struct {
	SchemaVersion int             `json:"schema_version"`
	Balance       domain.Balance  `json:"balance"`
	ActiveSession *domain.Session `json:"active_session"` // nil when no session is running
	Stats         domain.Stats    `json:"stats"`
	UpdatedAt     int64           `json:"updated_at"` // unix seconds
}

// LogEntry is one line in history.jsonl (append-only event log).
type LogEntry struct {
	TS     int64  `json:"ts"`             // unix seconds
	Op     string `json:"op"`             // "earn","spend","win","lose","rest"
	Delta  int64  `json:"delta"`          // change in minutes (+/-)
	Game   string `json:"game,omitempty"` // "slots","roulette","mines" or empty
	Reason string `json:"reason,omitempty"`
}

type Config struct {
	SchemaVersion int               `json:"schema_version"`
	Pomodoro      domain.Pomodoro   `json:"pomodoro"`
	Economy       domain.Economy    `json:"economy"`
	Games         domain.Games      `json:"games"`
	Safeguards    domain.Safeguards `json:"safeguards"`
	Behavior      domain.Behavior   `json:"behavior"`
	DayResetHour  int               `json:"day_reset_hour"` // 0..23, local hour when the day rolls over
}

// DefaultState returns a fresh State for a first run (no file yet).
func DefaultState(now int64) *State {
	return &State{
		SchemaVersion: StateSchemaVersion,
		Balance:       domain.Balance{},
		ActiveSession: nil,
		Stats:         domain.Stats{},
		UpdatedAt:     now,
	}
}

func DefaultConfig() *Config {
	return &Config{
		SchemaVersion: ConfigSchemaVersion,
		Pomodoro: domain.Pomodoro{
			Preset:            "classic",
			FocusMinutes:      25,
			ShortBreakMinutes: 5,
			LongBreakMinutes:  20,
			CycleLength:       4,
		},
		Economy: domain.Economy{
			BaseRatio:             0.12,
			FairRatio:             0.20,
			MaxRatio:              0.40,
			DailySpendMultiplier:  1.80,
			BankCapMinutes:        120,
			InterruptKeepFraction: 0.0,
		},
		Games: domain.Games{
			RTP:              0.90,
			TargetFraction:   0.44,
			BaseStakeMinutes: 30,
			MinesCount:       3,
			RouletteType:     "european",
			ProvablyFair:     false,
		},
		Safeguards: domain.Safeguards{
			DailyPlayCap:        20,
			CooldownSeconds:     1,
			LossMeansWait:       false,
			MaxDailyLossMinutes: 0,
		},
		Behavior: domain.Behavior{
			AfterFocus:       "ask",
			ConfirmEarlyStop: true,
		},
		DayResetHour: 0,
	}
}
