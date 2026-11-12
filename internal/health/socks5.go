package health

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

func ThroughProxy(ctx context.Context, proxyURL, targetURL string, timeout time.Duration) (int, time.Duration, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return 0, 0, fmt.Errorf("proxy: %w", err)
	}
	if proxy.Scheme != "socks5" && proxy.Scheme != "socks5h" {
		return 0, 0, fmt.Errorf("proxy: unsupported scheme %q", proxy.Scheme)
	}
	target, err := url.Parse(targetURL)
	if err != nil {
		return 0, 0, fmt.Errorf("target: %w", err)
	}
	port := target.Port()
	if port == "" {
		if target.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	dialer := func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialSOCKS5(ctx, proxy.Host, target.Hostname(), port, timeout)
	}
	transport := &http.Transport{
		DialContext:       dialer,
		ForceAttemptHTTP2: false,
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
	}
	client := &http.Client{Transport: transport, Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return 0, 0, err
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, time.Since(start), err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
	_ = resp.Body.Close()
	return resp.StatusCode, time.Since(start), nil
}

func dialSOCKS5(ctx context.Context, proxyAddr, host, port string, timeout time.Duration) (net.Conn, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)

	if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if reply[0] != 5 || reply[1] != 0 {
		_ = conn.Close()
		return nil, fmt.Errorf("socks5: auth rejected")
	}

	request := []byte{5, 1, 0, 3, byte(len(host))}
	request = append(request, host...)
	request = binary.BigEndian.AppendUint16(request, parsePort(port))
	if _, err := conn.Write(request); err != nil {
		_ = conn.Close()
		return nil, err
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if header[1] != 0 {
		_ = conn.Close()
		return nil, fmt.Errorf("socks5: connect failed (%d)", header[1])
	}
	if err := skipAddress(conn, header[3]); err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

func skipAddress(conn net.Conn, atyp byte) error {
	switch atyp {
	case 1:
		_, err := io.ReadFull(conn, make([]byte, 4+2))
		return err
	case 3:
		length := make([]byte, 1)
		if _, err := io.ReadFull(conn, length); err != nil {
			return err
		}
		_, err := io.ReadFull(conn, make([]byte, int(length[0])+2))
		return err
	case 4:
		_, err := io.ReadFull(conn, make([]byte, 16+2))
		return err
	default:
		return fmt.Errorf("socks5: unknown address type %d", atyp)
	}
}

func parsePort(port string) uint16 {
	var value uint16
	for _, ch := range port {
		if ch < '0' || ch > '9' {
			return 0
		}
		value = value*10 + uint16(ch-'0')
	}
	return value
}
