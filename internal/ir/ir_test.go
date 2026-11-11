package ir

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNodeIDStableAcrossSecret(t *testing.T) {
	a := Node{Protocol: "vless", Server: "example.com", Port: 443, Account: "u1", Secret: "one"}
	b := a
	b.Secret = "two"
	if a.ID() != b.ID() {
		t.Fatalf("secret changed the identity: %s != %s", a.ID(), b.ID())
	}
}

func TestNodeIDChangesWithServer(t *testing.T) {
	a := Node{Protocol: "vless", Server: "example.com", Port: 443, Account: "u1"}
	b := a
	b.Server = "example.net"
	if a.ID() == b.ID() {
		t.Fatal("server change kept the identity")
	}
}

func TestNormalizeTags(t *testing.T) {
	nodes := []Node{{Tag: "x"}, {Tag: "x"}, {Tag: ""}}
	got, warnings := NormalizeTags(nodes)
	if got[0].Tag != "x" || got[1].Tag != "x#2" || got[2].Tag != "node" {
		t.Fatalf("unexpected tags: %#v", got)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(warnings))
	}
}

func TestValidate(t *testing.T) {
	if err := (Node{Protocol: "vless", Server: "example.com", Port: 443}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Node{Protocol: "vless", Server: "", Port: 443}).Validate(); err == nil {
		t.Fatal("empty server passed validation")
	}
	if err := (Node{Protocol: "vless", Server: "example.com", Port: 0}).Validate(); err == nil {
		t.Fatal("zero port passed validation")
	}
}

func TestProfileJSONOmitsSecret(t *testing.T) {
	p := Profile{Nodes: []Node{{Protocol: "trojan", Server: "example.com", Port: 443, Secret: "hunter2"}}}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "hunter2") {
		t.Fatalf("secret leaked: %s", b)
	}
}
