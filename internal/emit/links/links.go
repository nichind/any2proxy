package links

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"

	"github.com/nichind/any2proxy/internal/ir"
)

func Emit(profile ir.Profile) ([]string, []string) {
	out := make([]string, 0, len(profile.Nodes))
	dropped := make([]string, 0)
	for _, node := range profile.Nodes {
		link, ok := linkFor(node)
		if !ok {
			dropped = append(dropped, node.Tag)
			continue
		}
		out = append(out, link)
	}
	return out, dropped
}

func linkFor(node ir.Node) (string, bool) {
	host := net.JoinHostPort(node.Server, strconv.Itoa(int(node.Port)))
	switch node.Protocol {
	case "vless":
		return vless(node, host), true
	case "vmess":
		return vmess(node, host), true
	case "trojan":
		return trojan(node, host), true
	case "ss":
		return shadowsocks(node, host), true
	case "hysteria2":
		return hysteria2(node, host), true
	case "tuic":
		return tuic(node, host), true
	case "anytls":
		return anytls(node, host), true
	default:
		return "", false
	}
}

func vless(node ir.Node, host string) string {
	u := &url.URL{Scheme: "vless", User: url.User(node.Account), Host: host, Fragment: node.Tag}
	q := url.Values{}
	q.Set("encryption", "none")
	q.Set("type", network(node))
	security := "none"
	if extra(node, "public-key") != "" {
		security = "reality"
		q.Set("pbk", extra(node, "public-key"))
		if short := extra(node, "short-id"); short != "" {
			q.Set("sid", short)
		}
	} else if extra(node, "tls") == "true" || serverName(node) != "" {
		security = "tls"
	}
	q.Set("security", security)
	setCommon(q, node)
	u.RawQuery = q.Encode()
	return u.String()
}

func vmess(node ir.Node, host string) string {
	payload := map[string]string{
		"v":    "2",
		"ps":   node.Tag,
		"add":  node.Server,
		"port": strconv.Itoa(int(node.Port)),
		"id":   node.Account,
		"aid":  "0",
		"net":  network(node),
		"type": "none",
	}
	if extra(node, "tls") == "true" || serverName(node) != "" {
		payload["tls"] = "tls"
	}
	if name := serverName(node); name != "" {
		payload["sni"] = name
	}
	data, _ := json.Marshal(payload)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func trojan(node ir.Node, host string) string {
	u := &url.URL{Scheme: "trojan", User: url.User(node.Secret), Host: host, Fragment: node.Tag}
	q := url.Values{}
	if extra(node, "tls") == "true" || serverName(node) != "" {
		q.Set("security", "tls")
	}
	q.Set("type", network(node))
	setCommon(q, node)
	u.RawQuery = q.Encode()
	return u.String()
}

func shadowsocks(node ir.Node, host string) string {
	userinfo := base64.StdEncoding.EncodeToString([]byte(node.Account + ":" + node.Secret))
	return "ss://" + userinfo + "@" + host + "#" + url.PathEscape(node.Tag)
}

func hysteria2(node ir.Node, host string) string {
	u := &url.URL{Scheme: "hysteria2", User: url.User(node.Secret), Host: host, Fragment: node.Tag}
	q := url.Values{}
	setCommon(q, node)
	if obfs := extra(node, "obfs"); obfs != "" {
		q.Set("obfs", obfs)
	}
	if password := extra(node, "obfs-password"); password != "" {
		q.Set("obfs-password", password)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func tuic(node ir.Node, host string) string {
	u := &url.URL{Scheme: "tuic", User: url.UserPassword(node.Account, node.Secret), Host: host, Fragment: node.Tag}
	q := url.Values{}
	setCommon(q, node)
	u.RawQuery = q.Encode()
	return u.String()
}

func anytls(node ir.Node, host string) string {
	u := &url.URL{Scheme: "anytls", User: url.User(node.Secret), Host: host, Fragment: node.Tag}
	q := url.Values{}
	setCommon(q, node)
	u.RawQuery = q.Encode()
	return u.String()
}

func setCommon(q url.Values, node ir.Node) {
	if name := serverName(node); name != "" {
		q.Set("sni", name)
	}
	if fp := extra(node, "client-fingerprint"); fp != "" {
		q.Set("fp", fp)
	} else if fp := extra(node, "fingerprint"); fp != "" {
		q.Set("fp", fp)
	}
	if extra(node, "skip-cert-verify") == "true" {
		q.Set("insecure", "1")
	}
	if flow := extra(node, "flow"); flow != "" {
		q.Set("flow", flow)
	}
	if alpn := extra(node, "alpn"); alpn != "" {
		q.Set("alpn", alpn)
	}
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
