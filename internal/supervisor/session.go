package supervisor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Command      []string
	Env          []string
	Dir          string
	Lease        *Lease
	ReadyTimeout time.Duration
	StopTimeout  time.Duration
}

type Session struct {
	lease       Lease
	cmd         *exec.Cmd
	out         *bytes.Buffer
	stopTimeout time.Duration

	mu      sync.Mutex
	stopped bool
	done    chan struct{}
	exitErr error
}

func Start(ctx context.Context, cfg Config) (*Session, error) {
	if len(cfg.Command) == 0 {
		return nil, fmt.Errorf("supervisor: empty command")
	}
	if cfg.ReadyTimeout <= 0 {
		cfg.ReadyTimeout = 5 * time.Second
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 3 * time.Second
	}

	var (
		lease Lease
		err   error
	)
	if cfg.Lease != nil && cfg.Lease.Port != 0 {
		lease = *cfg.Lease
	} else {
		lease, err = AcquireRetry(3)
		if err != nil {
			return nil, err
		}
	}

	args := make([]string, len(cfg.Command)-1)
	for i, arg := range cfg.Command[1:] {
		arg = strings.ReplaceAll(arg, "{addr}", lease.Addr())
		args[i] = strings.ReplaceAll(arg, "{port}", fmt.Sprint(lease.Port))
	}

	cmd := exec.CommandContext(ctx, cfg.Command[0], args...)
	cmd.Env = append(os.Environ(), cfg.Env...)
	if cfg.Dir != "" {
		cmd.Dir = cfg.Dir
	}
	out := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("supervisor start: %w", err)
	}

	session := &Session{
		lease:       lease,
		cmd:         cmd,
		out:         out,
		stopTimeout: cfg.StopTimeout,
		done:        make(chan struct{}),
	}
	go func() {
		session.exitErr = cmd.Wait()
		close(session.done)
	}()

	if err := session.waitReady(cfg.ReadyTimeout); err != nil {
		_ = session.kill()
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
	return session, nil
}

func (s *Session) Endpoint() string {
	return s.lease.URL("socks5")
}

func (s *Session) Lease() Lease {
	return s.lease
}

func (s *Session) Output() string {
	return s.out.String()
}

func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) Err() error {
	<-s.done
	return s.exitErr
}

func (s *Session) Stop() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	s.mu.Unlock()
	return s.kill()
}

func (s *Session) kill() error {
	if s.cmd.Process == nil {
		return nil
	}
	err := s.cmd.Process.Kill()
	select {
	case <-s.done:
	case <-time.After(s.stopTimeout):
	}
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("supervisor stop: %w", err)
	}
	return nil
}

func (s *Session) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", s.lease.Addr(), 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case <-s.done:
			return fmt.Errorf("core exited before ready: %v", s.exitErr)
		default:
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("core did not become ready in %s", timeout)
}
