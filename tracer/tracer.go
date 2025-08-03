package tracer

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var ELASTIC_APM_SERVICE_NAME = "test-service"
var ELASTIC_APM_API_KEY = "RUU2ZmE1Z0JCMnI4REdVdi1zVDk6aS1sZlp0RVJhemJ4YU15djc0cmhGQQ=="
var ELASTIC_APM_ENDPOINT = "https://my-observability-project-b29ff9.apm.us-central1.gcp.elastic.cloud:443"

func InitTracer() func() {
	ctx := context.Background()
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(ELASTIC_APM_ENDPOINT),
		// otlptracehttp.WithInsecure(),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "ApiKey " + ELASTIC_APM_API_KEY,
		}),
	)
	if err != nil {
		panic(err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(ELASTIC_APM_SERVICE_NAME),
			semconv.DeploymentEnvironment("TEST"),
			semconv.TelemetrySDKLanguageGo,
		)),
	)
	otel.SetTracerProvider(tp)
	return func() {
		_ = tp.Shutdown(ctx)
	}
}
