package pkg

import (
	"time"
)

type Drug struct {
	Name                  string
	PillsTotal, PillsLeft int
	TakingHour            int

	PillTakenTime YearMonthDay
}

func (d *Drug) Reset() {
	d.PillsLeft = d.PillsTotal
}

func (d *Drug) TakePill() {
	d.PillsLeft--
	if d.PillsLeft == 0 {
		d.Reset()
	}
	y, m, day := time.Now().Date()
	d.PillTakenTime = YearMonthDay{y, m, day}
}

func (d *Drug) IsAlreadyTaken(now time.Time) bool {
	return d.PillTakenTime.IsToday(now)
}

func (d *Drug) IsPillsRunOut() bool {
	if d.PillsLeft < 3 {
		return true
	}
	return false
}

func (d *Drug) PillsTakingHour() int {
	return d.TakingHour
}
