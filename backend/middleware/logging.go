package middleware

import (
	"log"
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		reqID := chiMiddleware.GetReqID(r.Context())
		duration := time.Since(start)

		if rw.statusCode >= 500 {
			log.Printf("[ERROR] [req:%s] %s %s %d %s", reqID, r.Method, r.URL.Path, rw.statusCode, duration)
		} else if rw.statusCode >= 400 {
			log.Printf("[WARN]  [req:%s] %s %s %d %s", reqID, r.Method, r.URL.Path, rw.statusCode, duration)
		} else {
			log.Printf("[INFO]  [req:%s] %s %s %d %s", reqID, r.Method, r.URL.Path, rw.statusCode, duration)
		}
	})
}
