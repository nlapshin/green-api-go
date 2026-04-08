package httpx

import (
	"net/http"
	"strings"

	"green-api-test/internal/domain"
)

const (
	HeaderInstanceID = "X-Instance-Id"
	HeaderAPIToken   = "X-Api-Token"
)

func ConnectFromHeaders(r *http.Request) domain.ConnectRequest {
	var c domain.ConnectRequest
	if v := strings.TrimSpace(r.Header.Get(HeaderInstanceID)); v != "" {
		c.IDInstance = v
	}
	if v := strings.TrimSpace(r.Header.Get(HeaderAPIToken)); v != "" {
		c.APITokenInstance = v
	}
	return c
}

func MergeConnectHeaders(r *http.Request, c *domain.ConnectRequest) {
	if v := strings.TrimSpace(r.Header.Get(HeaderInstanceID)); v != "" {
		c.IDInstance = v
	}
	if v := strings.TrimSpace(r.Header.Get(HeaderAPIToken)); v != "" {
		c.APITokenInstance = v
	}
}
