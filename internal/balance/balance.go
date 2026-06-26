// Package balance is the currency core: the single place the rest bank is ever
// changed. Pomodoro and games never write to the store directly — they go
// through here, so every mutation is checked against the invariants and recorded
// in the money log exactly once.
package balance

import (
	"errors"
	"fmt"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/repository"
)

// Store is the persistence port the balance core depends on. It is declared here,
// on the consumer side, so balance owns the contract and repository.Store
// satisfies it without knowing about balance.
type Store interface {
	SaveState(*repository.State) error
	AppendLog(repository.LogEntry) error
}

// Sentinel errors for invariant violations. Callers (and the UI) match on these.
var (
	// ErrInvalidAmount is returned for a non-positive amount where positive is required.
	ErrInvalidAmount = errors.New("balance: amount must be positive")
	// ErrInsufficientFunds is returned when a debit exceeds the bank.
	ErrInsufficientFunds = errors.New("balance: insufficient funds")
)

// Bank is the in-memory guardian of the rest bank, backed by a Store. It is not
// safe for concurrent use; the TUI drives it from one goroutine.
type Bank struct {
	store   Store
	state   *repository.State
	economy domain.Economy
}

// New builds a Bank over an already-loaded state and the economy config. The
// state pointer is owned by the Bank from here on; read it back via the getters.
func New(store Store, state *repository.State, e domain.Economy) *Bank {
	return &Bank{store: store, state: state, economy: e}
}

// Bank returns the current bank balance in minutes.
func (b *Bank) Bank() domain.Minutes { return b.state.Balance.Minutes }

// Earn credits rest earned by focusing — the only source of new currency. It is
// capped by BankCapMinutes; the surplus that would overflow the cap is dropped
// (you can't hoard hours). A full bank is not an error: it credits zero and
// returns (0, nil) so a completed session still settles cleanly. Returns the
// amount actually banked.
func (b *Bank) Earn(now int64, amount domain.Minutes, reason string) (domain.Minutes, error) {
	if amount <= 0 {
		return 0, ErrInvalidAmount
	}
	credited := amount
	if cap := domain.Minutes(b.economy.BankCapMinutes); cap > 0 {
		room := cap - b.state.Balance.Minutes
		if room <= 0 {
			return 0, nil // at cap: drop the surplus silently, not an error
		}
		if credited > room {
			credited = room
		}
	}
	b.state.Balance.Minutes += credited
	return credited, b.commit(now, repository.LogEntry{
		Op:     "earn",
		Delta:  credited.Seconds(),
		Reason: reason,
	})
}

// Spend debits a game stake. It enforces sufficient funds.
func (b *Bank) Spend(now int64, amount domain.Minutes, game string) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount > b.state.Balance.Minutes {
		return ErrInsufficientFunds
	}
	b.state.Balance.Minutes -= amount
	return b.commit(now, repository.LogEntry{
		Op:    "spend",
		Delta: -amount.Seconds(),
		Game:  game,
	})
}

// Win credits game winnings, respecting the bank ceiling.
// Win credits a game payout. Unlike focus earnings it is NOT subject to the bank
// cap — winnings you fought for shouldn't evaporate just because the bank is full,
// which would make playing near the cap pointless.
func (b *Bank) Win(now int64, amount domain.Minutes, game string) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	b.state.Balance.Minutes += amount
	return b.commit(now, repository.LogEntry{
		Op:    "win",
		Delta: amount.Seconds(),
		Game:  game,
	})
}

// RecordLoss logs rest minutes lost in a game. The stake was already removed by
// Spend; this only records the loss event in the money log.
func (b *Bank) RecordLoss(now int64, amount domain.Minutes, game string) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	return b.commit(now, repository.LogEntry{
		Op:    "lose",
		Delta: -amount.Seconds(),
		Game:  game,
	})
}

// SpendOnRest debits the bank to top up a break beyond the free fair break.
func (b *Bank) SpendOnRest(now int64, amount domain.Minutes) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount > b.state.Balance.Minutes {
		return ErrInsufficientFunds
	}
	b.state.Balance.Minutes -= amount
	b.state.Stats.RestSeconds += amount.Seconds()
	return b.commit(now, repository.LogEntry{
		Op:    "rest",
		Delta: -amount.Seconds(),
	})
}

// CanSpend reports whether amount could be staked right now, and if not, why.
// It mutates nothing, so the UI can call it to enable/disable controls.
func (b *Bank) CanSpend(amount domain.Minutes) (bool, string) {
	switch {
	case amount <= 0:
		return false, "amount must be positive"
	case amount > b.state.Balance.Minutes:
		return false, "not enough banked rest"
	default:
		return true, ""
	}
}

// commit stamps the timestamp, persists the state, then appends one log line.
// State is saved first so a crash never leaves a logged event that didn't
// actually change the bank.
func (b *Bank) commit(now int64, entry repository.LogEntry) error {
	entry.TS = now
	b.state.UpdatedAt = now
	if err := b.store.SaveState(b.state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	if err := b.store.AppendLog(entry); err != nil {
		return fmt.Errorf("append log: %w", err)
	}
	return nil
}
