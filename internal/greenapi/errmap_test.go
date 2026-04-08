package greenapi

import (
	"context"
	"net/http"
	"testing"
)

func TestMapIntegrationError_429(t *testing.T) {
	m := MapIntegrationError(&HTTPError{
		Status: http.StatusTooManyRequests,
		Header: http.Header{"Retry-After": []string{"120"}},
		Body:   []byte(`{"reason":"slow down"}`),
	})
	if m.HTTPStatus != http.StatusServiceUnavailable || m.APICode != "upstream_rate_limited" {
		t.Fatalf("unexpected mapping: %+v", m)
	}
	details, _ := m.Details.(map[string]any)
	if details["retry_after"] != "120" {
		t.Fatalf("retry_after: %v", details["retry_after"])
	}
}

func TestMapIntegrationError_Canceled(t *testing.T) {
	m := MapIntegrationError(context.Canceled)
	if m.HTTPStatus != http.StatusBadGateway || m.APICode != "upstream_canceled" {
		t.Fatalf("unexpected: %+v", m)
	}
	if m.LogMsg == "" {
		t.Fatal("expected server log hint")
	}
}
