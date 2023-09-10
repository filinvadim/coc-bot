package pkg

import "time"

var Weekdays = map[time.Weekday]string{
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
	time.Sunday:    "Воскресенье",
}

func GetWeekdayName(now time.Time) string {
	return Weekdays[now.Weekday()]
}

type YearMonthDay struct {
	Y int
	M time.Month
	D int
}

func (ymd *YearMonthDay) IsToday(now time.Time) bool {
	y, m, d := now.Date()
	return y == ymd.Y && m == ymd.M && d == ymd.D
}
