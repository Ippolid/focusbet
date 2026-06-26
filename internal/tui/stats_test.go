package tui

import (
	"testing"
)

func TestWinRate(t *testing.T) {
	tests := []struct {
		wins, played int64
		want         string
	}{
		{0, 0, "—"},
		{10, 68, "15%"},
		{1, 2, "50%"},
		{68, 68, "100%"},
	}
	for _, tc := range tests {
		if got := winRate(tc.wins, tc.played); got != tc.want {
			t.Errorf("winRate(%d,%d) = %q, want %q", tc.wins, tc.played, got, tc.want)
		}
	}
}

func TestBestMultiplier(t *testing.T) {
	if got := bestMultiplier(0); got != "—" {
		t.Errorf("bestMultiplier(0) = %q, want —", got)
	}
	if got := bestMultiplier(16); got != "×16.00" {
		t.Errorf("bestMultiplier(16) = %q, want ×16.00", got)
	}
}
