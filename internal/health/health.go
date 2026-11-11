package health

import (
	"context"
	"net"
	"time"
)

func TCP(ctx context.Context, addr string, timeout time.Duration) (time.Duration, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	latency := time.Since(start)
	if err != nil {
		return latency, err
	}
	_ = conn.Close()
	return latency, nil
}
