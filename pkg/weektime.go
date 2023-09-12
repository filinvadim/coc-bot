package pkg

import (
	"time"
)

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
