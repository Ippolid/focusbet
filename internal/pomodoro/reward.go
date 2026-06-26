package pomodoro

import (
	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/economy"
)

// Reward is what a completed focus session yields under the single-currency
// model: a break of Fair minutes, which is what gets banked. There is no
// base/max corridor — resting and gambling both just spend banked minutes.
//
// Fair is scaled down when the session was stopped before its planned length.
type Reward struct {
	FocusedMinutes domain.Minutes // minutes actually focused this session
	Fair           domain.Minutes // earned break minutes (banked / available to rest)
}

// ComputeReward derives the reward for a session: the break (breakMinutes, the
// preset's short break) scaled by completion (focused/planned, clamped to 1).
func ComputeReward(focusedMinutes, plannedFocus, breakMinutes domain.Minutes, _ domain.Economy) Reward {
	return Reward{
		FocusedMinutes: focusedMinutes,
		Fair:           economy.EarnForSession(focusedMinutes, plannedFocus, breakMinutes),
	}
}

// EarnMinutes is what goes into the bank for this session: the earned break.
func (r Reward) EarnMinutes() domain.Minutes {
	if r.Fair < 0 {
		return 0
	}
	return r.Fair
}
