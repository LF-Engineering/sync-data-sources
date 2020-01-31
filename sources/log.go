package syncdatasources

import (
	"fmt"
	"sync"
	"time"
)

var (
	logCtx  *Ctx
	logOnce sync.Once
)

// Returns new context when not yet created
func newLogContext() *Ctx {
	var ctx Ctx
	ctx.Init()
	EnsureIndex(&ctx, "sdslog", true)
	return &ctx
}

// Printf is a wrapper around Printf(...) that supports logging.
func Printf(format string, args ...interface{}) (n int, err error) {
	// Initialize context once
	logOnce.Do(func() { logCtx = newLogContext() })

	// Actual logging to stdout & DB
	if logCtx.LogTime {
		n, err = fmt.Printf("%s: "+format, append([]interface{}{ToYMDHMSDate(time.Now())}, args...)...)
	} else {
		n, err = fmt.Printf(format, args...)
	}
	return
}
