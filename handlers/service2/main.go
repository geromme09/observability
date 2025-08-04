package main

import (
	"log"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var tracer = otel.Tracer("receiver-service")

func handler(w http.ResponseWriter, r *http.Request) {
	// Extract context from incoming request headers
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// Start a new span with the extracted context
	_, span := tracer.Start(ctx, "incoming-request")
	defer span.End()

	// Business logic here
	log.Println("Handling request with trace ID:", span.SpanContext().TraceID().String())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello from receiver"))
}

func main() {
	http.HandleFunc("/receive", handler)
	log.Println("Receiver listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
