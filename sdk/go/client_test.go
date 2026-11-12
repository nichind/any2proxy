package any2proxy

import (
	"context"
	"strings"
	"testing"
)

func TestClient(t *testing.T) {
	client, err := StartCommand("go", "run", "../../cmd/a2pd")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	version, err := client.Version(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version == "" {
		t.Fatal("empty version")
	}

	body := "vless" + "://u@example.com:443#one\n"
	profile, err := client.Parse(ctx, body, "plain")
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Nodes) != 1 || profile.Nodes[0].Tag != "one" {
		t.Fatalf("unexpected profile: %#v", profile)
	}

	result, err := client.Compile(ctx, body, "plain", "mihomo", 1080, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Config, "socks-port") {
		t.Fatalf("unexpected config: %s", result.Config)
	}
}
