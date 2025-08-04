package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"observability/logs"
	"observability/middleware"
	"observability/tracer"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var ELASTIC_APM_SERVICE_NAME = "test-service"
var ELASTIC_APM_API_KEY = "RUU2ZmE1Z0JCMnI4REdVdi1zVDk6aS1sZlp0RVJhemJ4YU15djc0cmhGQQ=="
var ELASTIC_APM_ENDPOINT = "https://my-observability-project-b29ff9.apm.us-central1.gcp.elastic.cloud:443"

func main() {
	l, err := logs.NewOtelLoggerBuilder().
		WithEndpointUrl(ELASTIC_APM_ENDPOINT).
		WithServiceName(ELASTIC_APM_SERVICE_NAME).
		WithAuthHeader(ELASTIC_APM_API_KEY).
		Build(context.Background())
	if err != nil {
		panic(err)
	}

	cleanup := tracer.InitTracer()
	defer cleanup()

	mux := http.NewServeMux()
	mux.Handle("/test", TestHandler(l))
	mux.Handle("/test-error", TestErrorHandler(l))
	mux.Handle("/greet", GreetHandler(l))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	wrapped := middleware.TraceMiddleware(ELASTIC_APM_SERVICE_NAME, l)(mux)

	l.Info(nil, "Starting server on :8080")
	http.ListenAndServe(":8080", wrapped)
}

func TestHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		tracer := otel.Tracer("test-service")
		ctx, span := tracer.Start(ctx, "client-request")
		defer span.End()

		req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/test-error", nil)
		if err != nil {
			l.Error(span, "Failed to create outbound request", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "Request creation failed", http.StatusInternalServerError)
			return
		}

		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			l.Error(span, "Outbound request failed", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "Outbound call failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		l.Info(span, "Outbound request completed", map[string]interface{}{
			"status": resp.Status,
		})

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Log triggered!"))
	}
}

func TestErrorHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Propagate context from incoming request
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
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
		l.Error(span, "Error scenario completed", map[string]interface{}{
			"error": "Simulated error for testing"})

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error scenario completed! Logs captured."))
	}
}

type GreetRequest struct {
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

type GreetResponse struct {
	Message string `json:"message"`
}

func GreetHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		tracer := otel.Tracer("test-service")
		_, span := tracer.Start(ctx, "Handle /greet")
		defer span.End()

		var req GreetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			l.Error(span, "Invalid JSON", err.Error())
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		l.LogJson(span, "request_body", req)

		resp := GreetResponse{
			Message: fmt.Sprintf("Hello %s %s!", req.Name, req.Surname),
		}

		l.LogJson(span, "response_body", resp)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
