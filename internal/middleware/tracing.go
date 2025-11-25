package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/segmentio/ksuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

/*
LEARNING: DISTRIBUTED TRACING & OBSERVABILITY

Tracing helps you:
1. Track requests through the entire system
2. Identify performance bottlenecks
3. Debug errors with full context
4. Understand request flow in distributed systems

Key concepts:
- Trace: End-to-end request flow
- Span: Single operation in a trace
- Context: Passes trace information between functions

This middleware uses OpenTelemetry, the standard for distributed tracing.
*/

var tracer = otel.Tracer("ai-kms")

// TracingMiddleware adds distributed tracing to HTTP requests
// Learning: Creates a root span for each request and propagates context
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID using KSUID (time-ordered, for log correlation)
		requestID := ksuid.New().String()

		// Start a new span for this HTTP request
		// Learning: This is the "root span" for the request
		ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
				attribute.String("http.user_agent", r.Header.Get("User-Agent")),
				attribute.String("request.id", requestID),
			),
		)
		defer span.End()

		// Add request ID to context
		// Learning: Context is the Go way to pass request-scoped data
		ctx = context.WithValue(ctx, "request_id", requestID)

		// Wrap ResponseWriter to capture status code
		// Learning: Middleware pattern for capturing response metadata
		wrapped := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Add request ID to response header (for client-side correlation)
		w.Header().Set("X-Request-ID", requestID)

		// Record request start time
		startTime := time.Now()

		// Call the next handler with enriched context
		// Learning: Context propagates through the entire call chain
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Record response metadata in span
		duration := time.Since(startTime)
		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
			attribute.Int64("http.response_time_ms", duration.Milliseconds()),
		)

		// Mark span as error if status >= 400
		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
		}

		// Log completed request
		log.Printf("[%s] %s %s - %d (%dms)",
			requestID,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration.Milliseconds(),
		)
	})
}

// ErrorRecoveryMiddleware recovers from panics and records them in spans
// Learning: Always recover from panics in HTTP handlers to prevent server crashes
func ErrorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get current span from context
				span := trace.SpanFromContext(r.Context())

				// Record panic in span
				span.RecordError(fmt.Errorf("panic: %v", err))
				span.SetStatus(codes.Error, "panic recovered")
				span.SetAttributes(
					attribute.String("error.type", "panic"),
					attribute.String("error.stacktrace", string(debug.Stack())),
				)

				// Log the panic with stack trace
				requestID, _ := r.Context().Value("request_id").(string)
				log.Printf("[%s] PANIC: %v\n%s", requestID, err, debug.Stack())

				// Return 500 error to client
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Helper functions for creating spans in application code

// StartSpan creates a new span from the given context
// Use this in your service/repository methods to create child spans
//
// Example:
//
//	func (s *Service) DoSomething(ctx context.Context) error {
//	    ctx, span := middleware.StartSpan(ctx, "Service.DoSomething")
//	    defer span.End()
//	    // ... do work ...
//	}
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// AddSpanError records an error in the current span
// Use this when an error occurs to track it in tracing
func AddSpanError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// AddSpanEvent adds a named event to the current span
// Use this to mark important moments in the request lifecycle
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// GetRequestID extracts the request ID from context
// Useful for logging
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return "unknown"
}
