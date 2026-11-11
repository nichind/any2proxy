package plain

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/nichind/any2proxy/internal/ir"
)

const maxLine = 256

func Parse(r io.Reader) (ir.Profile, error) {
	profile := ir.Profile{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
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

	if u.User != nil {
		user := u.User.Username()
		pass, hasPass := u.User.Password()
		switch protocol {
		case "trojan":
			node.Secret = user
		case "ss":
			node.Account = user
			if hasPass {
				node.Secret = pass
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
