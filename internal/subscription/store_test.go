package subscription

import (
	"path/filepath"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subscriptions.json")
	store, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Add("main", "https://example.com/sub"); err != nil {
		t.Fatal(err)
	}
	if err := store.Add("backup", "https://example.net/sub"); err != nil {
		t.Fatal(err)
	}
	if err := store.Add("main", "https://example.org/sub"); err == nil {
		t.Fatal("duplicate name passed")
	}
	if err := store.Update("main", 12, "ok"); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reloaded.Get("main")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Nodes != 12 || entry.LastStatus != "ok" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if len(reloaded.List()) != 2 {
		t.Fatalf("unexpected list: %#v", reloaded.List())
	}
	if err := reloaded.Remove("main"); err != nil {
		t.Fatal(err)
	}
	if _, err := reloaded.Get("main"); err == nil {
		t.Fatal("removed entry still exists")
	}
}

func TestStoreRejectsInvalidInput(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "subscriptions.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Add("bad name", "https://example.com/sub"); err == nil {
		t.Fatal("invalid name passed")
	}
	if err := store.Add("main", "file:///tmp/sub"); err == nil {
		t.Fatal("invalid url passed")
	}
	if _, err := store.Get("missing"); err == nil {
		t.Fatal("missing entry passed")
	}
}
