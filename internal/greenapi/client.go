package greenapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"green-api-test/internal/domain"
)

type Config struct {
	BaseURL              string
	Timeout              time.Duration
	Logger               *slog.Logger
	RequestIDFromContext func(context.Context) string
}

type Client struct {
	base                 *url.URL
	http                 *http.Client
	timeout              time.Duration
	logger               *slog.Logger
	requestIDFromContext func(context.Context) string
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("greenapi config: Timeout must be positive")
	}
	base, err := parseGreenAPIBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("greenapi config: BaseURL: %w", err)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 16
	rt := newLoggingTransport(transport, logger)
	return &Client{
		base:                 base,
		timeout:              cfg.Timeout,
		logger:               logger,
		requestIDFromContext: cfg.RequestIDFromContext,
		http: &http.Client{
			Transport: rt,
		},
	}, nil
}

func (c *Client) GetSettings(ctx context.Context, cred domain.ConnectRequest) ([]byte, error) {
	u, err := instanceMethodURL(c.base, cred.IDInstance, "getSettings", cred.APITokenInstance)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, "getSettings", http.MethodGet, u, nil)
}

func (c *Client) GetStateInstance(ctx context.Context, cred domain.ConnectRequest) ([]byte, error) {
	u, err := instanceMethodURL(c.base, cred.IDInstance, "getStateInstance", cred.APITokenInstance)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, "getStateInstance", http.MethodGet, u, nil)
}

func (c *Client) SendMessage(ctx context.Context, cred domain.ConnectRequest, msg domain.OutboundTextMessage) ([]byte, error) {
	u, err := instanceMethodURL(c.base, cred.IDInstance, "sendMessage", cred.APITokenInstance)
	if err != nil {
		return nil, err
	}
	body := struct {
		ChatID  string `json:"chatId"`
		Message string `json:"message"`
	}{ChatID: msg.ChatID, Message: msg.Message}
	return c.do(ctx, "sendMessage", http.MethodPost, u, body)
}

func (c *Client) SendFileByURL(ctx context.Context, cred domain.ConnectRequest, file domain.OutboundFileMessage) ([]byte, error) {
	u, err := instanceMethodURL(c.base, cred.IDInstance, "sendFileByUrl", cred.APITokenInstance)
	if err != nil {
		return nil, err
	}
	body := struct {
		ChatID   string `json:"chatId"`
		URLFile  string `json:"urlFile"`
		FileName string `json:"fileName"`
		Caption  string `json:"caption,omitempty"`
	}{
		ChatID:   file.ChatID,
		URLFile:  file.URLFile,
		FileName: file.FileName,
		Caption:  file.Caption,
	}
	return c.do(ctx, "sendFileByUrl", http.MethodPost, u, body)
}

func (c *Client) do(ctx context.Context, op, method, requestURL string, body any) ([]byte, error) {
	ctx = withOperation(ctx, op)
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, &RoundTripError{Op: op, Kind: RoundTripKindMarshal, Err: fmt.Errorf("marshal request: %w", err)}
		}
		payload = b
	}

	for attempt := 0; attempt < maxUpstreamAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, classifyRoundTripFailure(op, ctx.Err())
			case <-time.After(retryDelay(attempt - 1)):
			}
		}

		var rdr io.Reader
		if len(payload) > 0 {
			rdr = bytes.NewReader(payload)
		}

		req, err := http.NewRequestWithContext(ctx, method, requestURL, rdr)
		if err != nil {
			return nil, classifyRoundTripFailure(op, fmt.Errorf("new request: %w", err))
		}
		if len(payload) > 0 {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
		if c.requestIDFromContext != nil {
			if rid := strings.TrimSpace(c.requestIDFromContext(ctx)); rid != "" {
				req.Header.Set("X-Request-Id", rid)
			}
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				err = context.DeadlineExceeded
			}
			classified := classifyRoundTripFailure(op, err)
			if isRetryableTransportErr(classified) && attempt < maxUpstreamAttempts-1 {
				continue
			}
			return nil, classified
		}

		data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil, &RoundTripError{Op: op, Kind: RoundTripKindRead, Err: fmt.Errorf("read response: %w", err)}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			data = bytes.TrimSpace(data)
			if len(data) > 0 && !json.Valid(data) {
				return nil, &InvalidJSONResponseError{Reason: "upstream returned non-JSON body on success status"}
			}
			return data, nil
		}

		he := &HTTPError{
			Status:     resp.StatusCode,
			StatusText: resp.Status,
			Header:     resp.Header.Clone(),
			Body:       data,
		}
		if he.Retryable() && attempt < maxUpstreamAttempts-1 {
			continue
		}
		return nil, he
	}
	return nil, classifyRoundTripFailure(op, errors.New("upstream: retry loop exhausted"))
}
