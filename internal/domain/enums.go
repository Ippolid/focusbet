package domain

import "fmt"

// Mode is the high-level state the app is in. The TUI root model owns the
// transitions; other packages only read it.
type Mode int

const (
	// ModeIdle is the dashboard, nothing running.
	ModeIdle Mode = iota
	// ModeFocus is an active focus session.
	ModeFocus
	// ModeBreak is an active break (short or long).
	ModeBreak
	// ModeResult is the post-focus fork: bank / rest / play.
	ModeResult
	// ModePlaying is an active game.
	ModePlaying
)

// String implements fmt.Stringer.
func (m Mode) String() string {
	switch m {
	case ModeIdle:
		return "idle"
	case ModeFocus:
		return "focus"
	case ModeBreak:
		return "break"
	case ModeResult:
		return "result"
	case ModePlaying:
		return "playing"
	default:
		return fmt.Sprintf("Mode(%d)", int(m))
	}
}

// GameKind identifies a game implementation.
type GameKind int

const (
	// GameSlots is the slot machine.
	GameSlots GameKind = iota
	// GameRoulette is roulette.
	GameRoulette
	// GameMines is the minesweeper-style game.
	GameMines
)

// String implements fmt.Stringer.
func (g GameKind) String() string {
	switch g {
	case GameSlots:
		return "slots"
	case GameRoulette:
		return "roulette"
	case GameMines:
		return "mines"
	default:
		return fmt.Sprintf("GameKind(%d)", int(g))
	}
}

// PresetKind names a built-in pomodoro timing. It mirrors the string stored in
// Pomodoro.Preset, giving callers typed constants instead of bare strings.
type PresetKind string

const (
	// PresetClassic is the 25/5/20 ×4 timing.
	PresetClassic PresetKind = "classic"
	// PresetDeep is a long deep-work timing.
	PresetDeep PresetKind = "deep"
	// PresetDeskTime is the 52/17 timing.
	PresetDeskTime PresetKind = "desktime"
	// PresetShort is a short timing.
	PresetShort PresetKind = "short"
	// PresetCustom uses the explicit minute fields on Pomodoro.
	PresetCustom PresetKind = "custom"
)
