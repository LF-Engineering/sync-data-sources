package syncdatasources

import (
	"strings"
	"sync"
)

var (
	// GRedactedStrings - need to be global, to redact them from error logs
	GRedactedStrings map[string]struct{}
	// GRedactedMtx - guard access to this map while in MT
	GRedactedMtx *sync.RWMutex
	redactedOnce sync.Once
)

// AddRedacted - adds redacted string
func AddRedacted(newRedacted string, useMutex bool) {
	// Initialize map & mutex once
	redactedOnce.Do(func() {
		GRedactedStrings = make(map[string]struct{})
		GRedactedMtx = &sync.RWMutex{}
	})
	if useMutex {
		GRedactedMtx.Lock()
		defer func() {
			GRedactedMtx.Unlock()
		}()
	}
	if len(newRedacted) > 3 {
		GRedactedStrings[newRedacted] = struct{}{}
	}
}

// FilterRedacted - filter out all known redacted starings
func FilterRedacted(str string) string {
	GRedactedMtx.RLock()
	defer func() {
		GRedactedMtx.RUnlock()
	}()
	for redacted := range GRedactedStrings {
		str = strings.Replace(str, redacted, Redacted, -1)
	}
	return str
}

// GetRedacted - get redacted
func GetRedacted() (str string) {
	GRedactedMtx.RLock()
	defer func() {
		GRedactedMtx.RUnlock()
	}()
	str = "["
	for redacted := range GRedactedStrings {
		str += redacted + " "
	}
	str += "]"
	return
}
