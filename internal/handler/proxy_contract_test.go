package handler_test

import (
	"testing"

	"green-api-test/internal/greenapi"
	"green-api-test/internal/handler"
)

func TestGreenAPIClientImplementsProxy(t *testing.T) {
	var _ handler.Proxy = (*greenapi.Client)(nil)
}
