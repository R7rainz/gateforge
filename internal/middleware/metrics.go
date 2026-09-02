package middleware

import (
	"net/http"
	"time"

	"github.com/r7rainz/gateforge/internal/metrics"
)

type metricsWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *metricsWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true

	w.ResponseWriter.WriteHeader(status)
}

func (w *metricsWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(body)
}

func (w *metricsWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func Metrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			writer := &metricsWriter{
				ResponseWriter: w,
			}

			next.ServeHTTP(writer, r)

			if writer.status == 0 {
				writer.status = http.StatusOK
			}

			m.Record(writer.status, time.Since(start))
		})
	}
}
