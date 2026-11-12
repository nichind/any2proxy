package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/nichind/any2proxy/internal/service"
)

type reply struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func line(t *testing.T, id int, method string, params any) string {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]any{"id": id, "method": method, "params": json.RawMessage(raw)})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func serve(t *testing.T, input string) []reply {
	t.Helper()
	var out bytes.Buffer
	if err := Serve(context.Background(), strings.NewReader(input), &out, service.New()); err != nil {
		t.Fatal(err)
	}
	var replies []reply
	decoder := json.NewDecoder(&out)
	for {
		var item reply
		err := decoder.Decode(&item)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		replies = append(replies, item)
	}
	return replies
}

func TestVersionAndCapabilities(t *testing.T) {
	input := line(t, 1, "version", nil) + "\n" + line(t, 2, "capabilities", nil) + "\n"
	replies := serve(t, input)
	if len(replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(replies))
	}
	if !bytes.Contains(replies[0].Result, []byte("version")) {
		t.Fatalf("unexpected version: %s", replies[0].Result)
	}
	if !bytes.Contains(replies[1].Result, []byte("sing-box")) {
		t.Fatalf("unexpected capabilities: %s", replies[1].Result)
	}
}

func TestParseAndCompile(t *testing.T) {
	body := "vless" + "://u@example.com:443#one\n"
	input := line(t, 1, "parse", map[string]any{"body": body, "format": "plain"}) + "\n" +
		line(t, 2, "compile", map[string]any{"body": body, "format": "plain", "target": "mihomo", "port": 1080}) + "\n"
	replies := serve(t, input)
	if len(replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(replies))
	}
	if replies[0].Error != nil {
		t.Fatal(replies[0].Error.Message)
	}
	if replies[1].Error != nil {
		t.Fatal(replies[1].Error.Message)
	}
	if !bytes.Contains(replies[1].Result, []byte("socks-port")) {
		t.Fatalf("unexpected compile result: %s", replies[1].Result)
	}
}

func TestUnknownMethod(t *testing.T) {
	replies := serve(t, line(t, 1, "nope", nil)+"\n")
	if len(replies) != 1 || replies[0].Error == nil {
		t.Fatal("expected an error")
	}
}
