// Package cachecontrol marks responses that a shared cache must never store.
package cachecontrol

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CacheControlHeader = "Cache-Control"
	VaryHeader         = "Vary"

	// privateDirective keeps the response out of every cache between the API
	// and the browser. no-store does the work at the Cloudflare edge: with
	// Origin Cache Control on, it overrides a "Cache Everything" rule, so a
	// per-user response cannot be stored and replayed to another visitor.
	privateDirective = "private, no-store"
)

// Private marks the response as belonging to one session.
//
// Apply it to every route that reads the session cookie. Cloudflare varies on
// Accept-Encoding only, so Vary alone cannot protect the response there. Vary
// is still sent for the caches that do honour it.
func Private(context *gin.Context) {
	context.Header(CacheControlHeader, privateDirective)
	AddVary(context, "Cookie")
}

// AddVary appends a field to Vary, keeping the fields already set. CORS writes
// Vary: Origin, so replacing the header would drop it.
func AddVary(context *gin.Context, field string) {
	header := context.Writer.Header()

	for _, value := range header.Values(VaryHeader) {
		for existing := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(existing), field) {
				return
			}
		}
	}

	header.Add(VaryHeader, field)
}
