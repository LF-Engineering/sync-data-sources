package hnanalysis

import (
	"fmt"
	"time"
)

// MonthStart - return time rounded to current month start
func MonthStart(dt time.Time) time.Time {
	return time.Date(
		dt.Year(),
		dt.Month(),
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
}

// ToYMDDate - return time formatted as YYYY-MM-DD
func ToYMDDate(dt time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d", dt.Year(), dt.Month(), dt.Day())
}

// TimeAry sortable time array
type TimeAry []time.Time

func (p TimeAry) Len() int {
	return len(p)
}

func (p TimeAry) Less(i, j int) bool {
	return p[i].Before(p[j])
}

func (p TimeAry) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
