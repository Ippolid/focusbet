package tui

import (
	"strings"
	"testing"

	"github.com/Ippolid/focusbet/internal/app"
)

// TestRestSlider_NoNegativeRepeat guards the slider against a negative
// strings.Repeat count when the chosen amount exceeds the bank (which panicked
// the program before the clamp was added).
func TestRestSlider_NoNegativeRepeat(t *testing.T) {
	core, err := app.New(t.TempDir(), 1)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	m := New(core)
	r := &restScreen{root: m, amount: 100} // amount far above an empty bank

	// Must not panic and must produce a fixed-width bar.
	out := r.slider(5) // bank smaller than amount
	if !strings.Contains(out, "[") || !strings.Contains(out, "]") {
		t.Errorf("slider output malformed: %q", out)
	}

	// Also exercise the full View, which is where the panic surfaced.
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("View panicked: %v", p)
		}
	}()
	m.width, m.height = 70, 18
	_ = r.View()
}
