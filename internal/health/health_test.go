package health

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	if _, err := TCP(context.Background(), listener.Addr().String(), time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestTCPFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	if _, err := TCP(context.Background(), addr, 200*time.Millisecond); err == nil {
		t.Fatal("closed port passed")
	}
}
