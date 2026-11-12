package mihomo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nichind/any2proxy/internal/ir"
	"gopkg.in/yaml.v3"
)

type proxy struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"`
	Server         string   `yaml:"server"`
	Port           uint16   `yaml:"port"`
	UUID           string   `yaml:"uuid,omitempty"`
	Password       string   `yaml:"password,omitempty"`
	Cipher         string   `yaml:"cipher,omitempty"`
	Network        string   `yaml:"network,omitempty"`
	TLS            bool     `yaml:"tls,omitempty"`
	Servername     string   `yaml:"servername,omitempty"`
	SNI            string   `yaml:"sni,omitempty"`
	Flow           string   `yaml:"flow,omitempty"`
	AlterID        int      `yaml:"alterId,omitempty"`
	SkipCertVerify bool     `yaml:"skip-cert-verify,omitempty"`
	ClientFP       string   `yaml:"client-fingerprint,omitempty"`
	Fingerprint    string   `yaml:"fingerprint,omitempty"`
	Obfs           string   `yaml:"obfs,omitempty"`
	ObfsPass       string   `yaml:"obfs-password,omitempty"`
	ALPN           []string `yaml:"alpn,omitempty"`
	RealityOpts    *reality `yaml:"reality-opts,omitempty"`
}

type reality struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id,omitempty"`
}

type group struct {
	Name      string   `yaml:"name"`
	Type      string   `yaml:"type"`
	Proxies   []string `yaml:"proxies"`
	URL       string   `yaml:"url,omitempty"`
	Interval  int      `yaml:"interval,omitempty"`
	Tolerance int      `yaml:"tolerance,omitempty"`
}

type config struct {
	SocksPort          uint16   `yaml:"socks-port,omitempty"`
	ExternalController string   `yaml:"external-controller,omitempty"`
	AllowLAN           bool     `yaml:"allow-lan"`
	Mode               string   `yaml:"mode"`
	LogLevel           string   `yaml:"log-level"`
	Proxies            []proxy  `yaml:"proxies"`
	ProxyGroups        []group  `yaml:"proxy-groups"`
	Rules              []string `yaml:"rules"`
}

type Options struct {
	Port   uint16
	Strict bool
}

var supported = []string{"anytls", "hysteria2", "ss", "trojan", "tuic", "vless", "vmess"}

func Emit(profile ir.Profile, opts Options) ([]byte, ir.CapabilityReport, error) {
	if len(profile.Nodes) == 0 {
		return nil, ir.CapabilityReport{}, fmt.Errorf("profile has no nodes")
	}

	report := ir.CapabilityReport{TargetSupported: append([]string(nil), supported...)}
	cfg := config{
		SocksPort:          opts.Port,
		ExternalController: "127.0.0.1:0",
		Mode:               "rule",
		LogLevel:           "info",
		Proxies:            make([]proxy, 0, len(profile.Nodes)),
		Rules:              []string{"MATCH,PROXY"},
	}

	for _, node := range profile.Nodes {
		p, ok := convert(node)
		if !ok {
			report.Dropped = append(report.Dropped, node.Tag)
			continue
		}
		report.Mapped = append(report.Mapped, node.Tag)
		cfg.Proxies = append(cfg.Proxies, p)
	}

	if opts.Strict && len(report.Dropped) > 0 {
		sort.Strings(report.Dropped)
		return nil, report, fmt.Errorf("unsupported nodes: %s", strings.Join(report.Dropped, ", "))
	}
	if len(cfg.Proxies) == 0 {
		return nil, report, fmt.Errorf("no supported nodes")
	}

	if len(profile.Groups) > 0 {
		for _, item := range profile.Groups {
			cfg.ProxyGroups = append(cfg.ProxyGroups, group{
				Name:      item.Name,
				Type:      item.Type,
				Proxies:   append([]string(nil), item.Proxies...),
				URL:       item.URL,
				Interval:  item.Interval,
				Tolerance: item.Tolerance,
			})
		}
		cfg.Rules = []string{"MATCH," + profile.Groups[0].Name}
	} else {
		cfg.ProxyGroups = []group{{Name: "PROXY", Type: "select", Proxies: report.Mapped}}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, report, err
	}
	return data, report, nil
}

func convert(node ir.Node) (proxy, bool) {
	p := proxy{
		Name:   node.Tag,
		Server: node.Server,
		Port:   node.Port,
	}

	switch node.Protocol {
	case "vless":
		p.Type = "vless"
		p.UUID = node.Account
		p.Network = network(node)
		p.Flow = extra(node, "flow")
		p.TLS, p.Servername = tlsFields(node)
		p.ClientFP = extra(node, "client-fingerprint")
		if key := extra(node, "public-key"); key != "" {
			p.TLS = true
			p.RealityOpts = &reality{PublicKey: key, ShortID: extra(node, "short-id")}
		}
	case "vmess":
		p.Type = "vmess"
		p.UUID = node.Account
		p.Cipher = "auto"
		p.AlterID = 0
		p.Network = network(node)
		p.TLS, p.Servername = tlsFields(node)
	case "trojan":
		p.Type = "trojan"
		p.Password = node.Secret
		p.SNI = serverName(node)
		p.SkipCertVerify = extra(node, "skip-cert-verify") == "true"
	case "ss":
		p.Type = "ss"
		p.Cipher = node.Account
		p.Password = node.Secret
	case "hysteria2":
		p.Type = "hysteria2"
		p.Password = node.Secret
		p.SNI = serverName(node)
		p.Obfs = extra(node, "obfs")
		p.ObfsPass = extra(node, "obfs-password")
		p.Fingerprint = extra(node, "fingerprint")
		p.SkipCertVerify = extra(node, "skip-cert-verify") == "true"
	case "anytls":
		p.Type = "anytls"
		p.Password = node.Secret
		p.SNI = serverName(node)
		p.Fingerprint = extra(node, "fingerprint")
		p.ClientFP = extra(node, "client-fingerprint")
		p.SkipCertVerify = extra(node, "skip-cert-verify") == "true"
	case "tuic":
		p.Type = "tuic"
		p.UUID = node.Account
		p.Password = node.Secret
		p.SNI = serverName(node)
		p.SkipCertVerify = extra(node, "skip-cert-verify") == "true"
	default:
		return proxy{}, false
	}
	if alpn := extra(node, "alpn"); alpn != "" {
		p.ALPN = strings.Split(alpn, ",")
	}
	return p, true
}

func network(node ir.Node) string {
	if value := extra(node, "network"); value != "" {
		return value
	}
	return "tcp"
}

func tlsFields(node ir.Node) (bool, string) {
	name := serverName(node)
	return name != "" || extra(node, "tls") == "true", name
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
