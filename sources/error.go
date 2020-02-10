package syncdatasources

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

// GElasticURL - needed to be global, to redact it from error logs
var GElasticURL string

// FatalOnError displays error message (if error present) and exits program
func FatalOnError(err error) string {
	if err != nil {
		tm := time.Now()
		s := fmt.Sprintf("Error(time=%+v):\nError: '%s'\nStacktrace:\n%s\n", tm, err.Error(), string(debug.Stack()))
		s = strings.Replace(s, GElasticURL, Redacted, -1)
		Printf("%s", s)
		fmt.Fprintf(os.Stderr, "%s", s)
		panic("stacktrace")
	}
	return OK
}

// Fatalf - it will call FatalOnError using fmt.Errorf with args provided
func Fatalf(f string, a ...interface{}) {
	FatalOnError(fmt.Errorf(f, a...))
}

// FatalNoLog displays error message (if error present) and exits program, should be used for very early init state
func FatalNoLog(err error) string {
	if err != nil {
		tm := time.Now()
		s := fmt.Sprintf("Error(time=%+v):\nError: '%s'\nStacktrace:\n", tm, err.Error())
		s = strings.Replace(s, GElasticURL, Redacted, -1)
		fmt.Fprintf(os.Stderr, "%s", s)
		panic("stacktrace")
	}
	return OK
}
