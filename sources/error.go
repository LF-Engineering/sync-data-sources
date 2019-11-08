package syncdatasources

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

// FatalOnError displays error message (if error present) and exits program
func FatalOnError(err error) string {
	if err != nil {
		tm := time.Now()
		Printf("Error(time=%+v):\nError: '%s'\nStacktrace:\n%s\n", tm, err.Error(), string(debug.Stack()))
		fmt.Fprintf(os.Stderr, "Error(time=%+v):\nError: '%s'\nStacktrace:\n", tm, err.Error())
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
		fmt.Fprintf(os.Stderr, "Error(time=%+v):\nError: '%s'\nStacktrace:\n", tm, err.Error())
		panic("stacktrace")
	}
	return OK
}
