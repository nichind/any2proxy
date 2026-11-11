package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func server(t *testing.T, body string) (*httptest.Server, *int) {
	t.Helper()
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &count
}

func TestGetUsesETagCache(t *testing.T) {
	srv, count := server(t, "hello")
	f := New(Options{CacheDir: t.TempDir(), AllowPrivate: true})

	body, meta, err := f.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" || meta.ETag != `"v1"` {
		t.Fatalf("unexpected response: %q %#v", body, meta)
	}
	body, _, err = f.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("unexpected cached body: %q", body)
	}
	if *count != 2 {
		t.Fatalf("expected 2 requests, got %d", *count)
	}
}

func TestNoCacheRefetches(t *testing.T) {
	srv, count := server(t, "hello")
	f := New(Options{CacheDir: t.TempDir(), AllowPrivate: true, NoCache: true})
	if _, _, err := f.Get(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.Get(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if *count != 2 {
		t.Fatalf("expected 2 requests, got %d", *count)
	}
}

func TestMaxBytes(t *testing.T) {
	srv, _ := server(t, strings.Repeat("x", 100))
	f := New(Options{CacheDir: t.TempDir(), AllowPrivate: true, MaxBytes: 10})
	if _, _, err := f.Get(context.Background(), srv.URL); err == nil {
		t.Fatal("oversized body passed")
	}
}

func TestOfflineUsesCache(t *testing.T) {
	srv, _ := server(t, "hello")
	dir := t.TempDir()
	f := New(Options{CacheDir: dir, AllowPrivate: true})
	if _, _, err := f.Get(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}

	offline := New(Options{CacheDir: dir, AllowPrivate: true, Offline: true})
	body, _, err := offline.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("unexpected offline body: %q", body)
	}
}

func TestPrivateAddressBlocked(t *testing.T) {
	srv, _ := server(t, "hello")
	f := New(Options{CacheDir: t.TempDir()})
	if _, _, err := f.Get(context.Background(), srv.URL); err == nil {
		t.Fatal("private address passed")
	}
}

func TestCacheDoesNotContainURL(t *testing.T) {
	srv, _ := server(t, "hello")
	dir := t.TempDir()
	f := New(Options{CacheDir: dir, AllowPrivate: true})
	if _, _, err := f.Get(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "127.0.0.1") {
			t.Fatalf("cache file leaks the host: %s", entry.Name())
		}
	}
}
