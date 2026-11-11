package singbox

import (
	"encoding/json"
	"testing"

	"github.com/nichind/any2proxy/internal/ir"
)

type decoded struct {
	Outbounds []struct {
		Type     string `json:"type"`
		Tag      string `json:"tag"`
		UUID     string `json:"uuid"`
		Password string `json:"password"`
		Method   string `json:"method"`
		Flow     string `json:"flow"`
		TLS      *struct {
			Enabled    bool   `json:"enabled"`
			ServerName string `json:"server_name"`
		} `json:"tls"`
	} `json:"outbounds"`
}

func sample() ir.Profile {
	return ir.Profile{Nodes: []ir.Node{
		{
			Tag: "vless", Protocol: "vless", Server: "example.com", Port: 443,
			Account: "uuid-1", Transport: ir.TransportTCP,
			Extra: map[string]string{"flow": "xtls-rprx-vision", "tls": "true", "sni": "example.com"},
		},
		{Tag: "trojan", Protocol: "trojan", Server: "example.net", Port: 443, Secret: "pw"},
		{Tag: "ss", Protocol: "ss", Server: "example.org", Port: 8388, Account: "aes-128-gcm", Secret: "pw"},
		{Tag: "unknown", Protocol: "wireguard", Server: "example.io", Port: 51820},
	}}
}

func TestEmitMapsSupportedNodes(t *testing.T) {
	data, report, err := Emit(sample(), false)
	if err != nil {
		t.Fatal(err)
	}
	var got decoded
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Outbounds) != 3 {
		t.Fatalf("expected 3 outbounds, got %d", len(got.Outbounds))
	}
	if got.Outbounds[0].Type != "vless" || got.Outbounds[0].Flow != "xtls-rprx-vision" {
		t.Fatalf("vless mapping is wrong: %#v", got.Outbounds[0])
	}
	if got.Outbounds[0].TLS == nil || got.Outbounds[0].TLS.ServerName != "example.com" {
		t.Fatal("vless tls mapping is wrong")
	}
	if got.Outbounds[1].Password != "pw" {
		t.Fatal("trojan password is missing")
	}
	if got.Outbounds[2].Method != "aes-128-gcm" {
		t.Fatal("ss method is missing")
	}
	if len(report.Dropped) != 1 || report.Dropped[0] != "unknown" {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestEmitStrictRejectsDropped(t *testing.T) {
	if _, _, err := Emit(sample(), true); err == nil {
		t.Fatal("strict mode accepted an unsupported node")
	}
}

func TestEmitRejectsEmptyProfile(t *testing.T) {
	if _, _, err := Emit(ir.Profile{}, false); err == nil {
		t.Fatal("empty profile passed")
	}
}

func TestEmitWithSocksInbound(t *testing.T) {
	data, _, err := EmitWithOptions(sample(), Options{SocksPort: 1080})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Inbounds []struct {
			Type       string `json:"type"`
			Listen     string `json:"listen"`
			ListenPort uint16 `json:"listen_port"`
		} `json:"inbounds"`
		Route struct {
			Final string `json:"final"`
		} `json:"route"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Inbounds) != 1 || got.Inbounds[0].Type != "socks" || got.Inbounds[0].ListenPort != 1080 {
		t.Fatalf("unexpected inbound: %#v", got.Inbounds)
	}
	if got.Route.Final != "vless" {
		t.Fatalf("unexpected route: %#v", got.Route)
	}
}
