package greenapi

import (
	"log/slog"
	"net/http"
	"time"

	"green-api-test/internal/metrics"
)

const pathShapeLog = "waInstance/{idInstance}/{method}/{apiTokenInstance}"

type loggingTransport struct {
	base   http.RoundTripper
	logger *slog.Logger
}

func newLoggingTransport(base http.RoundTripper, logger *slog.Logger) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &loggingTransport{base: base, logger: logger}
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	dur := time.Since(start)

	op := operationFromContext(req.Context())
	metrics.RecordUpstreamRoundTrip(op, dur, resp, err)

	attrs := []slog.Attr{
		slog.String("component", "greenapi"),
		slog.String("method", req.Method),
		slog.Duration("duration", dur),
		slog.String("path_shape", pathShapeLog),
	}
	if op := operationFromContext(req.Context()); op != "" {
		attrs = append(attrs, slog.String("op", op))
	}
	if rid := req.Header.Get("X-Request-Id"); rid != "" {
		attrs = append(attrs, slog.String("request_id", rid))
	}
	if req.URL != nil {
		attrs = append(attrs, slog.String("host", req.URL.Host))
	}

	if err != nil {
		t.logger.LogAttrs(req.Context(), slog.LevelWarn, "greenapi request failed", append(attrs,
			slog.String("error", err.Error()),
		)...)
		return resp, err
	}
	attrs = append(attrs, slog.Int("status", resp.StatusCode))
	level := slog.LevelInfo
	if resp.StatusCode >= 500 {
		level = slog.LevelWarn
	}
	t.logger.LogAttrs(req.Context(), level, "greenapi request", attrs...)
	return resp, nil
}
