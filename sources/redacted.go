package syncdatasources

import (
	"fmt"
	"strings"
	"sync"
)

var (
	// GRedactedStrings - need to be global, to redact them from error logs
	GRedactedStrings map[string]struct{}
	// GRedactedMtx - guard access to this map while in MT
	GRedactedMtx *sync.Mutex
	redactedOnce sync.Once
)

// AddRedacted - adds redacted string
func AddRedacted(newRedacted string, useMutex bool) {
	// Initialize map & mutex once
	redactedOnce.Do(func() {
		GRedactedStrings = make(map[string]struct{})
		GRedactedMtx = &sync.Mutex{}
	})
	if useMutex {
		GRedactedMtx.Lock()
		defer func() {
			GRedactedMtx.Unlock()
		}()
	}
	if len(newRedacted) > 3 {
		_, ok := GRedactedStrings[newRedacted]
		if !ok {
			fmt.Printf("Adding redacted %d\n", len(newRedacted))
		}
		GRedactedStrings[newRedacted] = struct{}{}
	}
}

// FilterRedacted - filter out all known redacted starings
func FilterRedacted(str string) string {
	for redacted := range GRedactedStrings {
		str = strings.Replace(str, redacted, Redacted, -1)
	}
	return str
}
