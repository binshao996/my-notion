package middleware

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/google/uuid"
)

// contextKey is a private type to avoid key collisions in context.WithValue.
type contextKey string

const RequestIDKey contextKey = "request_id"

// Recoverer returns a middleware that recovers from panics, logs the error
// and stack trace, and returns a 500 Internal Server Error response.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC: %v\n%s", rec, debug.Stack())
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestID returns a middleware that injects a unique request ID into the
// context of each request. If the incoming request already has an
// X-Request-ID header, that value is reused; otherwise a new UUID is
// generated. The request ID is also set on the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := r.Context()
		ctx = context.WithValue(ctx, RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
