package logs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type RequestMeta struct {
	Status    int
	Path      string
	Domain    string
	Agent     string
	Method    string
	RemoteIP  string
	Query     string
	RequestID string
}

type OtelLogging interface {
	Debug(span trace.Span, args ...interface{})
	Debugf(span trace.Span, template string, args ...interface{})
	Info(span trace.Span, args ...interface{})
	Infof(span trace.Span, template string, args ...interface{})
	Warn(span trace.Span, args ...interface{})
	Warnf(span trace.Span, template string, args ...interface{})
	Error(span trace.Span, args ...interface{})
	Errorf(span trace.Span, template string, args ...interface{})
	DPanic(span trace.Span, args ...interface{})
	DPanicf(span trace.Span, template string, args ...interface{})
	Panic(span trace.Span, args ...interface{})
	Panicf(span trace.Span, template string, args ...interface{})
	Fatal(span trace.Span, args ...interface{})
	Fatalf(span trace.Span, template string, args ...interface{})
	Logf(span trace.Span, template string, args ...interface{})
	LogHttpResponse(span trace.Span, meta RequestMeta)
	LogJson(span trace.Span, label string, value interface{})
}

type otelLog struct {
	logger *zap.Logger
}

func NewOtelLogging(zapLogger *zap.Logger) OtelLogging {
	return &otelLog{logger: zapLogger}
}

func (l *otelLog) logSpan(span trace.Span, level, message string) (string, string) {
	if span != nil {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		span.SetAttributes(
			attribute.String("log.level", level),
			attribute.String("log.message", message),
			attribute.String("trace_id", traceID),
			attribute.String("span_id", spanID),
		)
		return traceID, spanID
	}
	return "", ""
}

func (l *otelLog) LogJson(span trace.Span, label string, value interface{}) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		l.logger.Error("Failed to marshal JSON",
			zap.String("label", label),
			zap.Error(err),
		)
		return
	}

	traceID, spanID := l.logSpan(span, "INFO", label)
	l.logger.Info("Logging JSON",
		zap.String(label, string(jsonBytes)),
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	)
}

func (l *otelLog) LogHttpResponse(span trace.Span, meta RequestMeta) {
	if span == nil {
		return
	}

	spanAttrs := []attribute.KeyValue{
		attribute.Int("http.status_code", meta.Status),
	}
	if meta.Path != "" {
		spanAttrs = append(spanAttrs, attribute.String("http.path", meta.Path))
	}
	if meta.Domain != "" {
		spanAttrs = append(spanAttrs, attribute.String("http.domain", meta.Domain))
	}
	if meta.Agent != "" {
		spanAttrs = append(spanAttrs, attribute.String("http.user_agent", meta.Agent))
	}
	if meta.Method != "" {
		spanAttrs = append(spanAttrs, attribute.String("http.method", meta.Method))
	}
	if meta.RemoteIP != "" {
		spanAttrs = append(spanAttrs, attribute.String("http.remote_ip", meta.RemoteIP))
	}
	if meta.Query != "" {
		spanAttrs = append(spanAttrs, attribute.String("http.query_params", meta.Query))
	}
	if meta.RequestID != "" {
		spanAttrs = append(spanAttrs, attribute.String("http.request_id", meta.RequestID))
	}

	span.SetAttributes(spanAttrs...)

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	logFields := []zap.Field{
		zap.Int("http_status", meta.Status),
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	}
	if meta.Path != "" {
		logFields = append(logFields, zap.String("http_path", meta.Path))
	}
	if meta.Domain != "" {
		logFields = append(logFields, zap.String("http_domain", meta.Domain))
	}
	if meta.Agent != "" {
		logFields = append(logFields, zap.String("user_agent", meta.Agent))
	}
	if meta.Method != "" {
		logFields = append(logFields, zap.String("http_method", meta.Method))
	}
	if meta.RemoteIP != "" {
		logFields = append(logFields, zap.String("remote_ip", meta.RemoteIP))
	}
	if meta.Query != "" {
		logFields = append(logFields, zap.String("query_params", meta.Query))
	}
	if meta.RequestID != "" {
		logFields = append(logFields, zap.String("request_id", meta.RequestID))
	}

	log := l.logger.With(logFields...)

	switch {
	case meta.Status >= 500:
		log.Error("Internal Server Error occurred", zap.Stack("stacktrace"))
	case meta.Status >= 400:
		log.Warn("Client error response recorded")
	case meta.Status >= 300:
		log.Info("Redirection response recorded")
	case meta.Status >= 200:
		log.Info("Successful response recorded")
	default:
		log.Info("Unexpected status code recorded")
	}
}

func BuildRequestMeta(r *http.Request, status int, started time.Time) RequestMeta {
	domain := r.URL.Hostname()
	if domain == "" {
		domain = r.Host
	}
	return RequestMeta{
		Status:    status,
		Path:      r.URL.Path,
		Domain:    domain,
		Agent:     r.UserAgent(),
		Method:    r.Method,
		RemoteIP:  r.RemoteAddr,
		Query:     r.URL.RawQuery,
		RequestID: r.Header.Get("X-Request-ID"),
	}
}
func (l *otelLog) Debug(span trace.Span, args ...interface{}) {
	msg := fmt.Sprint(args...)
	traceID, spanID := l.logSpan(span, "DEBUG", msg)
	l.logger.Debug(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID))
}

func (l *otelLog) Debugf(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "DEBUG", msg)
	l.logger.Sugar().Debugf(template, args...)
	l.logger.Debug("Formatted Debug", zap.String("trace_id", traceID), zap.String("span_id", spanID))
}

func (l *otelLog) Info(span trace.Span, args ...interface{}) {
	msg := fmt.Sprint(args...)
	traceID, spanID := l.logSpan(span, "INFO", msg)
	l.logger.Info(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID))
}

func (l *otelLog) Infof(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "INFO", msg)
	l.logger.Sugar().Infof(template, args...)
	l.logger.Info("Formatted Info", zap.String("trace_id", traceID), zap.String("span_id", spanID))
}

func (l *otelLog) Warn(span trace.Span, args ...interface{}) {
	msg := fmt.Sprint(args...)
	traceID, spanID := l.logSpan(span, "WARN", msg)
	l.logger.Warn(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID))
}

func (l *otelLog) Warnf(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "WARN", msg)
	l.logger.Sugar().Warnf(template, args...)
	l.logger.Warn("Formatted Warn", zap.String("trace_id", traceID), zap.String("span_id", spanID))
}

func (l *otelLog) Error(span trace.Span, args ...interface{}) {
	msg := fmt.Sprint(args...)
	traceID, spanID := l.logSpan(span, "ERROR", msg)
	l.logger.Error(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID), zap.Stack("stacktrace"))
}

func (l *otelLog) Errorf(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "ERROR", msg)
	l.logger.Sugar().Errorf(template, args...)
	l.logger.Error("Formatted Error", zap.String("trace_id", traceID), zap.String("span_id", spanID), zap.Stack("stacktrace"))
}

func (l *otelLog) DPanic(span trace.Span, args ...interface{}) {
	msg := fmt.Sprint(args...)
	traceID, spanID := l.logSpan(span, "DPANIC", msg)
	l.logger.DPanic(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID), zap.Stack("stacktrace"))
}

func (l *otelLog) DPanicf(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "DPANIC", msg)
	l.logger.Sugar().DPanicf(template, args...)
	l.logger.DPanic("Formatted DPanic", zap.String("trace_id", traceID), zap.String("span_id", spanID), zap.Stack("stacktrace"))
}

func (l *otelLog) Panic(span trace.Span, args ...interface{}) {
	msg := fmt.Sprint(args...)
	traceID, spanID := l.logSpan(span, "PANIC", msg)
	l.logger.Panic(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID), zap.Stack("stacktrace"))
}

func (l *otelLog) Panicf(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "PANIC", msg)
	l.logger.Panic(
		msg,
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.Stack("stacktrace"),
	)
}

func (l *otelLog) Fatal(span trace.Span, args ...interface{}) {
	msg := fmt.Sprint(args...)
	traceID, spanID := l.logSpan(span, "FATAL", msg)
	l.logger.Fatal(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID), zap.Stack("stacktrace"))
}

func (l *otelLog) Fatalf(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "FATAL", msg)

	l.logger.Fatal(
		msg,
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.Stack("stacktrace"),
	)
}

func (l *otelLog) Logf(span trace.Span, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	traceID, spanID := l.logSpan(span, "LOG", msg)
	l.logger.Info(msg, zap.String("trace_id", traceID), zap.String("span_id", spanID))
}
