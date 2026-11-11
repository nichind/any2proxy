package clash

import (
	"fmt"
	"io"
	"strings"

	"github.com/nichind/any2proxy/internal/ir"
	"gopkg.in/yaml.v3"
)

type document struct {
	Proxies []proxy `yaml:"proxies"`
}

type proxy struct {
	Name       string         `yaml:"name"`
	Type       string         `yaml:"type"`
	Server     string         `yaml:"server"`
	Port       uint16         `yaml:"port"`
	UUID       string         `yaml:"uuid"`
	Password   string         `yaml:"password"`
	Cipher     string         `yaml:"cipher"`
	Network    string         `yaml:"network"`
	TLS        bool           `yaml:"tls"`
	SNI        string         `yaml:"sni"`
	Servername string         `yaml:"servername"`
	Flow       string         `yaml:"flow"`
	SkipCert   bool           `yaml:"skip-cert-verify"`
	Extra      map[string]any `yaml:",inline"`
}

func Parse(r io.Reader) (ir.Profile, error) {
	var doc document
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&doc); err != nil {
		return ir.Profile{}, fmt.Errorf("clash: %w", err)
	}
	if len(doc.Proxies) == 0 {
		return ir.Profile{}, fmt.Errorf("clash: no proxies found")
	}

	profile := ir.Profile{}
	for _, p := range doc.Proxies {
		node, err := convert(p)
		if err != nil {
			profile.Warnings = append(profile.Warnings, ir.Warning{Message: p.Name + ": " + err.Error()})
			continue
		}
		if err := node.Validate(); err != nil {
			profile.Warnings = append(profile.Warnings, ir.Warning{Message: p.Name + ": " + err.Error()})
			continue
		}
		profile.Nodes = append(profile.Nodes, node)
	}

	nodes, warnings := ir.NormalizeTags(profile.Nodes)
	profile.Nodes = nodes
	profile.Warnings = append(profile.Warnings, warnings...)
	return profile, nil
}

func convert(p proxy) (ir.Node, error) {
	protocol := strings.ToLower(p.Type)
	switch protocol {
	case "vless", "vmess", "trojan", "ss", "hysteria2", "hy2", "tuic":
	case "":
		return ir.Node{}, fmt.Errorf("type is empty")
	default:
		return ir.Node{}, fmt.Errorf("unsupported type %q", p.Type)
	}

	node := ir.Node{
		Tag:       p.Name,
		Protocol:  ir.Protocol(protocol),
		Server:    p.Server,
		Port:      p.Port,
		Transport: ir.TransportTCP,
		Extra:     extras(p),
	}
	if node.Tag == "" {
		node.Tag = p.Server
	}
	if protocol == "hy2" {
		node.Protocol = "hysteria2"
	}
	if node.Protocol == "hysteria2" || node.Protocol == "tuic" {
		node.Transport = ir.TransportUDP
	}

	switch protocol {
	case "vless", "vmess", "tuic":
		node.Account = p.UUID
	case "ss":
		node.Account = p.Cipher
	}
	switch protocol {
	case "trojan", "hysteria2", "hy2", "tuic":
		node.Secret = p.Password
	case "ss":
		node.Secret = p.Password
	}
	return node, nil
}

func extras(p proxy) map[string]string {
	extra := map[string]string{}
	if p.Network != "" {
		extra["network"] = p.Network
	}
	if p.TLS {
		extra["tls"] = "true"
	}
	if p.SNI != "" {
		extra["sni"] = p.SNI
	}
	if p.Servername != "" {
		extra["servername"] = p.Servername
	}
	if p.Flow != "" {
		extra["flow"] = p.Flow
	}
	if p.SkipCert {
		extra["skip-cert-verify"] = "true"
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}
