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

// PrintfRedacted is a wrapper around Printf(...) that supports logging.
func PrintfRedacted(format string, args ...interface{}) (n int, err error) {
	msg := fmt.Sprintf(format, args...)
	msg = strings.Replace(msg, GElasticURL, Redacted, -1)
	return fmt.Printf("%s", msg)
}

// Printf is a wrapper around Printf(...) that supports logging.
func Printf(format string, args ...interface{}) (n int, err error) {
	// Initialize context once
	logOnce.Do(func() { logCtx = newLogContext() })

	// Actual logging to stdout & DB
	now := time.Now()
	var msg string
	if logCtx.LogTime {
		msg = fmt.Sprintf("%s: "+format, append([]interface{}{ToYMDHMSDate(now)}, args...)...)
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	msg = strings.Replace(msg, GElasticURL, Redacted, -1)
	n, err = fmt.Printf("%s", msg)
	if logCtx.SkipEsLog {
		return
	}
	err = EsLog(logCtx, msg, now)
	return
}
