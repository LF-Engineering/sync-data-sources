package syncdatasources

import (
	"strconv"
	"strings"

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

// StringToBool - convert string value to boolean value
// returns false for anything that was parsed as false, zero, empty etc:
// f, F, false, False, fALSe, 0, "", 0.00
// else returns true
func StringToBool(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err == nil {
		return b
	}
	f, err := strconv.ParseFloat(v, 64)
	if err == nil {
		return f != 0.0
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return i != 0
	}
	if v == "no" || v == "n" {
		return false
	}
	return true
}
