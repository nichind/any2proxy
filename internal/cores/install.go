package cores

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Source struct {
	Repo   string
	Binary string
}

var sources = map[string]Source{
	"sing-box": {Repo: "SagerNet/sing-box", Binary: "sing-box"},
	"mihomo":   {Repo: "MetaCubeX/mihomo", Binary: "mihomo"},
	"xray":     {Repo: "XTLS/Xray-core", Binary: "xray"},
}

type asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type InstallOptions struct {
	Version string
	OS      string
	Arch    string
	APIBase string
	Client  *http.Client
	Repo    string
	URL     string
	Asset   string
	Binary  string
	SHA256  string
}

func (s *Store) Install(ctx context.Context, id string, opts InstallOptions) (Entry, error) {
	source, known := sources[id]
	repo := opts.Repo
	binaryName := opts.Binary
	if repo == "" && known {
		repo = source.Repo
		if binaryName == "" {
			binaryName = source.Binary
		}
	}
	if binaryName == "" {
		binaryName = id
	}
	if opts.URL == "" && repo == "" {
		return Entry{}, fmt.Errorf("cores: unknown id %q; use --repo or --url", id)
	}
	if opts.OS == "" {
		opts.OS = runtime.GOOS
	}
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
	}
	if opts.APIBase == "" {
		opts.APIBase = "https://api.github.com"
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 120 * time.Second}
	}

	if opts.URL != "" {
		return s.installURL(ctx, id, opts, binaryName)
	}
	return s.installRepo(ctx, id, repo, binaryName, opts)
}

func (s *Store) installRepo(ctx context.Context, id, repo, binaryName string, opts InstallOptions) (Entry, error) {
	rel, err := fetchRelease(ctx, opts.Client, opts.APIBase, repo, opts.Version)
	if err != nil {
		return Entry{}, err
	}
	item, err := selectAsset(id, rel.Assets, opts.OS, opts.Arch, opts.Asset)
	if err != nil {
		return Entry{}, err
	}
	archive, err := download(ctx, opts.Client, item)
	if err != nil {
		return Entry{}, err
	}
	defer os.Remove(archive)

	binary, err := prepareBinary(archive, item.Name, binaryName, opts.OS)
	if err != nil {
		return Entry{}, err
	}
	defer os.RemoveAll(filepath.Dir(binary))
	version := strings.TrimPrefix(rel.TagName, "v")
	return s.Import(id, binary, version)
}

func (s *Store) installURL(ctx context.Context, id string, opts InstallOptions, binaryName string) (Entry, error) {
	parsed, err := url.Parse(opts.URL)
	if err != nil || parsed.Host == "" {
		return Entry{}, fmt.Errorf("cores: invalid url")
	}
	item := asset{Name: filepath.Base(parsed.Path), URL: opts.URL}
	if opts.SHA256 != "" {
		item.Digest = "sha256:" + strings.ToLower(strings.TrimPrefix(opts.SHA256, "sha256:"))
	}
	archive, err := download(ctx, opts.Client, item)
	if err != nil {
		return Entry{}, err
	}
	defer os.Remove(archive)

	binary, err := prepareBinary(archive, item.Name, binaryName, opts.OS)
	if err != nil {
		return Entry{}, err
	}
	defer os.RemoveAll(filepath.Dir(binary))
	version := opts.Version
	if version == "latest" {
		version = ""
	}
	return s.Import(id, binary, version)
}

func prepareBinary(path, assetName, binaryName, goos string) (string, error) {
	dir, err := os.MkdirTemp("", "any2proxy-core-*")
	if err != nil {
		return "", fmt.Errorf("cores: %w", err)
	}
	name := strings.ToLower(assetName)
	if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".gz") {
		binary, err := extractBinary(path, assetName, binaryName, goos, dir)
		if err != nil {
			_ = os.RemoveAll(dir)
			return "", err
		}
		return binary, nil
	}
	target := filepath.Join(dir, binaryName)
	if goos == "windows" && !strings.HasSuffix(target, ".exe") {
		target += ".exe"
	}
	if err := copyFile(path, target); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return target, nil
}

func fetchRelease(ctx context.Context, client *http.Client, apiBase, repo, version string) (release, error) {
	paths := []string{}
	if version == "" || version == "latest" {
		paths = append(paths, "/repos/"+repo+"/releases/latest")
	} else {
		paths = append(paths, "/repos/"+repo+"/releases/tags/"+url.PathEscape(version))
		if !strings.HasPrefix(version, "v") {
			paths = append(paths, "/repos/"+repo+"/releases/tags/v"+url.PathEscape(version))
		}
	}

	var last error
	for _, path := range paths {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(apiBase, "/")+path, nil)
		if err != nil {
			return release{}, err
		}
		request.Header.Set("User-Agent", "any2proxy")
		response, err := client.Do(request)
		if err != nil {
			return release{}, err
		}
		if response.StatusCode != http.StatusOK {
			_ = response.Body.Close()
			last = fmt.Errorf("cores: release %s: status %d", version, response.StatusCode)
			continue
		}
		var rel release
		err = json.NewDecoder(response.Body).Decode(&rel)
		_ = response.Body.Close()
		if err != nil {
			return release{}, err
		}
		return rel, nil
	}
	if last == nil {
		last = fmt.Errorf("cores: release not found")
	}
	return release{}, last
}

func selectAsset(id string, assets []asset, goos, goarch, filter string) (asset, error) {
	if id == "xray" && filter == "" {
		name := xrayAsset(goos, goarch)
		for _, item := range assets {
			if item.Name == name {
				return item, nil
			}
		}
		return asset{}, fmt.Errorf("cores: no xray asset for %s/%s", goos, goarch)
	}

	osTokens := []string{goos}
	if goos == "darwin" {
		osTokens = []string{"darwin", "macos"}
	}
	archTokens := []string{goarch}
	switch goarch {
	case "amd64":
		archTokens = []string{"amd64", "x86_64"}
	case "arm64":
		archTokens = []string{"arm64", "aarch64"}
	}

	var candidates []asset
	for _, item := range assets {
		name := strings.ToLower(item.Name)
		if filter != "" && !strings.Contains(name, strings.ToLower(filter)) {
			continue
		}
		if !hasAny(name, osTokens) || !hasAny(name, archTokens) {
			continue
		}
		if strings.Contains(name, "sha256") || strings.Contains(name, "dgst") || strings.Contains(name, ".sig") ||
			strings.Contains(name, ".deb") || strings.Contains(name, ".rpm") || strings.Contains(name, ".apk") ||
			strings.Contains(name, ".pkg.tar") || strings.Contains(name, "legacy") {
			continue
		}
		if !strings.HasSuffix(name, ".zip") && !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".gz") {
			continue
		}
		if goarch == "amd64" && strings.Contains(name, "arm64") {
			continue
		}
		if goarch == "arm64" && strings.Contains(name, "amd64") {
			continue
		}
		candidates = append(candidates, item)
	}
	if len(candidates) == 0 {
		return asset{}, fmt.Errorf("cores: no %s asset for %s/%s", id, goos, goarch)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if len(candidates[i].Name) != len(candidates[j].Name) {
			return len(candidates[i].Name) < len(candidates[j].Name)
		}
		return candidates[i].Name < candidates[j].Name
	})
	return candidates[0], nil
}

func xrayAsset(goos, goarch string) string {
	switch goos + "/" + goarch {
	case "windows/amd64":
		return "Xray-windows-64.zip"
	case "windows/arm64":
		return "Xray-windows-arm64-v8a.zip"
	case "linux/amd64":
		return "Xray-linux-64.zip"
	case "linux/arm64":
		return "Xray-linux-arm64-v8a.zip"
	case "darwin/amd64":
		return "Xray-macos-64.zip"
	case "darwin/arm64":
		return "Xray-macos-arm64-v8a.zip"
	default:
		return ""
	}
}

func hasAny(name string, tokens []string) bool {
	for _, token := range tokens {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

func matchesBinary(name, base string) bool {
	candidate := strings.ToLower(filepath.Base(name))
	base = strings.ToLower(base)
	if candidate == base {
		return true
	}
	stem := strings.TrimSuffix(base, ".exe")
	if !strings.HasPrefix(candidate, stem) {
		return false
	}
	if strings.HasSuffix(candidate, ".exe") {
		return true
	}
	ext := filepath.Ext(candidate)
	return ext == "" || ext == ".bin"
}

func download(ctx context.Context, client *http.Client, item asset) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "any2proxy")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cores: download status %d", response.StatusCode)
	}

	file, err := os.CreateTemp("", "any2proxy-download-*")
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), response.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}

	if digest := strings.TrimPrefix(item.Digest, "sha256:"); digest != "" {
		sum := hex.EncodeToString(hash.Sum(nil))
		if sum != digest {
			_ = os.Remove(file.Name())
			return "", fmt.Errorf("cores: checksum mismatch")
		}
	}
	return file.Name(), nil
}

func extractBinary(archivePath, assetName, binaryName, goos, dir string) (string, error) {
	base := binaryName
	if goos == "windows" {
		base += ".exe"
	}
	switch {
	case strings.HasSuffix(strings.ToLower(assetName), ".zip"):
		return extractZip(archivePath, base, dir)
	case strings.HasSuffix(strings.ToLower(assetName), ".tar.gz"):
		return extractTarGz(archivePath, base, dir)
	case strings.HasSuffix(strings.ToLower(assetName), ".gz"):
		return extractGz(archivePath, base, dir)
	default:
		return "", fmt.Errorf("cores: unsupported archive %q", assetName)
	}
}

func extractZip(path, base, dir string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if !matchesBinary(file.Name, base) {
			continue
		}
		source, err := file.Open()
		if err != nil {
			return "", err
		}
		defer source.Close()
		return writeExtracted(source, base, dir)
	}
	return "", fmt.Errorf("cores: %s not found in archive", base)
}

func extractTarGz(path, base, dir string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if !matchesBinary(header.Name, base) {
			continue
		}
		return writeExtracted(tarReader, base, dir)
	}
	return "", fmt.Errorf("cores: %s not found in archive", base)
}

func extractGz(path, base, dir string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	return writeExtracted(reader, base, dir)
}

func writeExtracted(source io.Reader, base, dir string) (string, error) {
	target := filepath.Join(dir, base)
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(file, source); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return target, nil
}
