package domain

import "fmt"

// Minutes is the currency unit of focusbet: rest time, earned by focusing and
// wagered in games. It is fractional so partial minutes survive ratio math.
type Minutes float64

func (m *Minutes) Add(amount Minutes) {
	*m += amount
}

func (m *Minutes) Dec(amount Minutes) {
	*m -= amount
}

// Seconds returns the value as whole seconds, for the persistence layer which
// keeps timestamps and deltas in unix seconds (int64).
func (m Minutes) Seconds() int64 {
	return int64(float64(m) * 60)
}

// MinutesFromSeconds converts a seconds count to Minutes.
func MinutesFromSeconds(s int64) Minutes {
	return Minutes(float64(s) / 60)
}

// Clamp returns m bounded to [lo, hi]. If lo > hi the bounds are swapped so the
// result is always within the intended range.
func (m Minutes) Clamp(lo, hi Minutes) Minutes {
	if lo > hi {
		lo, hi = hi, lo
	}
	switch {
	case m < lo:
		return lo
	case m > hi:
		return hi
	default:
		return m
	}
}

// roundedSeconds returns the value as whole seconds rounded half-up, plus the
// sign as a separate string ("" or "-"). It is the shared rounding core for the
// display formatters so String and Human never diverge on edge cases.
func (m Minutes) roundedSeconds() (sign string, secs int64) {
	s := float64(m) * 60
	if s < 0 {
		return "-", int64(-s + 0.5)
	}
	return "", int64(s + 0.5)
}

// String renders the value for display as "MM:SS", rounding to the nearest
// second. Negative values are rendered with a leading minus.
func (m Minutes) String() string {
	sign, total := m.roundedSeconds()
	return fmt.Sprintf("%s%02d:%02d", sign, total/60, total%60)
}

// Human renders the value as a compact, readable duration for the stats screen:
// "9h 25m", "25m", "2m 30s", "30s", or "0m" for an exact zero. Seconds only show
// when the total is under an hour and not a whole number of minutes.
func (m Minutes) Human() string {
	sign, total := m.roundedSeconds()
	h := total / 3600
	mi := (total % 3600) / 60
	se := total % 60

	switch {
	case h > 0 && mi == 0:
		return fmt.Sprintf("%s%dh", sign, h)
	case h > 0:
		return fmt.Sprintf("%s%dh %dm", sign, h, mi)
	case mi > 0 && se == 0:
		return fmt.Sprintf("%s%dm", sign, mi)
	case mi > 0:
		return fmt.Sprintf("%s%dm %ds", sign, mi, se)
	case se > 0:
		return fmt.Sprintf("%s%ds", sign, se)
	default:
		return "0m"
	}
}
