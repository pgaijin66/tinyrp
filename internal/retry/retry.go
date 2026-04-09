package retry

import (
	"math"
	"time"
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

var Default = Config{
	MaxAttempts: 3,
	BaseDelay:   10 * time.Millisecond,
	MaxDelay:    200 * time.Millisecond,
}

// Delay returns the backoff delay for the given attempt (0-indexed).
func (c Config) Delay(attempt int) time.Duration {
	d := time.Duration(float64(c.BaseDelay) * math.Pow(2, float64(attempt)))
	if d > c.MaxDelay {
		return c.MaxDelay
	}
	return d
}
