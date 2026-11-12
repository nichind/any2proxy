package detect

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	plain := []byte("vless://uuid@example.com:443#one\n")
	encoded := base64.StdEncoding.EncodeToString(plain)

	kind, body := Decode([]byte(encoded))
	if kind != "plain" || string(body) != string(plain) {
		t.Fatalf("unexpected decode: %s %q", kind, body)
	}

	clash := []byte("proxies:\n  - name: one\n    type: ss\n")
	kind, body = Decode(clash)
	if kind != "clash" || !strings.HasPrefix(string(body), "proxies:") {
		t.Fatalf("unexpected clash decode: %s", kind)
	}
}
