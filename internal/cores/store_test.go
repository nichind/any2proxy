package cores

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "core")
	if err := os.WriteFile(source, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	store, err := Load(filepath.Join(dir, "cores.json"), filepath.Join(dir, "cores"))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.Import("sing-box", source, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if entry.SHA256 == "" || entry.Version != "1.0.0" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if err := store.Verify("sing-box"); err != nil {
		t.Fatal(err)
	}
	if len(store.List()) != 1 {
		t.Fatal("entry not listed")
	}

	if err := os.WriteFile(entry.Path, []byte("changed"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.Verify("sing-box"); err == nil {
		t.Fatal("tampered binary passed")
	}

	if err := store.Remove("sing-box"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("sing-box"); err == nil {
		t.Fatal("removed entry still exists")
	}
}

func TestStoreRejectsInvalidInput(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "cores.json"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Import("bad id", "missing", ""); err == nil {
		t.Fatal("invalid id passed")
	}
	if _, err := store.Import("sing-box", "missing", ""); err == nil {
		t.Fatal("missing source passed")
	}
}
