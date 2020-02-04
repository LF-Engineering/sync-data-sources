package syncdatasources

import (
	"fmt"
	"strings"
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
	if !ctx.SkipEsLog {
		EnsureIndex(&ctx, "sdslog", true)
	}
	return &ctx
}

// Printf is a wrapper around Printf(...) that supports logging.
func Printf(format string, args ...interface{}) (n int, err error) {
	// Initialize context once
	logOnce.Do(func() { logCtx = newLogContext() })

	// Actual logging to stdout & DB
	now := time.Now()
	var msg string
	format = strings.Replace(format, "%%", "%%%%", -1)
	if logCtx.LogTime {
		msg = fmt.Sprintf("%s: "+format, append([]interface{}{ToYMDHMSDate(now)}, args...)...)
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	n, err = fmt.Printf(msg)
	if logCtx.SkipEsLog {
		return
	}
	err = EsLog(logCtx, msg, now)
	return
}
