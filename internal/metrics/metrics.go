package metrics

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

// Metric names are prefixed to avoid collisions when this binary is embedded elsewhere.
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenapi_facade_http_requests_total",
			Help: "Total HTTP requests handled by this service.",
		},
		[]string{"method", "route", "status_class"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "greenapi_facade_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	upstreamRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenapi_facade_upstream_requests_total",
			Help: "Total outbound HTTP requests to Green-API (each attempt, including retries).",
		},
		[]string{"op"},
	)
	upstreamRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "greenapi_facade_upstream_request_duration_seconds",
			Help:    "Outbound Green-API request duration in seconds.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"op"},
	)
	upstreamErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "greenapi_facade_upstream_errors_total",
			Help: "Upstream attempts that ended in transport failure, cancel, HTTP 5xx, or 429.",
		},
		[]string{"op", "kind"},
	)
)

// Register adds all application metrics to reg. Safe to call once at startup.
func Register(reg prometheus.Registerer) {
	reg.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		upstreamRequestsTotal,
		upstreamRequestDuration,
		upstreamErrorsTotal,
	)
}

// HTTPMiddleware records request counts and latency using chi's route pattern (low cardinality).
func HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			statusClass := statusClass(ww.Status())
			httpRequestsTotal.WithLabelValues(r.Method, route, statusClass).Inc()
			httpRequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}

func statusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return strconv.Itoa(code)
	}
}

// RecordUpstreamRoundTrip observes one outbound HTTP round trip (including retries as separate events).
func RecordUpstreamRoundTrip(op string, duration time.Duration, resp *http.Response, rtErr error) {
	if op == "" {
		op = "unknown"
	}
	upstreamRequestsTotal.WithLabelValues(op).Inc()
	upstreamRequestDuration.WithLabelValues(op).Observe(duration.Seconds())

	switch {
	case rtErr != nil:
		if errors.Is(rtErr, context.Canceled) {
			upstreamErrorsTotal.WithLabelValues(op, "canceled").Inc()
			return
		}
		upstreamErrorsTotal.WithLabelValues(op, "transport").Inc()
	case resp != nil && resp.StatusCode == http.StatusTooManyRequests:
		upstreamErrorsTotal.WithLabelValues(op, "rate_limit").Inc()
	case resp != nil && resp.StatusCode >= 500:
		upstreamErrorsTotal.WithLabelValues(op, "http_5xx").Inc()
	}
}
