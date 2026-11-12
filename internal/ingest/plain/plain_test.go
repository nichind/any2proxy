package plain

import (
	"strings"
	"testing"
)

func TestParseLinks(t *testing.T) {
	vless := "vless" + "://"
	trojan := "trojan" + "://"
	ss := "ss" + "://"
	input := strings.Join([]string{
		vless + "uuid-1@example.com:443#one",
		trojan + "secret@example.net#two",
		ss + "aes-128-gcm:pass@example.org:8388#three",
		"",
		"# comment",
		"not a link",
	}, "\n")

	profile, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(profile.Nodes))
	}
	if len(profile.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(profile.Warnings))
	}
	if profile.Nodes[0].Port != 443 {
		t.Fatalf("default port not applied: %d", profile.Nodes[0].Port)
	}
	if profile.Nodes[1].Secret != "secret" {
		t.Fatal("trojan password did not land in Secret")
	}
	if profile.Nodes[2].Secret != "pass" {
		t.Fatal("ss password did not land in Secret")
	}
}

func TestParseRejectsOversizedLine(t *testing.T) {
	profile, err := Parse(strings.NewReader("vless" + "://u@example.com/" + strings.Repeat("a", 5000)))
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Nodes) != 0 || len(profile.Warnings) != 1 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}
