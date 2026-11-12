package cores

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSelectAsset(t *testing.T) {
	assets := []asset{
		{Name: "sing-box-1.14.4-windows-amd64.zip"},
		{Name: "sing-box-1.14.4-windows-amd64-legacy-windows-7.zip"},
		{Name: "sing-box-1.14.4-linux-amd64.tar.gz"},
		{Name: "sing-box-1.14.4-linux-amd64-glibc.tar.gz"},
	}
	item, err := selectAsset("sing-box", assets, "windows", "amd64", "")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "sing-box-1.14.4-windows-amd64.zip" {
		t.Fatalf("unexpected asset: %s", item.Name)
	}
	item, err = selectAsset("sing-box", assets, "linux", "amd64", "")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "sing-box-1.14.4-linux-amd64.tar.gz" {
		t.Fatalf("unexpected asset: %s", item.Name)
	}

	mihomoAssets := []asset{
		{Name: "mihomo-linux-amd64-v1-go120-v1.19.34.gz"},
		{Name: "mihomo-linux-amd64-v1.19.34.gz"},
		{Name: "mihomo-linux-amd64-compatible-v1.19.34.gz"},
	}
	item, err = selectAsset("mihomo", mihomoAssets, "linux", "amd64", "")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "mihomo-linux-amd64-v1.19.34.gz" {
		t.Fatalf("unexpected mihomo asset: %s", item.Name)
	}

	xrayAssets := []asset{{Name: "Xray-windows-arm64-v8a.zip"}, {Name: "Xray-windows-64.zip"}}
	item, err = selectAsset("xray", xrayAssets, "windows", "amd64", "")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Xray-windows-64.zip" {
		t.Fatalf("unexpected xray asset: %s", item.Name)
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "core.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("xray.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("binary"))
	_ = writer.Close()
	_ = file.Close()

	binary, err := extractBinary(path, "Xray-windows-64.zip", "xray", "windows", dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "binary" {
		t.Fatalf("unexpected binary: %q", data)
	}
}

func TestExtractGz(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "core.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	_, _ = writer.Write([]byte("binary"))
	_ = writer.Close()
	_ = file.Close()

	binary, err := extractBinary(path, "mihomo-linux-amd64.gz", "mihomo", "linux", dir)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(binary)
	if string(data) != "binary" {
		t.Fatalf("unexpected binary: %q", data)
	}
}

func TestInstallFromURL(t *testing.T) {
	archive := zipBytes(t, "custom.exe", []byte("binary"))
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	store, err := Load(filepath.Join(t.TempDir(), "cores.json"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.Install(context.Background(), "fork", InstallOptions{
		URL:    server.URL + "/custom.zip",
		SHA256: hex.EncodeToString(sum[:]),
		OS:     "windows",
		Arch:   "amd64",
		Binary: "custom",
		Client: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(entry.Path) != "custom.exe" {
		t.Fatalf("unexpected path: %s", entry.Path)
	}
}

func TestInstallFromRelease(t *testing.T) {
	archive := zipBytes(t, "xray.exe", []byte("binary"))
	sum := sha256.Sum256(archive)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download/core.zip" {
			_, _ = w.Write(archive)
			return
		}
		response := map[string]any{
			"tag_name": "v1.0.0",
			"assets": []map[string]any{{
				"name":                 "Xray-windows-64.zip",
				"browser_download_url": server.URL + "/download/core.zip",
				"digest":               "sha256:" + hex.EncodeToString(sum[:]),
			}},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	store, err := Load(filepath.Join(t.TempDir(), "cores.json"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.Install(context.Background(), "xray", InstallOptions{
		APIBase: server.URL,
		OS:      "windows",
		Arch:    "amd64",
		Client:  server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Version != "1.0.0" {
		t.Fatalf("unexpected version: %s", entry.Version)
	}
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "binary" {
		t.Fatalf("unexpected binary: %q", data)
	}
}

func zipBytes(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
