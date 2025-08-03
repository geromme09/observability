package handlers

import (
	"net/http"
	"observability/logs"

	"go.opentelemetry.io/otel"
)

func TestHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tracer := otel.Tracer("test-service")
		_, span := tracer.Start(ctx, "Handle /test")
		defer span.End()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Log triggered!"))
	}
}

func TestErrorHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tracer := otel.Tracer("test-service")
		ctx, span := tracer.Start(ctx, "Handle /test-error")
		defer span.End()

		l.Info(span, "Started /test-error handler")

		dbCtx, dbSpan := tracer.Start(ctx, "DB Query Error")
		l.Info(dbSpan, "Attempting DB Query...")
		l.Error(dbSpan, "DB connection failed: timeout")
		dbSpan.End()

		bizCtx, bizSpan := tracer.Start(dbCtx, "Business Logic Error")
		l.Info(bizSpan, "Running business logic...")
		l.Errorf(bizSpan, "Validation failed for user input: %v", "missing email")
		bizSpan.End()

		_, apiSpan := tracer.Start(bizCtx, "External API Error")
		l.Info(apiSpan, "Calling third-party API...")
		l.Error(apiSpan, "External API responded with 500 Internal Server Error")
		apiSpan.End()

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error scenario completed! Logs captured."))
	}
}
