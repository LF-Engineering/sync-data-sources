package hnanalysis

import (
	"testing"
	"time"

	lib "github.com/cncf/hnanalysis"
	testlib "github.com/cncf/hnanalysis/test"
)

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
