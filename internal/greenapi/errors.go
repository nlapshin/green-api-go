package greenapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	RoundTripKindTimeout   = "timeout"
	RoundTripKindCanceled  = "canceled"
	RoundTripKindTransport = "transport"
	RoundTripKindRead      = "read"
	RoundTripKindMarshal   = "marshal"
)

type RoundTripError struct {
	Op   string
	Kind string
	Err  error
}

func (e *RoundTripError) Error() string {
	if e == nil {
		return "greenapi: nil RoundTripError"
	}
	return fmt.Sprintf("greenapi.%s: %s: %v", e.Op, e.Kind, e.Err)
}

func (e *RoundTripError) Unwrap() error { return e.Err }

func classifyRoundTripFailure(op string, err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return &RoundTripError{Op: op, Kind: RoundTripKindTimeout, Err: err}
	case errors.Is(err, context.Canceled):
		return &RoundTripError{Op: op, Kind: RoundTripKindCanceled, Err: err}
	default:
		return &RoundTripError{Op: op, Kind: RoundTripKindTransport, Err: err}
	}
}

type HTTPError struct {
	Status     int
	StatusText string
	Header     http.Header
	Body       []byte
}

func (e *HTTPError) Error() string {
	if e == nil {
		return "green-api: nil HTTPError"
	}
	if len(e.Body) == 0 {
		return fmt.Sprintf("green-api http %d %s", e.Status, e.StatusText)
	}
	snip := publicBodySnippet(e.Body, 160)
	return fmt.Sprintf("green-api http %d %s: %s", e.Status, e.StatusText, snip)
}

func (e *HTTPError) Retryable() bool {
	if e == nil {
		return false
	}
	switch e.Status {
	case 408, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

type InvalidJSONResponseError struct {
	Reason string
}

func (e *InvalidJSONResponseError) Error() string {
	return "green-api: " + e.Reason
}

func publicBodySnippet(b []byte, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 160
	}
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	var sb strings.Builder
	n := 0
	for _, r := range s {
		if r < 0x20 && r != '\t' {
			r = ' '
		}
		sb.WriteRune(r)
		n++
		if n >= maxRunes {
			sb.WriteString("…")
			break
		}
	}
	return strings.TrimSpace(sb.String())
}
