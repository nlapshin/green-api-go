package greenapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"green-api-test/internal/domain"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustClient(t *testing.T, cfg Config) *Client {
	t.Helper()
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewClient_invalidTimeout(t *testing.T) {
	_, err := NewClient(Config{BaseURL: "https://api.example.com", Timeout: 0, Logger: testLogger()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewClient_invalidBaseURL(t *testing.T) {
	_, err := NewClient(Config{BaseURL: "ftp://x", Timeout: time.Second, Logger: testLogger()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_200ValidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/getSettings/") {
			t.Fatalf("path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := mustClient(t, Config{
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
		Logger:  testLogger(),
	})
	raw, err := c.GetSettings(context.Background(), domain.ConnectRequest{
		IDInstance:       "11001234",
		APITokenInstance: "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"ok":true}` {
		t.Fatalf("body: %s", raw)
	}
}

func TestClient_200NonJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := mustClient(t, Config{BaseURL: srv.URL, Timeout: 5 * time.Second, Logger: testLogger()})
	_, err := c.GetSettings(context.Background(), domain.ConnectRequest{
		IDInstance: "11001234", APITokenInstance: "t",
	})
	var inv *InvalidJSONResponseError
	if err == nil || !errors.As(err, &inv) {
		t.Fatalf("expected InvalidJSONResponseError, got %v", err)
	}
}

func TestClient_HTTPErrorRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "10")
		http.Error(w, "slow", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := mustClient(t, Config{BaseURL: srv.URL, Timeout: 5 * time.Second, Logger: testLogger()})
	_, err := c.GetSettings(context.Background(), domain.ConnectRequest{
		IDInstance: "11001234", APITokenInstance: "t",
	})
	var he *HTTPError
	if err == nil || !errors.As(err, &he) {
		t.Fatalf("expected HTTPError, got %v", err)
	}
	if he.Status != http.StatusTooManyRequests {
		t.Fatal(he.Status)
	}
	if he.Header.Get("Retry-After") != "10" {
		t.Fatal("missing retry-after")
	}
	if !he.Retryable() {
		t.Fatal("expected retryable")
	}
}

func TestClient_401NotRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := mustClient(t, Config{BaseURL: srv.URL, Timeout: 5 * time.Second, Logger: testLogger()})
	_, err := c.GetSettings(context.Background(), domain.ConnectRequest{
		IDInstance: "11001234", APITokenInstance: "t",
	})
	var he *HTTPError
	if err == nil || !errors.As(err, &he) {
		t.Fatal(err)
	}
	if he.Retryable() {
		t.Fatal("401 should not be retryable")
	}
}

func TestClient_200EmptyBodyOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, Config{BaseURL: srv.URL, Timeout: 5 * time.Second, Logger: testLogger()})
	raw, err := c.GetSettings(context.Background(), domain.ConnectRequest{
		IDInstance: "1", APITokenInstance: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 0 {
		t.Fatalf("expected empty body, got %q", raw)
	}
}
