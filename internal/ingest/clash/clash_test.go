package clash

import (
	"strings"
	"testing"
)

func TestParseClash(t *testing.T) {
	input := `
proxies:
  - name: one
    type: vless
    server: example.com
    port: 443
    uuid: uuid-1
    network: tcp
    tls: true
    servername: example.com
    flow: xtls-rprx-vision
  - name: two
    type: trojan
    server: example.net
    port: 8443
    password: hunter2
  - name: three
    type: ss
    server: example.org
    port: 8388
    cipher: aes-128-gcm
    password: hunter2
  - name: four
    type: hysteria2
    server: example.edu
    port: 443
    password: hunter2
  - name: skip
    type: wireguard
    server: example.io
    port: 51820
`
	profile, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(profile.Nodes))
	}
	if len(profile.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(profile.Warnings))
	}
	if profile.Nodes[0].Account != "uuid-1" || profile.Nodes[0].Extra["flow"] != "xtls-rprx-vision" {
		t.Fatalf("vless mapping is wrong: %#v", profile.Nodes[0])
	}
	if profile.Nodes[1].Secret != "hunter2" {
		t.Fatal("trojan password did not land in Secret")
	}
	if profile.Nodes[2].Account != "aes-128-gcm" || profile.Nodes[2].Secret != "hunter2" {
		t.Fatal("ss mapping is wrong")
	}
	if profile.Nodes[3].Protocol != "hysteria2" || profile.Nodes[3].Transport != "udp" {
		t.Fatal("hysteria2 mapping is wrong")
	}
}

func TestParseRejectsEmptyDocument(t *testing.T) {
	if _, err := Parse(strings.NewReader("proxies: []\n")); err == nil {
		t.Fatal("empty document passed")
	}
}
