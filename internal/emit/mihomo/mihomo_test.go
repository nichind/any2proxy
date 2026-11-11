package mihomo

import (
	"testing"

	"github.com/nichind/any2proxy/internal/ir"
	"gopkg.in/yaml.v3"
)

type decoded struct {
	SocksPort uint16 `yaml:"socks-port"`
	AllowLAN  bool   `yaml:"allow-lan"`
	Mode      string `yaml:"mode"`
	Proxies   []struct {
		Name       string `yaml:"name"`
		Type       string `yaml:"type"`
		Server     string `yaml:"server"`
		Port       uint16 `yaml:"port"`
		UUID       string `yaml:"uuid"`
		Password   string `yaml:"password"`
		Cipher     string `yaml:"cipher"`
		Network    string `yaml:"network"`
		TLS        bool   `yaml:"tls"`
		Servername string `yaml:"servername"`
		SNI        string `yaml:"sni"`
		Flow       string `yaml:"flow"`
	} `yaml:"proxies"`
	ProxyGroups []struct {
		Name    string   `yaml:"name"`
		Type    string   `yaml:"type"`
		Proxies []string `yaml:"proxies"`
	} `yaml:"proxy-groups"`
	Rules []string `yaml:"rules"`
}

func sample() ir.Profile {
	return ir.Profile{Nodes: []ir.Node{
		{
			Tag: "vless", Protocol: "vless", Server: "example.com", Port: 443,
			Account: "uuid-1", Transport: ir.TransportTCP,
			Extra: map[string]string{"network": "tcp", "tls": "true", "servername": "example.com", "flow": "xtls-rprx-vision"},
		},
		{Tag: "trojan", Protocol: "trojan", Server: "example.net", Port: 443, Secret: "pw", Extra: map[string]string{"sni": "example.net"}},
		{Tag: "ss", Protocol: "ss", Server: "example.org", Port: 8388, Account: "aes-128-gcm", Secret: "pw"},
		{Tag: "unknown", Protocol: "wireguard", Server: "example.io", Port: 51820},
	}}
}

func TestEmitMapsSupportedNodes(t *testing.T) {
	data, report, err := Emit(sample(), Options{Port: 1080})
	if err != nil {
		t.Fatal(err)
	}
	var got decoded
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.SocksPort != 1080 || got.Mode != "rule" {
		t.Fatalf("unexpected config: %#v", got)
	}
	if len(got.Proxies) != 3 {
		t.Fatalf("expected 3 proxies, got %d", len(got.Proxies))
	}
	if got.Proxies[0].Type != "vless" || got.Proxies[0].Flow != "xtls-rprx-vision" || got.Proxies[0].Servername != "example.com" {
		t.Fatalf("vless mapping is wrong: %#v", got.Proxies[0])
	}
	if got.Proxies[1].SNI != "example.net" || got.Proxies[1].Password != "pw" {
		t.Fatalf("trojan mapping is wrong: %#v", got.Proxies[1])
	}
	if got.Proxies[2].Cipher != "aes-128-gcm" {
		t.Fatalf("ss mapping is wrong: %#v", got.Proxies[2])
	}
	if len(got.ProxyGroups) != 1 || got.ProxyGroups[0].Name != "PROXY" {
		t.Fatalf("group is wrong: %#v", got.ProxyGroups)
	}
	if len(got.Rules) != 1 || got.Rules[0] != "MATCH,PROXY" {
		t.Fatalf("rules are wrong: %#v", got.Rules)
	}
	if len(report.Dropped) != 1 || report.Dropped[0] != "unknown" {
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
