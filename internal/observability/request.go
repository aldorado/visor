package observability

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

const requestIDHeader = "X-Request-ID"

func RequestIDMiddleware(next http.Handler) http.Handler {
	log := Component("http.request")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = newRequestID()
		}

		ctx := WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)
		w.Header().Set(requestIDHeader, requestID)

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()
		log.Debug(ctx, "request received", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		next.ServeHTTP(rec, r)
		log.Debug(ctx, "request completed", "method", r.Method, "path", r.URL.Path, "status", rec.statusCode, "duration_ms", time.Since(start).Milliseconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "req-fallback"
	}
	return hex.EncodeToString(buf)
}
