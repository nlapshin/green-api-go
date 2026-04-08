package greenapi

import (
	"fmt"
	"net/url"
)

func instanceMethodURL(base *url.URL, idInstance, method, apiToken string) (string, error) {
	if base == nil {
		return "", fmt.Errorf("greenapi: nil base URL")
	}
	segInstance := "waInstance" + idInstance
	u, err := url.JoinPath(base.String(), segInstance, method, apiToken)
	if err != nil {
		return "", fmt.Errorf("greenapi: join path: %w", err)
	}
	return u, nil
}
