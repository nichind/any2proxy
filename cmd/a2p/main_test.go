package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRunVersion(t *testing.T) {
	if code := run([]string{"version"}); code != 0 {
		t.Fatalf("version exit code %d", code)
	}
}

func TestRunUnknown(t *testing.T) {
	if code := run([]string{"nope"}); code != 2 {
		t.Fatalf("unknown exit code %d", code)
	}
}

func TestParseMissingArgument(t *testing.T) {
	if code := run([]string{"parse"}); code != 2 {
		t.Fatalf("missing argument exit code %d", code)
	}
}

func TestParseFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := parseCommand([]string{f.Name()}); code != 0 {
		t.Fatalf("parse exit code %d", code)
	}
}

func TestCompilePlain(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := compileCommand([]string{f.Name()}); code != 0 {
		t.Fatalf("compile exit code %d", code)
	}
}

func TestCompileUnsupportedTarget(t *testing.T) {
	if code := compileCommand([]string{"--to", "xray", "-"}); code != 4 {
		t.Fatalf("unsupported target exit code %d", code)
	}
}

func TestCompileMihomo(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := compileCommand([]string{"--to", "mihomo", f.Name()}); code != 0 {
		t.Fatalf("mihomo compile exit code %d", code)
	}
}

func TestRunDryRunMihomo(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := runCommand([]string{"--to", "mihomo", "--dry-run", f.Name()}); code != 0 {
		t.Fatalf("mihomo dry run exit code %d", code)
	}
}

func TestCompileXray(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := compileCommand([]string{"--to", "xray", f.Name()}); code != 0 {
		t.Fatalf("xray compile exit code %d", code)
	}
}

func TestRunDryRunXray(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := runCommand([]string{"--to", "xray", "--dry-run", f.Name()}); code != 0 {
		t.Fatalf("xray dry run exit code %d", code)
	}
}

func TestRunDryRunPool(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("trojan" + "://pw@example.net:443#two\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := runCommand([]string{"--mode", "pool", "--dry-run", f.Name()}); code != 0 {
		t.Fatalf("pool dry run exit code %d", code)
	}
}

func TestParseURL(t *testing.T) {
	t.Setenv("A2P_HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("vless" + "://u@example.com:443#one\n"))
	}))
	defer srv.Close()

	if code := parseCommand([]string{"--allow-private", srv.URL}); code != 0 {
		t.Fatalf("url parse exit code %d", code)
	}
}

func TestSubLifecycle(t *testing.T) {
	t.Setenv("A2P_HOME", t.TempDir())
	if code := subCommand([]string{"add", "main", "https://example.com/sub"}); code != 0 {
		t.Fatalf("sub add exit code %d", code)
	}
	if code := subCommand([]string{"list"}); code != 0 {
		t.Fatalf("sub list exit code %d", code)
	}
	if code := subCommand([]string{"remove", "main"}); code != 0 {
		t.Fatalf("sub remove exit code %d", code)
	}
	if code := subCommand([]string{"list"}); code != 0 {
		t.Fatalf("sub list after remove exit code %d", code)
	}
}

func TestSubRejectsBadInput(t *testing.T) {
	t.Setenv("A2P_HOME", t.TempDir())
	if code := subCommand([]string{"add", "main"}); code != 2 {
		t.Fatalf("missing url exit code %d", code)
	}
	if code := subCommand([]string{"add", "bad name", "https://example.com/sub"}); code != 3 {
		t.Fatalf("bad name exit code %d", code)
	}
}

func TestCheckEndpoint(t *testing.T) {
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
	if code := checkCommand([]string{listener.Addr().String()}); code != 0 {
		t.Fatalf("check exit code %d", code)
	}
}

func TestCheckEndpointFails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	if code := checkCommand([]string{addr}); code != 1 {
		t.Fatalf("check failure exit code %d", code)
	}
}

func TestCoresLifecycle(t *testing.T) {
	t.Setenv("A2P_HOME", t.TempDir())
	source := filepath.Join(t.TempDir(), "sing-box")
	if err := os.WriteFile(source, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code := coresCommand([]string{"import", "sing-box", source}); code != 0 {
		t.Fatalf("cores import exit code %d", code)
	}
	if code := coresCommand([]string{"list"}); code != 0 {
		t.Fatalf("cores list exit code %d", code)
	}
	if code := coresCommand([]string{"verify", "sing-box"}); code != 0 {
		t.Fatalf("cores verify exit code %d", code)
	}
	if code := coresCommand([]string{"remove", "sing-box"}); code != 0 {
		t.Fatalf("cores remove exit code %d", code)
	}
}

func TestCompileClashStrict(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("proxies:\n  - name: one\n    type: vless\n    server: example.com\n    port: 443\n    uuid: u\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := compileCommand([]string{"--strict", f.Name()}); code != 0 {
		t.Fatalf("clash compile exit code %d", code)
	}
}

func TestRunDryRun(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := runCommand([]string{"--dry-run", f.Name()}); code != 0 {
		t.Fatalf("dry run exit code %d", code)
	}
}

func TestRunMissingBinary(t *testing.T) {
	t.Setenv("A2P_CORE", "")
	t.Setenv("A2P_HOME", t.TempDir())
	f, err := os.CreateTemp(t.TempDir(), "links")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("vless" + "://u@example.com:443#one\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if code := runCommand([]string{f.Name()}); code != 5 {
		t.Fatalf("missing binary exit code %d", code)
	}
}
