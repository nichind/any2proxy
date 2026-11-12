package plain

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/nichind/any2proxy/internal/ir"
)

const maxLine = 4096

func Parse(r io.Reader) (ir.Profile, error) {
	profile := ir.Profile{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		text = strings.TrimPrefix(text, "\uFEFF")
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if len(text) > maxLine {
			profile.Warnings = append(profile.Warnings, ir.Warning{Line: line, Message: "line is too long"})
			continue
		}
		node, err := parseLink(text)
		if err != nil {
			profile.Warnings = append(profile.Warnings, ir.Warning{Line: line, Message: err.Error()})
			continue
		}
		if err := node.Validate(); err != nil {
			profile.Warnings = append(profile.Warnings, ir.Warning{Line: line, Message: err.Error()})
			continue
		}
		profile.Nodes = append(profile.Nodes, node)
	}
	if err := scanner.Err(); err != nil {
		return profile, err
	}

	nodes, tagWarnings := ir.NormalizeTags(profile.Nodes)
	profile.Nodes = nodes
	profile.Warnings = append(profile.Warnings, tagWarnings...)
	return profile, nil
}

func parseLink(text string) (ir.Node, error) {
	if strings.HasPrefix(strings.ToLower(text), "vmess://") {
		if node, err := parseVMess(strings.TrimPrefix(text, "vmess://")); err == nil {
			return node, nil
		}
	}
	u, err := url.Parse(text)
	if err != nil {
		return ir.Node{}, fmt.Errorf("invalid link: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return ir.Node{}, fmt.Errorf("link has no scheme or host")
	}

	protocol := strings.ToLower(u.Scheme)
	portText := u.Port()
	if portText == "" {
		portText = strconv.Itoa(defaultPort(protocol))
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return ir.Node{}, fmt.Errorf("invalid port %q", portText)
	}

	node := ir.Node{
		Tag:       u.Fragment,
		Protocol:  ir.Protocol(protocol),
		Server:    u.Hostname(),
		Port:      uint16(port),
		Transport: transportFor(protocol),
	}
	if node.Tag == "" {
		node.Tag = node.Server
	}
	node.Extra = extrasFromQuery(u.Query())

	if u.User != nil {
		user := u.User.Username()
		pass, hasPass := u.User.Password()
		switch protocol {
		case "trojan", "hysteria2", "hy2", "anytls":
			node.Secret = user
		case "ss":
			if hasPass {
				node.Account = user
				node.Secret = pass
			} else if decoded, ok := decodeSSUser(user); ok {
				node.Account = decoded[0]
				node.Secret = decoded[1]
			} else {
				node.Account = user
			}
		default:
			node.Account = user
			if hasPass {
				node.Secret = pass
			}
		}
	}

	return node, nil
}

func extrasFromQuery(query url.Values) map[string]string {
	extra := map[string]string{}
	set := func(key string) {
		if value := query.Get(key); value != "" {
			extra[key] = value
		}
	}
	set("sni")
	set("servername")
	set("flow")
	set("obfs")
	set("obfs-password")
	set("alpn")
	if value := query.Get("pbk"); value != "" {
		extra["public-key"] = value
	}
	if value := query.Get("sid"); value != "" {
		extra["short-id"] = value
	}
	if value := query.Get("fp"); value != "" {
		extra["client-fingerprint"] = value
	}
	if value := query.Get("type"); value != "" {
		extra["network"] = value
	}
	if value := query.Get("security"); value == "tls" || value == "reality" {
		extra["tls"] = "true"
	}
	if value := query.Get("insecure"); value == "1" || value == "true" {
		extra["skip-cert-verify"] = "true"
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

func decodeSSUser(value string) ([2]string, bool) {
	decoded, err := decodeBase64(value)
	if err != nil {
		return [2]string{}, false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return [2]string{}, false
	}
	return [2]string{parts[0], parts[1]}, true
}

func parseVMess(payload string) (ir.Node, error) {
	decoded, err := decodeBase64(payload)
	if err != nil {
		return ir.Node{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(decoded, &raw); err != nil {
		return ir.Node{}, err
	}
	server := stringValue(raw["add"])
	portText := stringValue(raw["port"])
	if server == "" || portText == "" {
		return ir.Node{}, fmt.Errorf("vmess: missing add or port")
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return ir.Node{}, fmt.Errorf("vmess: invalid port")
	}
	node := ir.Node{
		Tag:       stringValue(raw["ps"]),
		Protocol:  "vmess",
		Server:    server,
		Port:      uint16(port),
		Transport: ir.TransportTCP,
		Account:   stringValue(raw["id"]),
	}
	if node.Tag == "" {
		node.Tag = server
	}
	extra := map[string]string{}
	if network := stringValue(raw["net"]); network != "" {
		extra["network"] = network
	}
	if stringValue(raw["tls"]) == "tls" {
		extra["tls"] = "true"
	}
	if sni := stringValue(raw["sni"]); sni != "" {
		extra["sni"] = sni
	}
	if len(extra) > 0 {
		node.Extra = extra
	}
	return node, nil
}

func decodeBase64(value string) ([]byte, error) {
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if decoded, err := encoding.DecodeString(value); err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("invalid base64")
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return ""
	}
}

func defaultPort(protocol string) int {
	switch protocol {
	case "ss":
		return 8388
	default:
		return 443
	}
}

func transportFor(protocol string) ir.Transport {
	switch protocol {
	case "hy2", "tuic":
		return ir.TransportUDP
	default:
		return ir.TransportTCP
	}
}
