package detect

import (
	"bytes"
	"encoding/base64"
	"strings"
)

func Decode(body []byte) (string, []byte) {
	trimmed := bytes.TrimSpace(body)
	if isClash(trimmed) {
		return "clash", trimmed
	}
	if bytes.Contains(trimmed, []byte("://")) {
		return "plain", trimmed
	}
	decoded, ok := decodeBase64(trimmed)
	if ok && bytes.Contains(decoded, []byte("://")) {
		if isClash(decoded) {
			return "clash", decoded
		}
		return "plain", decoded
	}
	return "plain", trimmed
}

func isClash(body []byte) bool {
	text := string(body)
	return strings.HasPrefix(strings.TrimSpace(text), "proxies:") ||
		strings.Contains(text, "\nproxies:")
}

func decodeBase64(body []byte) ([]byte, bool) {
	text := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, string(body))
	if text == "" {
		return nil, false
	}
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if decoded, err := encoding.DecodeString(text); err == nil {
			return decoded, true
		}
	}
	return nil, false
}
