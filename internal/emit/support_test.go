package emit_test

import (
	"testing"

	"github.com/nichind/any2proxy/internal/emit/links"
	"github.com/nichind/any2proxy/internal/emit/mihomo"
	"github.com/nichind/any2proxy/internal/emit/singbox"
	"github.com/nichind/any2proxy/internal/emit/xray"
	"github.com/nichind/any2proxy/internal/ir"
)

func allNodes() ir.Profile {
	return ir.Profile{Nodes: []ir.Node{
		{Tag: "vless", Protocol: "vless", Server: "example.com", Port: 443, Account: "uuid", Extra: map[string]string{"tls": "true", "public-key": "key"}},
		{Tag: "vmess", Protocol: "vmess", Server: "example.com", Port: 443, Account: "uuid"},
		{Tag: "trojan", Protocol: "trojan", Server: "example.com", Port: 443, Secret: "pass"},
		{Tag: "ss", Protocol: "ss", Server: "example.com", Port: 8388, Account: "aes-128-gcm", Secret: "pass"},
		{Tag: "hysteria2", Protocol: "hysteria2", Server: "example.com", Port: 443, Secret: "pass"},
		{Tag: "tuic", Protocol: "tuic", Server: "example.com", Port: 443, Account: "uuid", Secret: "pass"},
		{Tag: "anytls", Protocol: "anytls", Server: "example.com", Port: 443, Secret: "pass"},
	}}
}

func TestAllProtocolsSupported(t *testing.T) {
	profile := allNodes()

	if _, report, err := singbox.Emit(profile, false); err != nil {
		t.Fatal(err)
	} else if len(report.Mapped) != 7 || len(report.Dropped) != 0 {
		t.Fatalf("sing-box: mapped=%v dropped=%v", report.Mapped, report.Dropped)
	}

	if _, report, err := mihomo.Emit(profile, mihomo.Options{Port: 1080}); err != nil {
		t.Fatal(err)
	} else if len(report.Mapped) != 7 || len(report.Dropped) != 0 {
		t.Fatalf("mihomo: mapped=%v dropped=%v", report.Mapped, report.Dropped)
	}

	if _, report, err := xray.Emit(profile, xray.Options{Port: 1080}); err != nil {
		t.Fatal(err)
	} else if len(report.Mapped) != 4 || len(report.Dropped) != 3 {
		t.Fatalf("xray: mapped=%v dropped=%v", report.Mapped, report.Dropped)
	}

	lines, dropped := links.Emit(profile)
	if len(lines) != 7 || len(dropped) != 0 {
		t.Fatalf("links: %d links, %d dropped", len(lines), len(dropped))
	}
}
