package middleware

import (
	"net/http"

	"github.com/r7rainz/gateforge/internal/ratelimit"
)

func RateLimit(limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := ratelimit.ClientIP(r.RemoteAddr)

			if !limiter.Allow(clientIP) {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
