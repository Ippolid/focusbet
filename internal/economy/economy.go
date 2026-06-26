// Package economy holds the rest-bank math as pure, stateless functions. The
// model is single-currency: the bank holds rest minutes. Focusing earns minutes;
// resting and gambling spend them. There is no separate base/fair/max corridor —
// the break you earn is what goes into the bank, and games (RTP < 1) risk it.
//
// The unit throughout is domain.Minutes. Functions read no clock and no global
// state, so the package is trivially testable.
package economy

import "github.com/Ippolid/focusbet/internal/domain"

// EarnForSession is the rest credited for a completed focus session: the break
// it earned, scaled by how much of the planned focus was actually done.
//
//	earn = breakMinutes × clamp(focused/planned, 0, 1)
//
// A full session banks the whole break; stopping at half the planned focus banks
// half the break. planned ≤ 0 is treated as a full session.
func EarnForSession(focused, planned, breakMinutes domain.Minutes) domain.Minutes {
	if breakMinutes <= 0 {
		return 0
	}
	completion := 1.0
	if planned > 0 {
		completion = float64(focused) / float64(planned)
		if completion < 0 {
			completion = 0
		}
		if completion > 1 {
			completion = 1
		}
	}
	return domain.Minutes(float64(breakMinutes) * completion)
}
