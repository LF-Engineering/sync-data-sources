package syncdatasources

import (
	"testing"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
)

func TestHash(t *testing.T) {
	// Test cases
	var testCases = []struct {
		str  string
		idx  int
		num  int
		hash int
		run  bool
	}{
		{str: "cncf-github-kubernetes-kubernetes", num: 4, idx: 0, hash: 1, run: false},
		{str: "cncf-github-kubernetes-kubernetes", num: 4, idx: 1, hash: 1, run: true},
		{str: "cncf-github-kubernetes-kubernetes", num: 4, idx: 2, hash: 1, run: false},
		{str: "cncf-github-kubernetes-kubernetes", num: 4, idx: 3, hash: 1, run: false},
		{str: "cncf-github-kubernetes-kubernetes", num: 5, idx: 0, hash: 4, run: false},
		{str: "cncf-github-kubernetes-kubernetes", num: 5, idx: 1, hash: 4, run: false},
		{str: "cncf-github-kubernetes-kubernetes", num: 5, idx: 2, hash: 4, run: false},
		{str: "cncf-github-kubernetes-kubernetes", num: 5, idx: 3, hash: 4, run: false},
		{str: "cncf-github-kubernetes-kubernetes", num: 5, idx: 4, hash: 4, run: true},
		{str: "hello", num: 3, idx: 0, hash: 1, run: false},
		{str: "hello", num: 3, idx: 1, hash: 1, run: true},
		{str: "hello", num: 3, idx: 2, hash: 1, run: false},
	}
	// Execute test cases
	for index, test := range testCases {
		expectedHash := test.hash
		expectedRun := test.run
		gotHash, gotRun := lib.Hash(test.str, test.idx, test.num)
		if gotHash != expectedHash || gotRun != expectedRun {
			t.Errorf(
				"test number %d, expected hash %v, got %v, expected run %v, got %v",
				index+1, expectedHash, gotHash, expectedRun, gotRun,
			)
		}
	}
}
