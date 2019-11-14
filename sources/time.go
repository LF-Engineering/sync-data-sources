package syncdatasources

import (
	"fmt"
	"os"
	"time"
)

// ProgressInfo display info about progress: i/n if current time >= last + period
// If displayed info, update last
func ProgressInfo(i, n int, start time.Time, last *time.Time, period time.Duration, msg string) {
	now := time.Now()
	if last.Add(period).Before(now) {
		perc := 0.0
		if n > 0 {
			perc = (float64(i) * 100.0) / float64(n)
		}
		eta := start
		if i > 0 && n > 0 {
			etaNs := float64(now.Sub(start).Nanoseconds()) * (float64(n) / float64(i))
			etaDuration := time.Duration(etaNs) * time.Nanosecond
			eta = start.Add(etaDuration)
			if msg != "" {
				Printf("%d/%d (%.3f%%), ETA: %v: %s\n", i, n, perc, eta, msg)
			} else {
				Printf("%d/%d (%.3f%%), ETA: %v\n", i, n, perc, eta)
			}
		} else {
			Printf("%s\n", msg)
		}
		*last = now
	}
}

// HourStart - return time rounded to current hour start
func HourStart(dt time.Time) time.Time {
	return time.Date(
		dt.Year(),
		dt.Month(),
		dt.Day(),
		dt.Hour(),
		0,
		0,
		0,
		time.UTC,
	)
}

// NextHourStart - return time rounded to next hour start
func NextHourStart(dt time.Time) time.Time {
	return HourStart(dt).Add(time.Hour)
}

// PrevHourStart - return time rounded to prev hour start
func PrevHourStart(dt time.Time) time.Time {
	return HourStart(dt).Add(-time.Hour)
}

// DayStart - return time rounded to current day start
func DayStart(dt time.Time) time.Time {
	return time.Date(
		dt.Year(),
		dt.Month(),
		dt.Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)
}

// NextDayStart - return time rounded to next day start
func NextDayStart(dt time.Time) time.Time {
	return DayStart(dt).AddDate(0, 0, 1)
}

// PrevDayStart - return time rounded to prev day start
func PrevDayStart(dt time.Time) time.Time {
	return DayStart(dt).AddDate(0, 0, -1)
}

// WeekStart - return time rounded to current week start
// Assumes first week day is Sunday
func WeekStart(dt time.Time) time.Time {
	wDay := int(dt.Weekday())
	// Go returns negative numbers for `modulo` operation when argument is negative
	// So instead of wDay-1 I'm using wDay+6
	subDays := (wDay + 6) % 7
	return DayStart(dt).AddDate(0, 0, -subDays)
}

// NextWeekStart - return time rounded to next week start
func NextWeekStart(dt time.Time) time.Time {
	return WeekStart(dt).AddDate(0, 0, 7)
}

// PrevWeekStart - return time rounded to prev week start
func PrevWeekStart(dt time.Time) time.Time {
	return WeekStart(dt).AddDate(0, 0, -7)
}

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

// NextMonthStart - return time rounded to next month start
func NextMonthStart(dt time.Time) time.Time {
	return MonthStart(dt).AddDate(0, 1, 0)
}

// PrevMonthStart - return time rounded to prev month start
func PrevMonthStart(dt time.Time) time.Time {
	return MonthStart(dt).AddDate(0, -1, 0)
}

// QuarterStart - return time rounded to current month start
func QuarterStart(dt time.Time) time.Time {
	month := ((dt.Month()-1)/3)*3 + 1
	return time.Date(
		dt.Year(),
		month,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
}

// NextQuarterStart - return time rounded to next quarter start
func NextQuarterStart(dt time.Time) time.Time {
	return QuarterStart(dt).AddDate(0, 3, 0)
}

// PrevQuarterStart - return time rounded to prev quarter start
func PrevQuarterStart(dt time.Time) time.Time {
	return QuarterStart(dt).AddDate(0, -3, 0)
}

// YearStart - return time rounded to current month start
func YearStart(dt time.Time) time.Time {
	return time.Date(
		dt.Year(),
		1,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
}

// NextYearStart - return time rounded to next year start
func NextYearStart(dt time.Time) time.Time {
	return YearStart(dt).AddDate(1, 0, 0)
}

// PrevYearStart - return time rounded to prev year start
func PrevYearStart(dt time.Time) time.Time {
	return YearStart(dt).AddDate(-1, 0, 0)
}

// TimeParseAny - attempts to parse time from string YYYY-MM-DD HH:MI:SS
// Skipping parts from right until only YYYY id left
func TimeParseAny(dtStr string) time.Time {
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15",
		"2006-01-02",
		"2006-01",
		"2006",
	}
	for _, format := range formats {
		t, e := time.Parse(format, dtStr)
		if e == nil {
			return t
		}
	}
	Printf("Error:\nCannot parse date: '%v'\n", dtStr)
	fmt.Fprintf(os.Stdout, "Error:\nCannot parse date: '%v'\n", dtStr)
	os.Exit(1)
	return time.Now()
}

// ToYMDDate - return time formatted as YYYY-MM-DD
func ToYMDDate(dt time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d", dt.Year(), dt.Month(), dt.Day())
}

// ToYMDHMSDate - return time formatted as YYYY-MM-DD HH:MI:SS
func ToYMDHMSDate(dt time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", dt.Year(), dt.Month(), dt.Day(), dt.Hour(), dt.Minute(), dt.Second())
}
