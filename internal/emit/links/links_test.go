package links

import (
	"strings"
	"testing"

	"github.com/nichind/any2proxy/internal/ir"
)

func TestEmitLinks(t *testing.T) {
	profile := ir.Profile{Nodes: []ir.Node{
		{
			Tag: "vless", Protocol: "vless", Server: "example.com", Port: 443, Account: "uuid",
			Extra: map[string]string{"tls": "true", "servername": "example.com", "flow": "xtls-rprx-vision", "public-key": "key", "short-id": "ab"},
		},
		{Tag: "ss", Protocol: "ss", Server: "example.com", Port: 8388, Account: "aes-128-gcm", Secret: "pw"},
		{Tag: "hy2", Protocol: "hysteria2", Server: "example.com", Port: 443, Secret: "pw", Extra: map[string]string{"obfs": "salamander", "obfs-password": "obfs"}},
		{Tag: "any", Protocol: "anytls", Server: "example.com", Port: 443, Secret: "pw", Extra: map[string]string{"sni": "example.com"}},
		{Tag: "bad", Protocol: "wireguard", Server: "example.com", Port: 51820},
	}}
	got, dropped := Emit(profile)
	if len(got) != 4 || len(dropped) != 1 {
		t.Fatalf("unexpected result: %d links, %d dropped", len(got), len(dropped))
	}
	if !strings.HasPrefix(got[0], "vless://") || !strings.Contains(got[0], "security=reality") || !strings.Contains(got[0], "pbk=key") {
		t.Fatalf("unexpected vless link: %s", got[0])
	}
	if !strings.HasPrefix(got[1], "ss://") {
		t.Fatalf("unexpected ss link: %s", got[1])
	}
	if !strings.HasPrefix(got[2], "hysteria2://") || !strings.Contains(got[2], "obfs=salamander") {
		t.Fatalf("unexpected hysteria2 link: %s", got[2])
	}
	if !strings.HasPrefix(got[3], "anytls://") {
		t.Fatalf("unexpected anytls link: %s", got[3])
	}
}
