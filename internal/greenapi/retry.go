package greenapi

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

const maxUpstreamAttempts = 4 // initial try + up to 3 retries

// retryDelay returns backoff duration before retry attempt retryIdx (0 = first retry after a failure).
func retryDelay(retryIdx int) time.Duration {
	const base = 50 * time.Millisecond
	const cap = 2 * time.Second

	d := base
	for i := 0; i < retryIdx; i++ {
		if d >= cap {
			d = cap
			break
		}
		d *= 2
	}
	if d > cap {
		d = cap
	}
	// Jitter in [0, d/5] to reduce thundering herds.
	var jitter time.Duration
	if d > 0 {
		jitter = time.Duration(rand.Int64N(int64(d/5 + 1)))
	}
	return d + jitter
}

func isRetryableTransportErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	// Per-attempt deadline or dial timeouts are often transient under load.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var rt *RoundTripError
	if errors.As(err, &rt) {
		switch rt.Kind {
		case RoundTripKindCanceled:
			return false
		case RoundTripKindTimeout, RoundTripKindTransport:
			return true
		case RoundTripKindRead, RoundTripKindMarshal:
			return false
		default:
			return false
		}
	}
	return true
}
