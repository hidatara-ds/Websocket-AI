package gateway

import (
	"context"
	"net/http"
	"time"
)

// Middleware represents HTTP middleware function
type Middleware func(http.Handler) http.Handler

// CORS middleware for handling cross-origin requests
func CORSMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header.Set("Expires", "0")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Logging middleware for request logging
func LoggingMiddleware(logger *Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			logger.Info("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// Rate limiting middleware
func RateLimitMiddleware(maxRequests int, window time.Duration) Middleware {
	// Simple in-memory rate limiter (in production, use Redis or similar)
	requests := make(map[string][]time.Time)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			now := time.Now()

			// Clean old requests
			if clientRequests, exists := requests[clientIP]; exists {
				var validRequests []time.Time
				for _, reqTime := range clientRequests {
					if now.Sub(reqTime) < window {
						validRequests = append(validRequests, reqTime)
					}
				}
				requests[clientIP] = validRequests
			}

			// Check rate limit
			if len(requests[clientIP]) >= maxRequests {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Add current request
			requests[clientIP] = append(requests[clientIP], now)

			next.ServeHTTP(w, r)
		})
	}
}

// Timeout middleware
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// Connection limit middleware
func ConnectionLimitMiddleware(maxConnections int, activeConnections *int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if *activeConnections >= maxConnections {
				http.Error(w, "Server at capacity", http.StatusServiceUnavailable)
				return
			}

			*activeConnections++
			defer func() { *activeConnections-- }()

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
