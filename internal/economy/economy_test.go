package economy

import (
	"errors"
	"math"
	"testing"

	"github.com/Ippolid/focusbet/internal/domain"
)

func approx(a, b domain.Minutes) bool {
	return math.Abs(float64(a-b)) < 1e-9
}

func TestEarnForSession(t *testing.T) {
	tests := []struct {
		name                  string
		focused, planned, brk domain.Minutes
		want                  domain.Minutes
	}{
		{"full session banks whole break", 25, 25, 5, 5},
		{"half session banks half", 25, 50, 17, 8.5},
		{"overrun clamps to full", 60, 50, 10, 10},
		{"zero break earns nothing", 25, 25, 0, 0},
		{"planned zero treated as full", 25, 0, 5, 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := EarnForSession(tc.focused, tc.planned, tc.brk); !approx(got, tc.want) {
				t.Errorf("EarnForSession(%v,%v,%v) = %v, want %v",
					tc.focused, tc.planned, tc.brk, got, tc.want)
			}
		})
	}
}

func TestRTP(t *testing.T) {
	// A fair coin paying 1.9× on a win: RTP = 0.5*1.9 = 0.95.
	out := []Outcome{
		{Prob: 0.5, Multiplier: 1.9},
		{Prob: 0.5, Multiplier: 0},
	}
	rtp, err := RTP(out)
	if err != nil {
		t.Fatalf("RTP: %v", err)
	}
	if math.Abs(rtp-0.95) > 1e-9 {
		t.Errorf("rtp = %g, want 0.95", rtp)
	}
	if edge, _ := HouseEdge(out); math.Abs(edge-0.05) > 1e-9 {
		t.Errorf("edge = %g, want 0.05", edge)
	}
	if !WithinRTPCorridor(rtp, 0.90, 0.97) {
		t.Errorf("rtp %g not in corridor", rtp)
	}
}

func TestRTP_NotDistribution(t *testing.T) {
	out := []Outcome{{Prob: 0.3, Multiplier: 2}, {Prob: 0.3, Multiplier: 0}}
	if _, err := RTP(out); !errors.Is(err, ErrNotDistribution) {
		t.Errorf("err = %v, want ErrNotDistribution", err)
	}
}

func TestExpectedFraction(t *testing.T) {
	// Two outcomes; map multiplier to fraction as min(mult/2,1).
	out := []Outcome{
		{Prob: 0.5, Multiplier: 2}, // fraction 1
		{Prob: 0.5, Multiplier: 0}, // fraction 0
	}
	ef, err := ExpectedFraction(out, func(m float64) float64 {
		f := m / 2
		if f > 1 {
			f = 1
		}
		return f
	})
	if err != nil {
		t.Fatalf("ExpectedFraction: %v", err)
	}
	if math.Abs(ef-0.5) > 1e-9 {
		t.Errorf("ef = %g, want 0.5", ef)
	}
}
