package supervisor

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLeaseURL(t *testing.T) {
	lease := Lease{Port: 1080, Generation: 1}
	if got := lease.URL("socks5"); got != "socks5://127.0.0.1:1080" {
		t.Fatalf("unexpected URL: %s", got)
	}
}

func TestAcquire(t *testing.T) {
	lease, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if lease.Port == 0 {
		t.Fatal("lease has no port")
	}
}

func TestStartAndStop(t *testing.T) {
	core := buildFakeCore(t)
	session, err := Start(context.Background(), Config{
		Command:      []string{core, "-addr", "{addr}"},
		ReadyTimeout: 3 * time.Second,
		StopTimeout:  2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Endpoint() == "" {
		t.Fatal("empty endpoint")
	}
	conn, err := net.DialTimeout("tcp", session.Lease().Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	if err := session.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := session.Stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestCrashBeforeReady(t *testing.T) {
	core := buildFakeCore(t)
	_, err := Start(context.Background(), Config{
		Command:      []string{core, "-mode", "crash"},
		ReadyTimeout: 2 * time.Second,
	})
	if err == nil {
		t.Fatal("crashing core started")
	}
}

func TestHangTimesOut(t *testing.T) {
	core := buildFakeCore(t)
	start := time.Now()
	_, err := Start(context.Background(), Config{
		Command:      []string{core, "-mode", "hang"},
		ReadyTimeout: 300 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("hanging core started")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout took too long")
	}
}

func buildFakeCore(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(strings.TrimSpace(string(out)))
	name := "fakecore"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", binary, "./tests/fakecore")
	cmd.Dir = root
	cmd.Env = os.Environ()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fakecore: %v\n%s", err, output)
	}
	return binary
}
