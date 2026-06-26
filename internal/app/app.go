// Package app is the headless composition layer: it wires the store, balance
// core, pomodoro engine and games together and drives the full focus→earn→play
// loop without any TUI. main and the tests both build on it, so the wiring is
// exercised end to end before a single screen exists.
package app

import (
	"fmt"

	"github.com/Ippolid/focusbet/internal/balance"
	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/game"
	"github.com/Ippolid/focusbet/internal/pomodoro"
	"github.com/Ippolid/focusbet/internal/repository"
)

// App holds the wired-together core. It owns the loaded state and the bank; the
// pomodoro engine is created per run from config.
type App struct {
	store  *repository.Store
	config *repository.Config
	state  *repository.State
	bank   *balance.Bank
}

// New loads (or initialises) state and config from dir and wires the bank.
func New(dir string, now int64) (*App, error) {
	store, err := repository.New(dir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	cfg, err := store.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	state, err := store.LoadState(now)
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}
	return &App{
		store:  store,
		config: cfg,
		state:  state,
		bank:   balance.New(store, state, cfg.Economy),
	}, nil
}

// Bank exposes the currency core (the only path that mutates the balance).
func (a *App) Bank() *balance.Bank { return a.bank }

// Config exposes the loaded config.
func (a *App) Config() *repository.Config { return a.config }

// Stats exposes the lifetime stats for display.
func (a *App) Stats() domain.Stats { return a.state.Stats }

// NewEngine builds a fresh pomodoro engine from the loaded config.
func (a *App) NewEngine() *pomodoro.Engine {
	return pomodoro.NewEngine(a.config.Pomodoro, a.config.Economy, a.config.Games)
}

// IsFirstRun reports whether the user has never completed a session, used to
// decide whether to show onboarding.
func (a *App) IsFirstRun() bool { return a.state.Stats.TimerCount == 0 }

// Save persists the current state. Bank operations already save on every change;
// this is for stat-only changes the TUI makes (e.g. a completed session's
// counters) and for graceful shutdown.
func (a *App) Save(now int64) error {
	a.state.UpdatedAt = now
	if err := a.store.SaveState(a.state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	return nil
}

// SetPreset switches the pomodoro preset and persists the config.
func (a *App) SetPreset(preset string) error {
	a.config.Pomodoro.Preset = preset
	if err := a.store.SaveConfig(a.config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// FlowMode reports whether flow mode is on: a finished focus session goes
// straight into the rest break, with no fork to confirm. Off shows the fork.
func (a *App) FlowMode() bool {
	return a.config.Behavior.AfterFocus == "auto_rest"
}

// SetFlowMode turns flow mode on (auto-rest) or off (ask) and persists.
func (a *App) SetFlowMode(on bool) error {
	if on {
		a.config.Behavior.AfterFocus = "auto_rest"
	} else {
		a.config.Behavior.AfterFocus = "ask"
	}
	if err := a.store.SaveConfig(a.config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// Timings returns the resolved focus / short-break minutes for the active
// preset, for display in settings.
func (a *App) Timings() (focus, shortBreak int) {
	d := pomodoro.Resolve(a.config.Pomodoro)
	return d.Focus, d.ShortBreak
}

// LongBreakMinutes returns the resolved long-break length for the active preset.
func (a *App) LongBreakMinutes() int {
	return pomodoro.Resolve(a.config.Pomodoro).LongBreak
}

// CycleLength returns how many focus sessions run before a long break.
func (a *App) CycleLength() int {
	return pomodoro.Resolve(a.config.Pomodoro).CycleLen
}

// SetCustomFocus sets a custom focus length in minutes, switches to the "custom"
// preset so it takes effect, and persists. Values below 1 are clamped to 1.
func (a *App) SetCustomFocus(minutes int) error {
	d := a.resolved()
	d.Focus = clampMin(minutes)
	return a.setCustom(d)
}

// SetCustomBreak sets a custom short-break length in minutes, switching to the
// "custom" preset, and persists. Values below 1 are clamped to 1.
func (a *App) SetCustomBreak(minutes int) error {
	d := a.resolved()
	d.ShortBreak = clampMin(minutes)
	return a.setCustom(d)
}

// SetCustomLongBreak sets a custom long-break length in minutes (the rest taken
// after a full cycle), switching to the "custom" preset. Clamped to >= 1.
func (a *App) SetCustomLongBreak(minutes int) error {
	d := a.resolved()
	d.LongBreak = clampMin(minutes)
	return a.setCustom(d)
}

// SetCustomCycleLength sets how many focus sessions run before a long break,
// switching to the "custom" preset. Clamped to >= 1.
func (a *App) SetCustomCycleLength(sessions int) error {
	d := a.resolved()
	d.CycleLen = clampMin(sessions)
	return a.setCustom(d)
}

// resolved returns the currently-active timings, so editing one custom field
// seeds the others from the active preset instead of zeroing them.
func (a *App) resolved() pomodoro.Durations { return pomodoro.Resolve(a.config.Pomodoro) }

// setCustom writes all four custom timings and persists, switching to the
// "custom" preset. Carrying every field through keeps long break and cycle
// length when only focus/break is edited (and vice versa).
func (a *App) setCustom(d pomodoro.Durations) error {
	a.config.Pomodoro.Preset = "custom"
	a.config.Pomodoro.FocusMinutes = d.Focus
	a.config.Pomodoro.ShortBreakMinutes = d.ShortBreak
	a.config.Pomodoro.LongBreakMinutes = d.LongBreak
	a.config.Pomodoro.CycleLength = d.CycleLen
	if err := a.store.SaveConfig(a.config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

func clampMin(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

// CompleteFocus records a finished focus session of focusedMinutes at now: it
// updates streak and work stats, banks the kept earnings, and persists. It
// returns the reward (for the post-focus fork) and the amount banked.
//
// endsCycle is true when this session completes a full cycle and the long break
// follows; the reward is then anchored on the long break (you earn the rest you
// are about to get) instead of the short one.
func (a *App) CompleteFocus(now int64, focusedMinutes domain.Minutes, interrupted, endsCycle bool) (pomodoro.Reward, domain.Minutes, error) {
	eng := a.NewEngine()
	focus, brk := a.Timings()
	if endsCycle {
		brk = a.LongBreakMinutes()
	}
	reward := pomodoro.ComputeReward(focusedMinutes, domain.Minutes(focus), domain.Minutes(brk), a.config.Economy)

	pomodoro.UpdateStreak(&a.state.Stats, now)
	a.state.Stats.TimerCount++
	a.state.Stats.WorkSeconds += focusedMinutes.Seconds()

	earn := eng.KeptEarnings(reward, interrupted)
	var banked domain.Minutes
	if earn > 0 {
		credited, err := a.bank.Earn(now, earn, "session")
		if err != nil {
			return reward, 0, fmt.Errorf("bank earnings: %w", err)
		}
		banked = credited
	}
	// Persist the session stats regardless of whether anything was banked: a full
	// bank credits zero (not an error), and the streak/counter increments must
	// still survive a reload. bank.Earn already saved when it credited; saving
	// again is cheap and keeps the no-earn path correct.
	if err := a.Save(now); err != nil {
		return reward, 0, err
	}
	return reward, banked, nil
}

// SessionResult summarises one completed focus session and what happened after.
type SessionResult struct {
	FocusedMinutes domain.Minutes
	Banked         domain.Minutes // credited to the bank (when banking the break)
	Played         bool
	GameOutcome    game.Outcome // valid when Played is true
}

// RunBankedSession runs a focus session that completes at completeAt and banks
// the fair-minus-base emission. It returns what was credited.
func (a *App) RunBankedSession(startAt, completeAt int64) (SessionResult, error) {
	eng := a.NewEngine()
	eng.Start(startAt)
	res := eng.Tick(completeAt)
	if !res.Completed || !res.WasFocus {
		return SessionResult{}, fmt.Errorf("session did not complete as focus: %+v", res)
	}

	pomodoro.UpdateStreak(&a.state.Stats, completeAt)
	a.state.Stats.TimerCount++
	a.state.Stats.WorkSeconds += res.FocusedMinutes.Seconds()

	earn := eng.KeptEarnings(res.Reward, false)
	banked, err := a.bank.Earn(completeAt, earn, "session")
	if err != nil {
		return SessionResult{}, fmt.Errorf("bank earnings: %w", err)
	}
	return SessionResult{FocusedMinutes: res.FocusedMinutes, Banked: banked}, nil
}

// NewGame builds a fresh game of the given kind drawing from rng (pass
// game.NewCryptoRand() in production). Mines uses the configured mine count.
func (a *App) NewGame(kind domain.GameKind, rng game.Rand) game.Game {
	switch kind {
	case domain.GameRoulette:
		return game.NewRoulette(rng)
	case domain.GameMines:
		return game.NewMines(rng, a.config.Games.MinesCount)
	default:
		return game.NewSlots(rng)
	}
}

// Stake debits the wager for a game before play begins. The bank enforces
// sufficient-funds and bank-cap invariants. Settle must be called once the round
// resolves to credit any winnings and record stats.
func (a *App) Stake(now int64, kind domain.GameKind, stake domain.Minutes) error {
	if err := a.bank.Spend(now, stake, kind.String()); err != nil {
		return fmt.Errorf("place stake: %w", err)
	}
	return nil
}

// Settle credits a resolved game outcome back to the bank and updates stats. The
// stake was already debited by Stake, so a win credits the full payout and a
// loss only records the lost minutes.
func (a *App) Settle(now int64, kind domain.GameKind, out game.Outcome) error {
	a.state.Stats.GameCount++
	// "Best multiplier" is a win stat: only count a round that actually won, so a
	// losing slots pair (×0.8) never shows up as the player's best result.
	if out.Win && out.Multiplier > a.state.Stats.BestMultiplier {
		a.state.Stats.BestMultiplier = out.Multiplier
	}

	// The stake was already debited by Stake. Always credit the full payout back,
	// so a partial return (e.g. a slots pair, or a mines cash-out below stake) is
	// not silently lost. Net for the round is payout − stake.
	if out.Payout > 0 {
		if err := a.bank.Win(now, out.Payout, kind.String()); err != nil {
			return fmt.Errorf("credit payout: %w", err)
		}
	}

	if out.Win {
		a.state.Stats.Wins++
		a.state.Stats.WonSeconds += (out.Payout - out.Stake).Seconds()
	} else {
		a.state.Stats.Loses++
		if err := a.bank.RecordLoss(now, out.Stake-out.Payout, kind.String()); err != nil {
			return fmt.Errorf("record loss: %w", err)
		}
	}
	return nil
}

// State exposes the current persisted state (read-only intent).
func (a *App) State() *repository.State { return a.state }
