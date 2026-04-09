package greenapi

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"green-api-test/internal/metrics"
)

func TestMain(m *testing.M) {
	metrics.Register(prometheus.DefaultRegisterer)
	os.Exit(m.Run())
}
