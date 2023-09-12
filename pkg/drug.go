package pkg

import (
	"time"
)

type Drug struct {
	Name                  string
	PillsTotal, PillsLeft int
	TakingHour            int

	PillTakenTime time.Time
}

func (d *Drug) Reset() {
	d.PillsLeft = d.PillsTotal
}

func (d *Drug) TakePill() {
	d.PillsLeft--
	if d.PillsLeft == 0 {
		d.Reset()
	}
}

func (d *Drug) IsAlreadyTaken(now time.Time) bool {
	y, m, day := now.Date()

	dy, dm, dday := d.PillTakenTime.Date()
	return dy == y && dm == m && dday == day
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
