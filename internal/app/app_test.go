package app

import (
	"testing"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/game"
	"github.com/Ippolid/focusbet/internal/repository"
)

// fixedRand drives slots to a deterministic outcome by returning reel-start
// weights for chosen symbol indices.
type fixedRand struct {
	seq []int
	i   int
}

func (r *fixedRand) Intn(n int) int {
	v := r.seq[r.i%len(r.seq)]
	r.i++
	return v % n
}
func (r *fixedRand) Float64() float64 { return 0 }

// TestHeadlessFullLoop drives the whole cycle from code with no TUI: load →
// focus → earn → play slots → settle → persist → reload, asserting the bank and
// money log survive a round trip. This is the critical wiring milestone.
func TestHeadlessFullLoop(t *testing.T) {
	dir := t.TempDir()
	const start = 1_700_000_000

	a, err := New(dir, start)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Default config is classic 25min, base 0.12, fair 0.20 -> earn 25*(0.20-0.12)=2.
	res, err := a.RunBankedSession(start, start+25*60)
	if err != nil {
		t.Fatalf("RunBankedSession: %v", err)
	}
	if res.FocusedMinutes != 25 {
		t.Errorf("focused = %v, want 25", res.FocusedMinutes)
	}
	// classic preset banks the whole 5-minute fair break.
	if res.Banked != 5 {
		t.Errorf("banked = %v, want 5", res.Banked)
	}
	if a.Bank().Bank() != 5 {
		t.Errorf("bank = %v, want 5", a.Bank().Bank())
	}

	// Earn more so there is comfortably enough banked to stake a few rounds.
	for i := 0; i < 20; i++ {
		if _, err := a.RunBankedSession(start, start+25*60); err != nil {
			t.Fatalf("extra session %d: %v", i, err)
		}
	}
	bankBeforePlay := a.Bank().Bank()
	if bankBeforePlay < 30 {
		t.Fatalf("bank %v too low to stake 30", bankBeforePlay)
	}

	// Force a jackpot (three of the highest symbol) so the play is a clear win.
	jackpot := game.ReelDrawForSymbol(game.SymbolCount() - 1)
	rng := &fixedRand{seq: []int{jackpot, jackpot, jackpot}}
	if err := a.Stake(start, domain.GameSlots, 30); err != nil {
		t.Fatalf("Stake: %v", err)
	}
	g := a.NewGame(domain.GameSlots, rng)
	g.Start(30)
	g.Step(game.Move{})
	out := g.Result()
	if !out.Win {
		t.Fatalf("expected jackpot win, got %+v", out)
	}
	if err := a.Settle(start, domain.GameSlots, out); err != nil {
		t.Fatalf("Settle: %v", err)
	}

	// Bank after play = before - stake + payout.
	wantBank := bankBeforePlay - 30 + out.Payout
	// Bank cap may clamp; allow either exact or the cap.
	cap := a.Config().Economy.BankCapMinutes
	if a.Bank().Bank() != wantBank && int(a.Bank().Bank()) != cap {
		t.Errorf("bank after play = %v, want %v (or cap %d)", a.Bank().Bank(), wantBank, cap)
	}

	// Reload from disk: state must have persisted.
	reloaded, err := New(dir, start)
	if err != nil {
		t.Fatalf("reload New: %v", err)
	}
	if reloaded.Bank().Bank() != a.Bank().Bank() {
		t.Errorf("reloaded bank = %v, want %v", reloaded.Bank().Bank(), a.Bank().Bank())
	}

	// The money log must contain earn + spend + win lines.
	store, err := repository.New(dir)
	if err != nil {
		t.Fatalf("open store for read: %v", err)
	}
	ops := map[string]int{}
	if err := store.ReadLog(func(e repository.LogEntry) error {
		ops[e.Op]++
		return nil
	}); err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if ops["earn"] == 0 || ops["spend"] == 0 || ops["win"] == 0 {
		t.Errorf("log ops = %v, want earn+spend+win present", ops)
	}
}

func TestPlayInsufficientFunds(t *testing.T) {
	dir := t.TempDir()
	const start = 1_700_000_000
	a, err := New(dir, start)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Fresh bank is empty; staking must fail with insufficient funds.
	if err := a.Stake(start, domain.GameSlots, 5); err == nil {
		t.Error("staking on empty bank should fail")
	}
}

// TestCustomTimings verifies that each custom setter switches to the custom
// preset, edits its own field, and carries the other three over so editing one
// timing never resets the rest.
func TestCustomTimings(t *testing.T) {
	a, err := New(t.TempDir(), 1_700_000_000)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Start from a built-in preset; first edit seeds custom from it.
	if err := a.SetPreset("classic"); err != nil {
		t.Fatalf("SetPreset: %v", err)
	}

	if err := a.SetCustomCycleLength(6); err != nil {
		t.Fatalf("SetCustomCycleLength: %v", err)
	}
	if err := a.SetCustomLongBreak(30); err != nil {
		t.Fatalf("SetCustomLongBreak: %v", err)
	}
	if err := a.SetCustomFocus(40); err != nil {
		t.Fatalf("SetCustomFocus: %v", err)
	}

	// All edits must coexist: focus from the last edit, cycle/long break from
	// earlier ones (not reset back to classic's 4 / 20).
	focus, brk := a.Timings()
	if focus != 40 {
		t.Errorf("focus = %d, want 40", focus)
	}
	if brk != 5 { // never edited -> carried from classic seed
		t.Errorf("short break = %d, want 5", brk)
	}
	if got := a.CycleLength(); got != 6 {
		t.Errorf("cycle length = %d, want 6", got)
	}
	if got := a.LongBreakMinutes(); got != 30 {
		t.Errorf("long break = %d, want 30", got)
	}

	// Clamping: zero/negative falls back to 1.
	if err := a.SetCustomCycleLength(0); err != nil {
		t.Fatalf("SetCustomCycleLength(0): %v", err)
	}
	if got := a.CycleLength(); got != 1 {
		t.Errorf("cycle length = %d, want clamped 1", got)
	}
}

// TestCompleteFocusAtCap verifies a session that completes with a full bank
// banks zero (not an error) and still records the session stats.
func TestCompleteFocusAtCap(t *testing.T) {
	dir := t.TempDir()
	const start = 1_700_000_000
	a, err := New(dir, start)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Fill the bank to its cap (default 120 min) via banked sessions.
	for a.Bank().Bank() < 120 {
		if _, err := a.RunBankedSession(start, start+25*60); err != nil {
			t.Fatalf("RunBankedSession: %v", err)
		}
	}
	before := a.Stats().TimerCount

	// A completing session at the cap must not error, must bank 0, and must still
	// bump the session counter.
	_, banked, err := a.CompleteFocus(start, 25, false, false)
	if err != nil {
		t.Fatalf("CompleteFocus at cap = %v, want nil", err)
	}
	if banked != 0 {
		t.Errorf("banked at cap = %v, want 0", banked)
	}
	if a.Stats().TimerCount != before+1 {
		t.Errorf("TimerCount = %d, want %d", a.Stats().TimerCount, before+1)
	}

	// And it survives a reload (was lost before the fix).
	reloaded, err := New(dir, start)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Stats().TimerCount != before+1 {
		t.Errorf("reloaded TimerCount = %d, want %d", reloaded.Stats().TimerCount, before+1)
	}
}

// TestCompleteFocusEndsCycle verifies the cycle-ending session banks the long
// break, not the short one.
func TestCompleteFocusEndsCycle(t *testing.T) {
	a, err := New(t.TempDir(), 1_700_000_000)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// classic: short break 5, long break 20.
	reward, _, err := a.CompleteFocus(1_700_000_000, 25, false, true)
	if err != nil {
		t.Fatalf("CompleteFocus: %v", err)
	}
	if reward.Fair != 20 { // full focus -> whole long break
		t.Errorf("cycle-ending reward = %v, want 20 (long break)", reward.Fair)
	}
}

// TestSettleStats verifies the BestMultiplier win-gate and that won/loss minutes
// are recorded in seconds without truncation.
func TestSettleStats(t *testing.T) {
	a, err := New(t.TempDir(), 1_700_000_000)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	const now = 1_700_000_000

	// A losing partial return (payout < stake, Win=false) must NOT set the best
	// multiplier, and records the loss.
	loss := game.Outcome{Stake: 1, Multiplier: 0.8, Payout: 0.8, Win: false}
	if err := a.Settle(now, domain.GameSlots, loss); err != nil {
		t.Fatalf("Settle loss: %v", err)
	}
	if a.Stats().BestMultiplier != 0 {
		t.Errorf("BestMultiplier = %v after a loss, want 0", a.Stats().BestMultiplier)
	}

	// A real win sets the best multiplier and records won seconds exactly.
	win := game.Outcome{Stake: 1, Multiplier: 2, Payout: 2, Win: true}
	if err := a.Settle(now, domain.GameMines, win); err != nil {
		t.Fatalf("Settle win: %v", err)
	}
	if a.Stats().BestMultiplier != 2 {
		t.Errorf("BestMultiplier = %v, want 2", a.Stats().BestMultiplier)
	}
	if got := a.Stats().WonSeconds; got != 60 { // net 1 min won = 60s
		t.Errorf("WonSeconds = %d, want 60", got)
	}
}
