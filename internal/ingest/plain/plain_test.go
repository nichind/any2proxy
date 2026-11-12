package plain

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseAllProtocolTypes(t *testing.T) {
	vmessPayload, err := json.Marshal(map[string]any{
		"v": "2", "ps": "vmess", "add": "example.com", "port": "443",
		"id": "uuid", "net": "ws", "tls": "tls", "sni": "example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	ssUser := base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:pass"))
	input := strings.Join([]string{
		"vless://uuid@example.com:443?security=reality&sni=example.com&pbk=key&sid=ab&fp=chrome&flow=xtls-rprx-vision&type=tcp#vless",
		"vmess://" + base64.StdEncoding.EncodeToString(vmessPayload),
		"trojan://pass@example.com:443?sni=example.com#trojan",
		"ss://" + ssUser + "@example.com:8388#ss",
		"hysteria2://pass@example.com:443?sni=example.com&obfs=salamander&obfs-password=obfs#hy2",
		"tuic://uuid:pass@example.com:443?sni=example.com#tuic",
		"anytls://pass@example.com:443?sni=example.com&fp=chrome#anytls",
	}, "\n")

	profile, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Nodes) != 7 {
		t.Fatalf("expected 7 nodes, got %d: %#v", len(profile.Nodes), profile.Warnings)
	}
	byTag := map[string]int{}
	for i, node := range profile.Nodes {
		byTag[node.Tag] = i
	}
	if profile.Nodes[byTag["vless"]].Extra["public-key"] != "key" {
		t.Fatal("vless reality key lost")
	}
	if profile.Nodes[byTag["vmess"]].Extra["network"] != "ws" {
		t.Fatal("vmess network lost")
	}
	if profile.Nodes[byTag["ss"]].Account != "aes-128-gcm" || profile.Nodes[byTag["ss"]].Secret != "pass" {
		t.Fatal("ss userinfo not decoded")
	}
	if profile.Nodes[byTag["hy2"]].Extra["obfs"] != "salamander" {
		t.Fatal("hysteria2 obfs lost")
	}
	if profile.Nodes[byTag["anytls"]].Secret != "pass" {
		t.Fatal("anytls password lost")
	}
}
