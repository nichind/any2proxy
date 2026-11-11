package ir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type Protocol string

type Transport string

const (
	TransportTCP Transport = "tcp"
	TransportUDP Transport = "udp"
)

type Node struct {
	Tag       string            `json:"tag"`
	Protocol  Protocol          `json:"protocol"`
	Server    string            `json:"server"`
	Port      uint16            `json:"port"`
	Transport Transport         `json:"transport,omitempty"`
	Account   string            `json:"account,omitempty"`
	Secret    string            `json:"-"`
	Extra     map[string]string `json:"extra,omitempty"`
}

func (n Node) ID() string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%d\x00%s\x00%s", n.Protocol, n.Server, n.Port, n.Transport, n.Account)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (n Node) Validate() error {
	if n.Protocol == "" {
		return fmt.Errorf("protocol is empty")
	}
	if n.Server == "" {
		return fmt.Errorf("server is empty")
	}
	if n.Port == 0 {
		return fmt.Errorf("port is zero")
	}
	if strings.ContainsAny(n.Server, " \t\r\n") {
		return fmt.Errorf("server contains whitespace")
	}
	return nil
}
