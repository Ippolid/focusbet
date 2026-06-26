package domain

import "testing"

func TestMinutesHuman(t *testing.T) {
	tests := []struct {
		name string
		in   Minutes
		want string
	}{
		{"exact zero", 0, "0m"},
		{"whole minutes", 25, "25m"},
		{"hours and minutes", 565, "9h 25m"},
		{"exact hours", 120, "2h"},
		{"fractional minute", 2.5, "2m 30s"},
		{"sub-minute", 0.5, "30s"},
		{"negative", -90, "-1h 30m"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Human(); got != tc.want {
				t.Errorf("Minutes(%v).Human() = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMinutesString(t *testing.T) {
	tests := []struct {
		in   Minutes
		want string
	}{
		{0, "00:00"},
		{25, "25:00"},
		{2.5, "02:30"},
		{-1.5, "-01:30"},
	}
	for _, tc := range tests {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("Minutes(%v).String() = %q, want %q", tc.in, got, tc.want)
		}
	}
}
