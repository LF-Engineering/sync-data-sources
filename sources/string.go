package syncdatasources

import (
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SafeString - return safe string without control characters and unicode correct
// Other options would be to replace non-OK characters with "%HH" - their hexcode
// ES would understand this
func SafeString(str string) string {
	isOk := func(r rune) bool {
		return r < 32 || r >= 127
	}
	t := transform.Chain(norm.NFKD, transform.RemoveFunc(isOk))
	str, _, _ = transform.String(t, str)
	return str
}
