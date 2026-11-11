package xray

import (
	"encoding/json"
	"testing"

	"github.com/nichind/any2proxy/internal/ir"
)

type decoded struct {
	Inbounds []struct {
		Port     uint16 `json:"port"`
		Protocol string `json:"protocol"`
	} `json:"inbounds"`
	Outbounds []struct {
		Tag      string `json:"tag"`
		Protocol string `json:"protocol"`
		Settings any    `json:"settings"`
		Stream   *struct {
			Network  string `json:"network"`
			Security string `json:"security"`
			TLS      *struct {
				ServerName string `json:"serverName"`
			} `json:"tlsSettings"`
		} `json:"streamSettings"`
	} `json:"outbounds"`
	Routing struct {
		Rules []struct {
			OutboundTag string `json:"outboundTag"`
		} `json:"rules"`
	} `json:"routing"`
}

func sample() ir.Profile {
	return ir.Profile{Nodes: []ir.Node{
		{
			Tag: "vless", Protocol: "vless", Server: "example.com", Port: 443,
			Account: "uuid-1", Transport: ir.TransportTCP,
			Extra: map[string]string{"tls": "true", "servername": "example.com", "flow": "xtls-rprx-vision"},
		},
		{Tag: "trojan", Protocol: "trojan", Server: "example.net", Port: 443, Secret: "pw", Extra: map[string]string{"sni": "example.net"}},
		{Tag: "ss", Protocol: "ss", Server: "example.org", Port: 8388, Account: "aes-128-gcm", Secret: "pw"},
		{Tag: "hysteria", Protocol: "hysteria2", Server: "example.edu", Port: 443, Secret: "pw"},
	}}
}

func TestEmitMapsSupportedNodes(t *testing.T) {
	data, report, err := Emit(sample(), Options{Port: 1080})
	if err != nil {
		t.Fatal(err)
	}
	var got decoded
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Inbounds) != 1 || got.Inbounds[0].Port != 1080 || got.Inbounds[0].Protocol != "socks" {
		t.Fatalf("unexpected inbound: %#v", got.Inbounds)
	}
	if len(got.Outbounds) != 3 {
		t.Fatalf("expected 3 outbounds, got %d", len(got.Outbounds))
	}
	if got.Outbounds[0].Protocol != "vless" || got.Outbounds[0].Tag != "proxy" {
		t.Fatalf("vless mapping is wrong: %#v", got.Outbounds[0])
	}
	if got.Outbounds[0].Stream == nil || got.Outbounds[0].Stream.Security != "tls" {
		t.Fatal("vless tls mapping is wrong")
	}
	if got.Outbounds[1].Protocol != "trojan" {
		t.Fatalf("trojan mapping is wrong: %#v", got.Outbounds[1])
	}
	if got.Outbounds[2].Protocol != "shadowsocks" || got.Outbounds[2].Stream != nil {
		t.Fatalf("ss mapping is wrong: %#v", got.Outbounds[2])
	}
	if len(got.Routing.Rules) != 1 || got.Routing.Rules[0].OutboundTag != "proxy" {
		t.Fatalf("routing is wrong: %#v", got.Routing)
	}
	if len(report.Dropped) != 1 || report.Dropped[0] != "hysteria" {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestEmitStrictRejectsDropped(t *testing.T) {
	if _, _, err := Emit(sample(), Options{Strict: true}); err == nil {
		t.Fatal("strict mode accepted an unsupported node")
	}
}

func TestEmitRejectsEmptyProfile(t *testing.T) {
	if _, _, err := Emit(ir.Profile{}, Options{}); err == nil {
		t.Fatal("empty profile passed")
	}
}
