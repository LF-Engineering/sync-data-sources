package syncdatasources

import (
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SafeString - return safe string without control characters and unicode correct
func SafeString(str string) string {
	isOk := func(r rune) bool {
		return r < 32 || r >= 127
	}
	t := transform.Chain(norm.NFKD, transform.RemoveFunc(isOk))
	str, _, _ = transform.String(t, str)
	return str
}
