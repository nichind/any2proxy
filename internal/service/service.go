package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nichind/any2proxy/internal/emit/mihomo"
	"github.com/nichind/any2proxy/internal/emit/singbox"
	"github.com/nichind/any2proxy/internal/emit/xray"
	"github.com/nichind/any2proxy/internal/ingest/clash"
	"github.com/nichind/any2proxy/internal/ingest/plain"
	"github.com/nichind/any2proxy/internal/ir"
	"github.com/nichind/any2proxy/internal/supervisor"
	"github.com/nichind/any2proxy/internal/version"
)

type Service struct {
	mu       sync.Mutex
	sessions map[string]*managedSession
}

type managedSession struct {
	id       string
	run      *supervisor.Session
	tempPath string
}

type CompileResult struct {
	Config string              `json:"config"`
	Report ir.CapabilityReport `json:"report"`
}

type RunResult struct {
	SessionID string `json:"session_id"`
	Endpoint  string `json:"endpoint"`
}

func New() *Service {
	return &Service{sessions: make(map[string]*managedSession)}
}

func (s *Service) Capabilities() map[string]any {
	return map[string]any{
		"version": version.Version,
		"targets": []string{"sing-box", "mihomo", "xray"},
		"formats": []string{"plain", "clash"},
	}
}

func (s *Service) Parse(body, format string) (ir.Profile, error) {
	reader := strings.NewReader(body)
	switch format {
	case "", "plain":
		return plain.Parse(reader)
	case "clash":
		return clash.Parse(reader)
	default:
		return ir.Profile{}, fmt.Errorf("unsupported format %q", format)
	}
}

func (s *Service) Compile(body, format, target string, port uint16, strict bool) (CompileResult, error) {
	profile, err := s.Parse(body, format)
	if err != nil {
		return CompileResult{}, err
	}
	data, report, err := emit(target, profile, port, strict)
	if err != nil {
		return CompileResult{}, err
	}
	return CompileResult{Config: string(data), Report: report}, nil
}

func (s *Service) Run(ctx context.Context, body, format, target, binary string, port uint16, strict bool) (RunResult, error) {
	if binary == "" {
		return RunResult{}, fmt.Errorf("binary is required")
	}
	profile, err := s.Parse(body, format)
	if err != nil {
		return RunResult{}, err
	}

	lease, err := supervisor.AcquireRetry(3)
	if err != nil {
		return RunResult{}, err
	}
	if port != 0 {
		lease.Port = port
	}
	data, _, err := emit(target, profile, lease.Port, strict)
	if err != nil {
		return RunResult{}, err
	}
	path, err := writeTemp(data, target)
	if err != nil {
		return RunResult{}, err
	}

	session, err := supervisor.Start(ctx, supervisor.Config{
		Command:      coreCommand(binary, target, path),
		Lease:        &lease,
		ReadyTimeout: 10 * time.Second,
		StopTimeout:  5 * time.Second,
	})
	if err != nil {
		_ = os.Remove(path)
		return RunResult{}, err
	}

	id := newID()
	s.mu.Lock()
	s.sessions[id] = &managedSession{id: id, run: session, tempPath: path}
	s.mu.Unlock()
	return RunResult{SessionID: id, Endpoint: session.Endpoint()}, nil
}

func (s *Service) Stop(id string) error {
	s.mu.Lock()
	session, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	err := session.run.Stop()
	_ = os.Remove(session.tempPath)
	return err
}

func (s *Service) Close() {
	s.mu.Lock()
	sessions := make([]*managedSession, 0, len(s.sessions))
	for _, item := range s.sessions {
		sessions = append(sessions, item)
	}
	s.sessions = make(map[string]*managedSession)
	s.mu.Unlock()
	for _, item := range sessions {
		_ = item.run.Stop()
		_ = os.Remove(item.tempPath)
	}
}

func emit(target string, profile ir.Profile, port uint16, strict bool) ([]byte, ir.CapabilityReport, error) {
	switch target {
	case "", "sing-box":
		return singbox.EmitWithOptions(profile, singbox.Options{Strict: strict, SocksPort: port})
	case "mihomo":
		return mihomo.Emit(profile, mihomo.Options{Port: port, Strict: strict})
	case "xray":
		return xray.Emit(profile, xray.Options{Port: port, Strict: strict})
	default:
		return nil, ir.CapabilityReport{}, fmt.Errorf("unsupported target %q", target)
	}
}

func writeTemp(data []byte, target string) (string, error) {
	pattern := "any2proxy-*.json"
	if target == "mihomo" {
		pattern = "any2proxy-*.yaml"
	}
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func coreCommand(binary, target, path string) []string {
	if target == "mihomo" {
		return []string{binary, "-f", path}
	}
	return []string{binary, "run", "-c", path}
}

func newID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}
