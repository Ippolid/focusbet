package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestEaseToward(t *testing.T) {
	// Snaps to target when already within epsilon.
	if got, more := easeToward(0.5, 0.5); got != 0.5 || more {
		t.Errorf("easeToward(0.5,0.5) = %v,%v want 0.5,false", got, more)
	}
	// Moves a fraction toward the target and reports more work remaining.
	got, more := easeToward(0, 1)
	if !more || got <= 0 || got >= 1 {
		t.Errorf("easeToward(0,1) = %v,%v want strictly between 0 and 1, true", got, more)
	}
	// Repeated application converges to the target.
	x := 0.0
	for i := 0; i < 200; i++ {
		x, _ = easeToward(x, 1)
	}
	if x < 0.999 {
		t.Errorf("did not converge: %v", x)
	}
}

func TestSmoothBar(t *testing.T) {
	plain := lipgloss.NewStyle()
	tests := []struct {
		name  string
		frac  float64
		width int
	}{
		{"empty", 0, 10},
		{"half", 0.5, 10},
		{"full", 1, 10},
		{"over", 1.5, 10},   // clamps, must not panic
		{"under", -0.5, 10}, // clamps, must not panic
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := smoothBar(tc.frac, tc.width, plain, plain)
			if out == "" {
				t.Errorf("smoothBar returned empty for %v", tc.frac)
			}
		})
	}
}

func TestConfettiOverlayStable(t *testing.T) {
	c := newConfetti(40)
	content := strings.Repeat("hello world\n", 6)
	// Stepping and overlaying must never panic and must preserve the box height.
	for i := 0; i < confettiFrames+2; i++ {
		out := c.overlay(content, 40, 8)
		if lines := strings.Count(out, "\n"); lines != 7 { // 8 rows -> 7 separators
			t.Fatalf("frame %d: overlay produced %d separators, want 7", i, lines)
		}
		c.step()
	}
	if c.active() {
		t.Error("confetti still active after all frames")
	}
}
