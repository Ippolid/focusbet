package domain

import "time"

type Balance struct {
	Minutes     Minutes `json:"bank_minutes"` // earned, not yet spent
	DailyLimit  Minutes `json:"daily_limit"`  //daily spending limit
	WeeklyLimit Minutes `json:"weekly_limit"` //weekly spending limit

	LastUpdateAt int64 `json:"last_update_at"` //unix timestamp
}

// Work with minutes
func (b *Balance) AddMinutes(a Minutes) {
	b.Minutes.Add(a)
	b.LastUpdateAt = time.Now().Unix()
}

func (b *Balance) GetMinutes() Minutes {
	return b.Minutes
}

// Work with limits
func (b *Balance) GetDailyLimits() Minutes {
	return b.DailyLimit
}

func (b *Balance) GetWeeklyLimits() Minutes {
	return b.DailyLimit
}

func (b *Balance) UpdateDailyLimits(a Minutes) {
	b.DailyLimit = a
	b.LastUpdateAt = time.Now().Unix()
}

func (b *Balance) UpdateWeeklyLimits(a Minutes) {
	b.WeeklyLimit = a
	b.LastUpdateAt = time.Now().Unix()
}

// Work with time
func (b *Balance) GetLastUpdateAt() int64 {
	return b.LastUpdateAt
}
