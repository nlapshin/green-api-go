package greenapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"green-api-test/internal/jsonfmt"
)

const (
	publicBodySnippetRunes = 512
	authSnippetRunes       = 120
)

type MappedIntegrationError struct {
	HTTPStatus int
	APICode    string
	APIMessage string
	Details    any

	LogLevel slog.Level
	LogMsg   string
	LogAttrs []slog.Attr
}

func MapIntegrationError(err error) MappedIntegrationError {
	if err == nil {
		return MappedIntegrationError{}
	}

	var rt *RoundTripError
	if errors.As(err, &rt) {
		return mapRoundTripError(rt)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return MappedIntegrationError{
			HTTPStatus: http.StatusGatewayTimeout,
			APICode:    "upstream_timeout",
			APIMessage: "GREEN-API request timed out",
		}
	}

	if errors.Is(err, context.Canceled) {
		return MappedIntegrationError{
			HTTPStatus: http.StatusBadGateway,
			APICode:    "upstream_canceled",
			APIMessage: "Request to GREEN-API was canceled",
			LogLevel:   slog.LevelWarn,
			LogMsg:     "greenapi context canceled",
			LogAttrs:   []slog.Attr{slog.String("error", err.Error())},
		}
	}

	var inv *InvalidJSONResponseError
	if errors.As(err, &inv) {
		return MappedIntegrationError{
			HTTPStatus: http.StatusBadGateway,
			APICode:    "upstream_invalid_response",
			APIMessage: "GREEN-API returned an unexpected response",
			LogLevel:   slog.LevelWarn,
			LogMsg:     "greenapi invalid json on success",
			LogAttrs:   []slog.Attr{slog.String("error", err.Error())},
		}
	}

	var he *HTTPError
	if errors.As(err, &he) {
		return mapHTTPError(he)
	}

	return MappedIntegrationError{
		HTTPStatus: http.StatusBadGateway,
		APICode:    "upstream_error",
		APIMessage: "Failed to call GREEN-API",
		LogLevel:   slog.LevelWarn,
		LogMsg:     "greenapi network or internal error",
		LogAttrs:   []slog.Attr{slog.String("error", err.Error())},
	}
}

func mapRoundTripError(rt *RoundTripError) MappedIntegrationError {
	baseAttrs := []slog.Attr{
		slog.String("op", rt.Op),
		slog.String("kind", rt.Kind),
		slog.String("error", rt.Err.Error()),
	}
	switch rt.Kind {
	case RoundTripKindTimeout:
		return MappedIntegrationError{
			HTTPStatus: http.StatusGatewayTimeout,
			APICode:    "upstream_timeout",
			APIMessage: "GREEN-API request timed out",
			LogLevel:   slog.LevelWarn,
			LogMsg:     "greenapi timeout",
			LogAttrs:   baseAttrs,
		}
	case RoundTripKindCanceled:
		return MappedIntegrationError{
			HTTPStatus: http.StatusBadGateway,
			APICode:    "upstream_canceled",
			APIMessage: "Request to GREEN-API was canceled",
			LogLevel:   slog.LevelWarn,
			LogMsg:     "greenapi canceled",
			LogAttrs:   baseAttrs,
		}
	default:
		return MappedIntegrationError{
			HTTPStatus: http.StatusBadGateway,
			APICode:    "upstream_transport",
			APIMessage: "Failed to call GREEN-API",
			LogLevel:   slog.LevelWarn,
			LogMsg:     "greenapi transport",
			LogAttrs:   baseAttrs,
		}
	}
}

func mapHTTPError(httpErr *HTTPError) MappedIntegrationError {
	snippetMax := publicBodySnippetRunes
	switch httpErr.Status {
	case http.StatusUnauthorized, http.StatusForbidden:
		snippetMax = authSnippetRunes
	}

	details := map[string]any{
		"status":    httpErr.Status,
		"retryable": httpErr.Retryable(),
	}
	if snippetMax > 0 {
		snip := jsonfmt.SafeSnippet(httpErr.Body, snippetMax)
		if snip != "" {
			details["body_snippet"] = snip
		}
	}
	if httpErr.Status == http.StatusTooManyRequests && httpErr.Header != nil {
		if ra := strings.TrimSpace(httpErr.Header.Get("Retry-After")); ra != "" {
			details["retry_after"] = ra
		}
	}

	var status int
	var code, msg string
	switch httpErr.Status {
	case http.StatusUnauthorized, http.StatusForbidden:
		status = http.StatusBadGateway
		code = "upstream_auth_error"
		msg = "GREEN-API rejected credentials"
	case http.StatusTooManyRequests:
		status = http.StatusServiceUnavailable
		code = "upstream_rate_limited"
		msg = "GREEN-API rate limit exceeded"
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
		status = http.StatusBadGateway
		code = "upstream_client_error"
		msg = "GREEN-API rejected the request"
	default:
		if httpErr.Status >= 500 {
			status = http.StatusBadGateway
			code = "upstream_server_error"
			msg = "GREEN-API server error"
		} else {
			status = http.StatusBadGateway
			code = "upstream_http_error"
			msg = "GREEN-API returned non-success response"
		}
	}

	return MappedIntegrationError{
		HTTPStatus: status,
		APICode:    code,
		APIMessage: msg,
		Details:    details,
		LogLevel:   slog.LevelInfo,
		LogMsg:     "greenapi non-2xx mapped",
		LogAttrs: []slog.Attr{
			slog.Int("upstream_status", httpErr.Status),
			slog.String("code", code),
			slog.Bool("retryable", httpErr.Retryable()),
		},
	}
}
