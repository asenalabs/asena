package handler

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func LoggingMiddleware(logg *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Serve the request
		next.ServeHTTP(w, r)

		// Log after response
		logg.Info("incoming request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.Duration("latency", time.Since(start)),
		)
	})
}
