package greenapi

import (
	"fmt"
	"net/url"
	"strings"
)

func parseGreenAPIBaseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("base URL is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("base URL must use http or https")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("base URL must include a host")
	}
	return u, nil
}
