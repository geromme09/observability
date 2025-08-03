package middleware

import (
	"net/http"
	"observability/logs"

	"go.opentelemetry.io/otel"
)

func TraceMiddleware(serviceName string, logger logs.OtelLogging) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx, span := tracer.Start(r.Context(), r.URL.Path)
			defer span.End()

			// Wrap ResponseWriter to capture status
			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(recorder, r.WithContext(ctx))

			meta := logs.RequestMeta{
				Status:    recorder.statusCode,
				Path:      r.URL.Path,
				Domain:    r.URL.Hostname(),
				Agent:     r.UserAgent(),
				Method:    r.Method,
				RemoteIP:  r.RemoteAddr,
				Query:     r.URL.RawQuery,
				RequestID: r.Header.Get("X-Request-ID"),
			}
			if meta.Domain == "" {
				meta.Domain = r.Host
			}

			logger.LogHttpResponse(span, meta)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
