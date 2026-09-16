package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// JobWithDebounce marks a job as debounceable. When multiple dispatches with
// the same DebounceKey land within the DebounceFor window, only the most
// recent one runs — earlier dispatches are silently discarded at execution
// time (last-writer-wins).
//
// This differs from JobWithoutOverlapping (first-writer-wins, deduped at
// dispatch time). The two interfaces should not be combined on the same job.
type JobWithDebounce interface {
	Job

	DebounceKey() string
	DebounceFor() time.Duration
}

// JobWithMaxDebounceWait optionally caps how long a debounceable job can be
// deferred. Once MaxDebounceWait has elapsed since the first dispatch in the
// current burst, the next dispatch runs immediately even if the quiet window
// has not been reached. Implement this on a JobWithDebounce when you want
// the cap; otherwise debounces can defer indefinitely under continuous load.
type JobWithMaxDebounceWait interface {
	JobWithDebounce

	MaxDebounceWait() time.Duration
}

// DebounceCachePrefix namespaces debounce tokens in the shared cache.
const DebounceCachePrefix = "debounce:"

// DebounceFirstAtCachePrefix namespaces first-dispatch timestamps used by the
// maxWait cap.
const DebounceFirstAtCachePrefix = "debounce-first-at:"

// NewDebounceToken returns a random opaque token used to identify a single
// dispatch of a debounceable job.
func NewDebounceToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// Fall back to a time-based token if the system RNG fails. The token
		// only needs to be unique within the debounce window, not secret.
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}

// DebounceTTL returns the cache TTL to use for storing a debounce token. The
// token only needs to outlive the debounce window for the supersession check
// to remain correct; we use a generous multiple to absorb worker delays.
func DebounceTTL(window time.Duration) time.Duration {
	ttl := window * 10
	if ttl < 5*time.Minute {
		ttl = 5 * time.Minute
	}
	return ttl
}

type debounceTokenCtxKeyType struct{}

var debounceTokenCtxKey debounceTokenCtxKeyType

// WithDebounceToken stores the debounce token of the currently executing
// dispatch on the context so resolvers and middleware can read it back.
func WithDebounceToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, debounceTokenCtxKey, token)
}

// DebounceTokenFromContext returns the token previously attached with
// WithDebounceToken, or an empty string if none is present.
func DebounceTokenFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(debounceTokenCtxKey).(string); ok {
		return v
	}
	return ""
}

// DebounceCache is the minimal cache surface the debounce dispatch helper
// needs. It matches the methods exposed by core/cache.Cache so resolvers can
// pass their cache directly without an extra adapter.
type DebounceCache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// DebounceDispatch is the resolved outcome of preparing a dispatch for a
// debounceable job: the token to write into the queue payload (or asynq task
// ID) and the delay to schedule the job with. Delay is zero when the maxWait
// cap has been reached and the job should run immediately.
type DebounceDispatch struct {
	Token string
	Delay time.Duration
}

// PrepareDebounceDispatch writes a fresh debounce token to the cache and
// returns it together with the delay that should be applied to this dispatch.
// When maxWait is greater than zero and the first dispatch in the current
// burst happened more than maxWait ago, the returned delay is zero and the
// first-dispatch marker is cleared so the next burst starts a new window.
func PrepareDebounceDispatch(ctx context.Context, cache DebounceCache, debounceable JobWithDebounce) DebounceDispatch {
	window := debounceable.DebounceFor()
	token := NewDebounceToken()

	dispatch := DebounceDispatch{Token: token, Delay: window}
	if cache == nil {
		return dispatch
	}

	tokenKey := DebounceCachePrefix + debounceable.DebounceKey()
	if err := cache.Set(ctx, tokenKey, token, DebounceTTL(window)); err != nil {
		log.Error().Ctx(ctx).Err(err).Msgf("[queueing] failed to write debounce token for %s", debounceable.Name())
	}

	withMaxWait, hasMaxWait := debounceable.(JobWithMaxDebounceWait)
	if !hasMaxWait {
		return dispatch
	}
	maxWait := withMaxWait.MaxDebounceWait()
	if maxWait <= 0 {
		return dispatch
	}

	firstAtKey := DebounceFirstAtCachePrefix + debounceable.DebounceKey()
	rawFirstAt, found, err := cache.Get(ctx, firstAtKey)
	if err != nil {
		log.Warn().Ctx(ctx).Err(err).Msgf("[queueing] debounce: failed to read first-dispatch marker for %s", debounceable.Name())
		return dispatch
	}

	if !found {
		nowNanos := strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := cache.Set(ctx, firstAtKey, nowNanos, maxWait+window); err != nil {
			log.Warn().Ctx(ctx).Err(err).Msgf("[queueing] debounce: failed to write first-dispatch marker for %s", debounceable.Name())
		}
		return dispatch
	}

	firstAtNanos, parseErr := strconv.ParseInt(rawFirstAt, 10, 64)
	if parseErr != nil {
		log.Warn().Ctx(ctx).Err(parseErr).Msgf("[queueing] debounce: invalid first-dispatch marker for %s", debounceable.Name())
		return dispatch
	}

	elapsed := time.Since(time.Unix(0, firstAtNanos))
	if elapsed < maxWait {
		return dispatch
	}

	if err := cache.Delete(ctx, firstAtKey); err != nil {
		log.Warn().Ctx(ctx).Err(err).Msgf("[queueing] debounce: failed to clear first-dispatch marker for %s", debounceable.Name())
	}
	dispatch.Delay = 0
	return dispatch
}
