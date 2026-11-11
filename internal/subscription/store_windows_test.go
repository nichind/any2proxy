//go:build windows

package subscription

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreEncryptsURLOnWindows(t *testing.T) {
	secret := "tok" + "en=secret"
	rawURL := "https://example.com/sub?" + secret
	path := filepath.Join(t.TempDir(), "subscriptions.json")
	store, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Add("main", rawURL); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(secret)) {
		t.Fatal("subscription URL is stored in plaintext")
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reloaded.Get("main")
	if err != nil {
		t.Fatal(err)
	}
	if entry.URL != rawURL {
		t.Fatalf("unexpected URL: %s", entry.URL)
	}
}
