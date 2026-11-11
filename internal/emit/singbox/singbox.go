package singbox

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nichind/any2proxy/internal/ir"
)

type tlsConfig struct {
	Enabled    bool   `json:"enabled"`
	ServerName string `json:"server_name,omitempty"`
}

type inbound struct {
	Type       string `json:"type"`
	Listen     string `json:"listen"`
	ListenPort uint16 `json:"listen_port"`
}

type route struct {
	Final string `json:"final"`
}

type outbound struct {
	Type       string     `json:"type"`
	Tag        string     `json:"tag"`
	Server     string     `json:"server"`
	ServerPort uint16     `json:"server_port"`
	UUID       string     `json:"uuid,omitempty"`
	Password   string     `json:"password,omitempty"`
	Method     string     `json:"method,omitempty"`
	Flow       string     `json:"flow,omitempty"`
	Security   string     `json:"security,omitempty"`
	TLS        *tlsConfig `json:"tls,omitempty"`
}

type config struct {
	Inbounds  []inbound  `json:"inbounds,omitempty"`
	Outbounds []outbound `json:"outbounds"`
	Route     *route     `json:"route,omitempty"`
}

type Options struct {
	Strict    bool
	SocksPort uint16
}

var supported = []string{"hysteria2", "shadowsocks", "trojan", "tuic", "vless", "vmess"}

func Emit(profile ir.Profile, strict bool) ([]byte, ir.CapabilityReport, error) {
	return EmitWithOptions(profile, Options{Strict: strict})
}

func EmitWithOptions(profile ir.Profile, opts Options) ([]byte, ir.CapabilityReport, error) {
	if len(profile.Nodes) == 0 {
		return nil, ir.CapabilityReport{}, fmt.Errorf("profile has no nodes")
	}

	report := ir.CapabilityReport{TargetSupported: append([]string(nil), supported...)}
	cfg := config{Outbounds: make([]outbound, 0, len(profile.Nodes))}

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

	if opts.SocksPort > 0 {
		cfg.Inbounds = []inbound{{Type: "socks", Listen: "127.0.0.1", ListenPort: opts.SocksPort}}
		cfg.Route = &route{Final: cfg.Outbounds[0].Tag}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, report, err
	}
	return data, report, nil
}

func convert(node ir.Node) (outbound, bool) {
	out := outbound{
		Tag:        node.Tag,
		Server:     node.Server,
		ServerPort: node.Port,
	}

	switch node.Protocol {
	case "vless":
		out.Type = "vless"
		out.UUID = node.Account
		out.Flow = extra(node, "flow")
		out.TLS = tlsFor(node, false)
	case "vmess":
		out.Type = "vmess"
		out.UUID = node.Account
		out.Security = "auto"
		out.TLS = tlsFor(node, false)
	case "trojan":
		out.Type = "trojan"
		out.Password = node.Secret
		out.TLS = tlsFor(node, false)
	case "ss":
		out.Type = "shadowsocks"
		out.Method = node.Account
		out.Password = node.Secret
	case "hysteria2":
		out.Type = "hysteria2"
		out.Password = node.Secret
		out.TLS = tlsFor(node, true)
	case "tuic":
		out.Type = "tuic"
		out.UUID = node.Account
		out.Password = node.Secret
		out.TLS = tlsFor(node, true)
	default:
		return outbound{}, false
	}
	return out, true
}

func tlsFor(node ir.Node, fallback bool) *tlsConfig {
	enabled := fallback
	if extra(node, "tls") == "true" {
		enabled = true
	}
	serverName := extra(node, "sni")
	if serverName == "" {
		serverName = extra(node, "servername")
	}
	if !enabled && serverName == "" {
		return nil
	}
	return &tlsConfig{Enabled: enabled, ServerName: serverName}
}

func extra(node ir.Node, key string) string {
	if node.Extra == nil {
		return ""
	}
	return node.Extra[key]
}
