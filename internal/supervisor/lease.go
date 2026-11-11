package supervisor

import (
	"fmt"
	"net"
)

type Lease struct {
	Port       uint16
	Generation uint32
}

func Acquire() (Lease, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Lease{}, fmt.Errorf("lease: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return Lease{}, fmt.Errorf("lease close: %w", err)
	}
	return Lease{Port: uint16(port), Generation: 1}, nil
}

func AcquireRetry(attempts int) (Lease, error) {
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		lease, err := Acquire()
		if err == nil {
			return lease, nil
		}
		last = err
	}
	return Lease{}, last
}

func (l Lease) Addr() string {
	return fmt.Sprintf("127.0.0.1:%d", l.Port)
}

func (l Lease) URL(scheme string) string {
	return fmt.Sprintf("%s://%s", scheme, l.Addr())
}
