package balance

import (
	"errors"
	"testing"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/repository"
)

// fakeStore is an in-memory Store for tests: it records saves and log entries
// without touching disk.
type fakeStore struct {
	saves   int
	logs    []repository.LogEntry
	saveErr error
}

func (f *fakeStore) SaveState(*repository.State) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saves++
	return nil
}

func (f *fakeStore) AppendLog(e repository.LogEntry) error {
	f.logs = append(f.logs, e)
	return nil
}

// testEconomy: bank cap 120.
var testEconomy = domain.Economy{
	BankCapMinutes: 120,
}

func newBank(t *testing.T, starting domain.Minutes) (*Bank, *fakeStore) {
	t.Helper()
	st := repository.DefaultState(1000)
	st.Balance.Minutes = starting
	fs := &fakeStore{}
	return New(fs, st, testEconomy), fs
}

func TestEarn(t *testing.T) {
	b, fs := newBank(t, 0)

	got, err := b.Earn(1000, 30, "session")
	if err != nil {
		t.Fatalf("Earn: %v", err)
	}
	if got != 30 || b.Bank() != 30 {
		t.Errorf("credited=%v bank=%v, want 30/30", got, b.Bank())
	}
	if len(fs.logs) != 1 || fs.logs[0].Op != "earn" || fs.logs[0].TS != 1000 {
		t.Errorf("log = %+v, want one earn at ts 1000", fs.logs)
	}
}

func TestEarn_BankCapClamps(t *testing.T) {
	b, _ := newBank(t, 110) // cap 120, room 10

	got, err := b.Earn(1000, 30, "session")
	if err != nil {
		t.Fatalf("Earn: %v", err)
	}
	if got != 10 || b.Bank() != 120 {
		t.Errorf("credited=%v bank=%v, want 10/120", got, b.Bank())
	}

	// Already at cap: a full bank credits zero and is NOT an error, so a
	// completed session still settles cleanly.
	credited, err := b.Earn(1000, 5, "session")
	if err != nil {
		t.Errorf("Earn at cap = %v, want nil error", err)
	}
	if credited != 0 || b.Bank() != 120 {
		t.Errorf("credited=%v bank=%v, want 0/120 at cap", credited, b.Bank())
	}
}

func TestEarn_InvalidAmount(t *testing.T) {
	b, _ := newBank(t, 0)
	if _, err := b.Earn(1000, 0, "x"); !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}
}

func TestSpend(t *testing.T) {
	b, fs := newBank(t, 100)

	if err := b.Spend(1000, 30, "slots"); err != nil {
		t.Fatalf("Spend: %v", err)
	}
	if b.Bank() != 70 {
		t.Errorf("bank=%v, want 70", b.Bank())
	}
	if fs.logs[0].Op != "spend" || fs.logs[0].Delta != -30*60 {
		t.Errorf("log = %+v, want spend -1800s", fs.logs[0])
	}
}

func TestSpend_InsufficientFunds(t *testing.T) {
	b, _ := newBank(t, 10)
	if err := b.Spend(1000, 30, "slots"); !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("err = %v, want ErrInsufficientFunds", err)
	}
	if b.Bank() != 10 {
		t.Errorf("bank changed on failed spend: %v", b.Bank())
	}
}

func TestWin_NotCapped(t *testing.T) {
	// Game winnings are not subject to the bank cap (cap 120 here).
	b, _ := newBank(t, 100)
	if err := b.Win(1000, 50, "slots"); err != nil { // 100+50 exceeds cap, but wins aren't capped
		t.Fatalf("Win: %v", err)
	}
	if b.Bank() != 150 {
		t.Errorf("bank = %v, want 150 (wins uncapped)", b.Bank())
	}
}

func TestSpendOnRest(t *testing.T) {
	b, fs := newBank(t, 50)
	if err := b.SpendOnRest(1000, 20); err != nil {
		t.Fatalf("SpendOnRest: %v", err)
	}
	if b.Bank() != 30 {
		t.Errorf("bank = %v, want 30", b.Bank())
	}
	if fs.logs[0].Op != "rest" {
		t.Errorf("op = %q, want rest", fs.logs[0].Op)
	}
}

func TestCanSpend(t *testing.T) {
	b, _ := newBank(t, 50)
	if ok, _ := b.CanSpend(20); !ok {
		t.Error("CanSpend(20) = false, want true")
	}
	if ok, reason := b.CanSpend(60); ok || reason == "" {
		t.Errorf("CanSpend(60) = %v %q, want false with reason", ok, reason)
	}
	if ok, _ := b.CanSpend(0); ok {
		t.Error("CanSpend(0) = true, want false")
	}
}

func TestRecordLoss(t *testing.T) {
	b, fs := newBank(t, 100)
	if err := b.RecordLoss(1000, 5, "slots"); err != nil {
		t.Fatalf("RecordLoss: %v", err)
	}
	if fs.logs[0].Op != "lose" {
		t.Errorf("op = %q, want lose", fs.logs[0].Op)
	}
	// RecordLoss only logs; it does not change the bank (the stake was already spent).
	if b.Bank() != 100 {
		t.Errorf("bank changed on RecordLoss: %v", b.Bank())
	}
}

func TestNeverNegative(t *testing.T) {
	b, _ := newBank(t, 5)
	// Spend more than the bank in one go is rejected; bank stays put.
	if err := b.Spend(1000, 10, "slots"); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("err = %v, want ErrInsufficientFunds", err)
	}
	if b.Bank() < 0 {
		t.Errorf("bank went negative: %v", b.Bank())
	}
}
