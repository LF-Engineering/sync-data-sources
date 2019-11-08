package syncdatasources

import (
	"testing"
	"time"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
	testlib "github.com/LF-Engineering/sync-data-sources/sources/test"
)

func TestHourStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 29, 12, 29, 3), expected: ft(2017, 8, 29, 12)},
		{time: ft(2017, 8, 29, 13), expected: ft(2017, 8, 29, 13)},
		{time: ft(2018), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.HourStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestNextHourStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 29, 12, 29, 3), expected: ft(2017, 8, 29, 13)},
		{time: ft(2017, 8, 29, 13), expected: ft(2017, 8, 29, 14)},
		{time: ft(2018), expected: ft(2018, 1, 1, 1)},
		{time: ft(2017, 12, 31, 23, 59, 59), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.NextHourStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestPrevHourStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 29, 12, 29, 3), expected: ft(2017, 8, 29, 11)},
		{time: ft(2017, 8, 29, 13), expected: ft(2017, 8, 29, 12)},
		{time: ft(2018), expected: ft(2017, 12, 31, 23)},
		{time: ft(2017, 12, 31, 23, 59, 59), expected: ft(2017, 12, 31, 22)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.PrevHourStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestDayStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 29, 12, 29, 3), expected: ft(2017, 8, 29, 0)},
		{time: ft(2017, 8, 29, 13), expected: ft(2017, 8, 29)},
		{time: ft(2018), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.DayStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestNextDayStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 29, 12, 29, 3), expected: ft(2017, 8, 30)},
		{time: ft(2017, 8, 31, 13), expected: ft(2017, 9, 1)},
		{time: ft(2018), expected: ft(2018, 1, 2)},
		{time: ft(2017, 12, 31, 23, 59, 59), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.NextDayStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestPrevDayStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 29, 12, 29, 3), expected: ft(2017, 8, 28)},
		{time: ft(2017, 8, 31, 13), expected: ft(2017, 8, 30)},
		{time: ft(2018), expected: ft(2017, 12, 31)},
		{time: ft(2017, 12, 31, 23, 59, 59), expected: ft(2017, 12, 30)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.PrevDayStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestWeekStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 8, 21)},
		{time: ft(2017, 8, 23, 13), expected: ft(2017, 8, 21)},
		{time: ft(2017, 8, 13), expected: ft(2017, 8, 7)},
		{time: ft(2017, 8, 14), expected: ft(2017, 8, 14)},
		{time: ft(2017, 8, 15), expected: ft(2017, 8, 14)},
		{time: ft(2017), expected: ft(2016, 12, 26)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.WeekStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestNextWeekStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 8, 28)},
		{time: ft(2017, 8, 23, 13), expected: ft(2017, 8, 28)},
		{time: ft(2017, 8, 13), expected: ft(2017, 8, 14)},
		{time: ft(2017, 8, 14), expected: ft(2017, 8, 21)},
		{time: ft(2017, 8, 15), expected: ft(2017, 8, 21)},
		{time: ft(2017, 12, 31), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.NextWeekStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestPrevWeekStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 8, 14)},
		{time: ft(2017, 8, 23, 13), expected: ft(2017, 8, 14)},
		{time: ft(2017, 8, 13), expected: ft(2017, 7, 31)},
		{time: ft(2017, 8, 14), expected: ft(2017, 8, 7)},
		{time: ft(2017, 8, 15), expected: ft(2017, 8, 7)},
		{time: ft(2017, 12, 31), expected: ft(2017, 12, 18)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.PrevWeekStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestMonthStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 8, 1)},
		{time: ft(2017), expected: ft(2017)},
		{time: ft(2017, 12, 10), expected: ft(2017, 12)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.MonthStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestNextMonthStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 9, 1)},
		{time: ft(2017), expected: ft(2017, 2)},
		{time: ft(2017, 12, 10), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.NextMonthStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestPrevMonthStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 7, 1)},
		{time: ft(2017), expected: ft(2016, 12)},
		{time: ft(2017, 12, 10), expected: ft(2017, 11)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.PrevMonthStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestQuarterStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 7, 1)},
		{time: ft(2017), expected: ft(2017)},
		{time: ft(2017, 12, 10), expected: ft(2017, 10)},
		{time: ft(2017, 10, 12), expected: ft(2017, 10)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.QuarterStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestNextQuarterStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 10)},
		{time: ft(2017), expected: ft(2017, 4)},
		{time: ft(2017, 12, 10), expected: ft(2018)},
		{time: ft(2017, 10, 12), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.NextQuarterStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestPrevQuarterStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017, 4)},
		{time: ft(2017), expected: ft(2016, 10)},
		{time: ft(2017, 12, 10), expected: ft(2017, 7)},
		{time: ft(2017, 10, 12), expected: ft(2017, 7)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.PrevQuarterStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestYearStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2017)},
		{time: ft(2017), expected: ft(2017)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.YearStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestNextYearStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2018)},
		{time: ft(2017), expected: ft(2018)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.NextYearStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}

func TestPrevYearStart(t *testing.T) {
	// Test cases
	ft := testlib.YMDHMS
	var testCases = []struct {
		time     time.Time
		expected time.Time
	}{
		{time: ft(2017, 8, 26, 12, 29, 3), expected: ft(2016)},
		{time: ft(2017), expected: ft(2016)},
	}
	// Execute test cases
	for index, test := range testCases {
		expected := test.expected
		got := lib.PrevYearStart(test.time)
		if got != expected {
			t.Errorf(
				"test number %d, expected %v, got %v",
				index+1, expected, got,
			)
		}
	}
}
