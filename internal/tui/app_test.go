package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ippolid/focusbet/internal/app"
)

// newTestModel builds a root model over a temp-dir app core, with a fixed clock
// and a known terminal size so centering renders.
func newTestModel(t *testing.T) *model {
	t.Helper()
	core, err := app.New(t.TempDir(), 1_700_000_000)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	m := New(core)
	m.now = func() int64 { return 1_700_000_000 }
	m.width, m.height = 80, 24
	return m
}

// press sends a key press through the model and returns the updated model.
func press(t *testing.T, m *model, k string) *model {
	t.Helper()
	var msg tea.Msg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	updated, _ := m.Update(msg)
	return updated.(*model)
}

func TestFirstRunShowsOnboarding(t *testing.T) {
	m := newTestModel(t)
	if m.current != screenOnboarding {
		t.Fatalf("first run screen = %v, want onboarding", m.current)
	}
	if !strings.Contains(m.View(), "Pick a rhythm to start") {
		t.Errorf("onboarding view missing intro text:\n%s", m.View())
	}
}

func TestOnboardingToDashboard(t *testing.T) {
	m := newTestModel(t)
	// Press enter to confirm the preset -> emits switch to dashboard.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("expected a switch command")
	}
	// Execute the command to get the switchMsg, then feed it back.
	m2, _ := m.Update(cmd())
	m = m2.(*model)
	if m.current != screenDashboard {
		t.Fatalf("after onboarding screen = %v, want dashboard", m.current)
	}
	// The brand renders as ASCII-art; assert on the stable footer hint instead.
	if !strings.Contains(m.View(), "enter select") {
		t.Errorf("dashboard view missing menu hint:\n%s", m.View())
	}
}

func TestDashboardNavigateToStats(t *testing.T) {
	m := newTestModel(t)
	// Jump straight to the dashboard.
	m2, _ := m.Update(switchMsg{to: screenDashboard})
	m = m2.(*model)

	// Menu order: Start, Games, Stats, Settings. Down twice -> Stats.
	m = press(t, m, "down")
	m = press(t, m, "down")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("expected switch command from menu")
	}
	m3, _ := m.Update(cmd())
	m = m3.(*model)
	if m.current != screenStats {
		t.Fatalf("screen = %v, want stats", m.current)
	}
	if !strings.Contains(m.View(), "STATISTICS") {
		t.Errorf("stats view missing header:\n%s", m.View())
	}
}

func TestTimerScreenRenders(t *testing.T) {
	m := newTestModel(t)
	m2, _ := m.Update(switchMsg{to: screenTimer})
	m = m2.(*model)
	if m.current != screenTimer {
		t.Fatalf("screen = %v, want timer", m.current)
	}
	// The time is ASCII-art now; assert on the stable status footer instead.
	v := m.View()
	if !strings.Contains(v, "pause") || !strings.Contains(v, "stop") {
		t.Errorf("timer view looks wrong:\n%s", v)
	}
}

func TestAllScreensBuildAndRender(t *testing.T) {
	m := newTestModel(t)
	for _, sc := range []screen{
		screenOnboarding, screenDashboard, screenTaskInput, screenTimer, screenResult,
		screenRest, screenRestTimer, screenCountdown, screenGames, screenPlay,
		screenRoulette, screenMines, screenStats, screenSettings,
	} {
		m2, _ := m.Update(switchMsg{to: sc})
		m = m2.(*model)
		if got := m.View(); strings.TrimSpace(got) == "" {
			t.Errorf("screen %v rendered empty", sc)
		}
	}
}
