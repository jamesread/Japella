// Package debuglog provides fine-grained, runtime-toggleable debug logging.
// Flags are stored as Settings cvars (debug.*) and cached in memory so hot paths
// never hit the database. When a category is enabled, messages are emitted at
// Info (with debug_category) so they appear without raising global logLevel.
package debuglog

import (
	"strings"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

// Cvar key names (also registered in db.CvarList).
const (
	KeyAuth = "debug.auth"
	KeyFeed = "debug.feed"
	KeyHTTP = "debug.http"
)

var (
	authEnabled atomic.Bool
	feedEnabled atomic.Bool
	httpEnabled atomic.Bool
)

// Set updates the in-memory flag for a debug.* cvar key. Unknown keys are ignored.
func Set(key string, enabled bool) {
	switch key {
	case KeyAuth:
		authEnabled.Store(enabled)
	case KeyFeed:
		feedEnabled.Store(enabled)
	case KeyHTTP:
		httpEnabled.Store(enabled)
	}
}

// Init loads all known debug flags from a key→enabled map (typically from cvars at startup).
func Init(flags map[string]bool) {
	for key, enabled := range flags {
		Set(key, enabled)
	}
}

// IsDebugCvar reports whether key is a debug.* flag this package manages.
func IsDebugCvar(key string) bool {
	return strings.HasPrefix(key, "debug.")
}

func AuthEnabled() bool { return authEnabled.Load() }
func FeedEnabled() bool { return feedEnabled.Load() }
func HTTPEnabled() bool { return httpEnabled.Load() }

func entry(category string) *log.Entry {
	return log.WithField("debug_category", category)
}

// Authf logs when debug.auth is enabled.
func Authf(format string, args ...any) {
	if !authEnabled.Load() {
		return
	}
	entry("auth").Infof(format, args...)
}

// Feedf logs when debug.feed is enabled.
func Feedf(format string, args ...any) {
	if !feedEnabled.Load() {
		return
	}
	entry("feed").Infof(format, args...)
}

// HTTPf logs when debug.http is enabled.
func HTTPf(format string, args ...any) {
	if !httpEnabled.Load() {
		return
	}
	entry("http").Infof(format, args...)
}

// HTTP logs with structured fields when debug.http is enabled.
func HTTP(fields log.Fields, msg string) {
	if !httpEnabled.Load() {
		return
	}
	entry("http").WithFields(fields).Info(msg)
}
