package domain

type Minutes float64

func (m *Minutes) Add(amount Minutes) {
	*m += amount
}

func (m *Minutes) Dec(amount Minutes) {
	*m -= amount
}
