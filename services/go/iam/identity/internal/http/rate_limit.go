package httpapi

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	httpLib "github.com/tea0112/omnitat/libs/go/http"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/exception"
)

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
	Burst    int
	Message  string
}

type visitor struct {
	windowStarted time.Time
	requests      int
	lastSeen      time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	config   RateLimitConfig
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.Requests <= 0 {
		config.Requests = 5
	}
	if config.Window <= 0 {
		config.Window = time.Minute
	}
	if config.Burst <= 0 {
		config.Burst = config.Requests
	}
	if config.Message == "" {
		config.Message = "too many requests"
	}

	return &RateLimiter{
		visitors: map[string]*visitor{},
		config:   config,
	}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if key == "" {
			key = "unknown"
		}

		limiter := l.visitorFor(key)
		if !limiter.allow(l.config) {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(httpLib.ErrorResponseCode(exception.ErrRateLimited.Code, l.config.Message))
			return
		}

		next.ServeHTTP(w, r)
	})
}
func (l *RateLimiter) visitorFor(key string) *visitor {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	for visitorKey, existing := range l.visitors {
		if now.Sub(existing.lastSeen) > l.config.Window {
			delete(l.visitors, visitorKey)
		}
	}

	existing, ok := l.visitors[key]
	if ok {
		existing.lastSeen = now
		return existing
	}

	entry := &visitor{windowStarted: now, lastSeen: now}
	l.visitors[key] = entry
	return entry
}

func (v *visitor) allow(config RateLimitConfig) bool {
	now := time.Now().UTC()
	if now.Sub(v.windowStarted) >= config.Window {
		v.windowStarted = now
		v.requests = 0
	}

	limit := config.Requests
	if config.Burst > limit {
		limit = config.Burst
	}
	if v.requests >= limit {
		return false
	}

	v.requests++
	return true
}
