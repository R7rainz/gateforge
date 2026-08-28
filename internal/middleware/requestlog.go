package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(status int) {
	if sw.wroteHeader {
		return
	}

	sw.status = status
	sw.wroteHeader = true

	sw.ResponseWriter.WriteHeader(status)
}

func (sw *statusWriter) Write(data []byte) (int, error) {
	if !sw.wroteHeader {
		sw.WriteHeader(http.StatusOK)
	}

	return sw.ResponseWriter.Write(data)
}

func (sw *statusWriter) Unwrap() http.ResponseWriter {
	return sw.ResponseWriter
}

func generateRequestID() string {
	var bytes [10]byte

	_, err := rand.Read(bytes[:])
	if err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}

	return hex.EncodeToString(bytes[:])
}

func RequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := strings.TrimSpace(
				r.Header.Get("X-Request-ID"),
			)

			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set("X-Request-ID", requestID)
			r.Header.Set("X-Request-ID", requestID)

			sw := &statusWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(sw, r)

			logger.Info(
				"HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"request_id", requestID,
				"latency", time.Since(start),
			)
		})
	}
}
