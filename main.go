package main

import (
	"context"
	"net/http"
	"observability/handlers"
	"observability/logs"
	"observability/middleware"
	"observability/tracer"
	"os"
)

var (
	ELASTIC_APM_SERVICE_NAME = os.Getenv("ELASTIC_APM_SERVICE_NAME")
	ELASTIC_APM_API_KEY      = os.Getenv("ELASTIC_APM_API_KEY")
	ELASTIC_APM_ENDPOINT     = os.Getenv("ELASTIC_APM_ENDPOINT")
)

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
	mux.Handle("/test", handlers.TestHandler(l))
	mux.Handle("/test-error", handlers.TestErrorHandler(l))

	// Apply middleware
	wrapped := middleware.TraceMiddleware(ELASTIC_APM_SERVICE_NAME, l)(mux)

	l.Info(nil, "Starting server on :8080")
	http.ListenAndServe(":8080", wrapped)
}
