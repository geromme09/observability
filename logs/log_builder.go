package logs

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Header map[string]string

type OtelLoggerBuilder struct {
	logExporterOpts    []otlploggrpc.Option
	serviceName        string
	useConsoleExporter bool
}

func NewOtelLoggerBuilder() *OtelLoggerBuilder {
	return &OtelLoggerBuilder{}
}

func (b *OtelLoggerBuilder) WithEndpointUrl(endpointUrl string) *OtelLoggerBuilder {
	b.logExporterOpts = append(b.logExporterOpts, otlploggrpc.WithEndpointURL(endpointUrl))
	return b
}

func (b *OtelLoggerBuilder) WithHeaders(headers Header) *OtelLoggerBuilder {
	headerMap := make(map[string]string)
	for key, value := range headers {
		headerMap[key] = value
	}
	b.logExporterOpts = append(b.logExporterOpts, otlploggrpc.WithHeaders(headerMap))
	return b
}

func (b *OtelLoggerBuilder) WithAuthHeader(token string) *OtelLoggerBuilder {
	return b.WithHeaders(Header{
		"Authorization": "ApiKey " + token,
	})
}

func (b *OtelLoggerBuilder) WithServiceName(serviceName string) *OtelLoggerBuilder {
	b.serviceName = serviceName
	return b
}

func (b *OtelLoggerBuilder) WithConsoleExporter() *OtelLoggerBuilder {
	b.useConsoleExporter = true
	return b
}

func (b *OtelLoggerBuilder) Build(ctx context.Context) (OtelLogging, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(b.serviceName),
			semconv.DeploymentEnvironment("TEST"),
			semconv.TelemetrySDKLanguageGo,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter sdklog.Exporter
	if b.useConsoleExporter {
		exporter, err = stdoutlog.New(stdoutlog.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
	} else {
		exporter, err = otlploggrpc.New(ctx, b.logExporterOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP log exporter with options %v: %w", b.logExporterOpts, err)
		}
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	global.SetLoggerProvider(provider)

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(zapcore.Lock(os.Stdout)), zap.DebugLevel),
		otelzap.NewCore(b.serviceName, otelzap.WithLoggerProvider(provider)),
	)
	zapLogger := zap.New(core)
	defer zapLogger.Sync()

	return NewOtelLogging(zapLogger), nil
}
