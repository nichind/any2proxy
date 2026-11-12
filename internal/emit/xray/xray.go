package xray

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nichind/any2proxy/internal/ir"
)

type logConfig struct {
	LogLevel string `json:"loglevel"`
}

type inbound struct {
	Tag      string `json:"tag,omitempty"`
	Listen   string `json:"listen"`
	Port     uint16 `json:"port"`
	Protocol string `json:"protocol"`
	Settings struct {
		Auth string `json:"auth"`
		UDP  bool   `json:"udp"`
	} `json:"settings"`
}

type user struct {
	ID         string `json:"id"`
	Encryption string `json:"encryption,omitempty"`
	Flow       string `json:"flow,omitempty"`
	AlterID    int    `json:"alterId,omitempty"`
	Security   string `json:"security,omitempty"`
}

type vnext struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
	Users   []user `json:"users"`
}

type server struct {
	Address  string `json:"address"`
	Port     uint16 `json:"port"`
	Password string `json:"password,omitempty"`
	Method   string `json:"method,omitempty"`
}

type outbound struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       any             `json:"settings"`
	StreamSettings *streamSettings `json:"streamSettings,omitempty"`
}

type streamSettings struct {
	Network         string           `json:"network,omitempty"`
	Security        string           `json:"security,omitempty"`
	TLSSettings     *tlsSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *realitySettings `json:"realitySettings,omitempty"`
}

type tlsSettings struct {
	ServerName string `json:"serverName,omitempty"`
}

type realitySettings struct {
	ServerName  string `json:"serverName,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortID     string `json:"shortId,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type rule struct {
	Type        string `json:"type"`
	Network     string `json:"network"`
	OutboundTag string `json:"outboundTag"`
}

type routing struct {
	DomainStrategy string `json:"domainStrategy,omitempty"`
	Rules          []rule `json:"rules"`
}

type config struct {
	Log       logConfig  `json:"log"`
	Inbounds  []inbound  `json:"inbounds"`
	Outbounds []outbound `json:"outbounds"`
	Routing   routing    `json:"routing"`
}

type Options struct {
	Port   uint16
	Strict bool
}

var supported = []string{"shadowsocks", "trojan", "vless", "vmess"}

func Emit(profile ir.Profile, opts Options) ([]byte, ir.CapabilityReport, error) {
	if len(profile.Nodes) == 0 {
		return nil, ir.CapabilityReport{}, fmt.Errorf("profile has no nodes")
	}
	if opts.Port == 0 {
		opts.Port = 1080
	}

	report := ir.CapabilityReport{TargetSupported: append([]string(nil), supported...)}
	cfg := config{
		Log: logConfig{LogLevel: "warning"},
		Inbounds: []inbound{{
			Tag:      "socks",
			Listen:   "127.0.0.1",
			Port:     opts.Port,
			Protocol: "socks",
		}},
		Routing: routing{
			DomainStrategy: "AsIs",
			Rules:          []rule{{Type: "field", Network: "tcp,udp", OutboundTag: "proxy"}},
		},
	}
	cfg.Inbounds[0].Settings.Auth = "noauth"
	cfg.Inbounds[0].Settings.UDP = true

	for _, node := range profile.Nodes {
		out, ok := convert(node)
		if !ok {
			report.Dropped = append(report.Dropped, node.Tag)
			continue
		}
		report.Mapped = append(report.Mapped, node.Tag)
		cfg.Outbounds = append(cfg.Outbounds, out)
	}

	if opts.Strict && len(report.Dropped) > 0 {
		sort.Strings(report.Dropped)
		return nil, report, fmt.Errorf("unsupported nodes: %s", strings.Join(report.Dropped, ", "))
	}
	if len(cfg.Outbounds) == 0 {
		return nil, report, fmt.Errorf("no supported nodes")
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, report, err
	}
	return data, report, nil
}

func convert(node ir.Node) (outbound, bool) {
	out := outbound{Tag: "proxy", StreamSettings: streamFor(node)}

	switch node.Protocol {
	case "vless":
		out.Protocol = "vless"
		out.Settings = map[string]any{"vnext": []vnext{{
			Address: node.Server,
			Port:    node.Port,
			Users: []user{{
				ID:         node.Account,
				Encryption: "none",
				Flow:       extra(node, "flow"),
			}},
		}}}
	case "vmess":
		out.Protocol = "vmess"
		out.Settings = map[string]any{"vnext": []vnext{{
			Address: node.Server,
			Port:    node.Port,
			Users: []user{{
				ID:       node.Account,
				AlterID:  0,
				Security: "auto",
			}},
		}}}
	case "trojan":
		out.Protocol = "trojan"
		out.Settings = map[string]any{"servers": []server{{
			Address:  node.Server,
			Port:     node.Port,
			Password: node.Secret,
		}}}
	case "ss":
		out.Protocol = "shadowsocks"
		out.Settings = map[string]any{"servers": []server{{
			Address:  node.Server,
			Port:     node.Port,
			Password: node.Secret,
			Method:   node.Account,
		}}}
		out.StreamSettings = nil
	default:
		return outbound{}, false
	}
	return out, true
}

func streamFor(node ir.Node) *streamSettings {
	if node.Protocol == "ss" {
		return nil
	}
	stream := &streamSettings{Network: network(node)}
	name := serverName(node)
	if key := extra(node, "public-key"); key != "" {
		stream.Security = "reality"
		stream.RealitySettings = &realitySettings{
			ServerName:  name,
			PublicKey:   key,
			ShortID:     extra(node, "short-id"),
			Fingerprint: firstNonEmpty(extra(node, "client-fingerprint"), extra(node, "fingerprint")),
		}
		return stream
	}
	if name != "" || extra(node, "tls") == "true" {
		stream.Security = "tls"
		stream.TLSSettings = &tlsSettings{ServerName: name}
	}
	return stream
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func network(node ir.Node) string {
	if value := extra(node, "network"); value != "" {
		return value
	}
	return "tcp"
}

func serverName(node ir.Node) string {
	if value := extra(node, "sni"); value != "" {
		return value
	}
	return extra(node, "servername")
}

func extra(node ir.Node, key string) string {
	if node.Extra == nil {
		return ""
	}
	return node.Extra[key]
}
