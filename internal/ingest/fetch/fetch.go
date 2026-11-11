package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type Options struct {
	CacheDir     string
	Timeout      time.Duration
	MaxBytes     int64
	NoCache      bool
	Offline      bool
	AllowPrivate bool
	UserAgent    string
}

type Meta struct {
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	FetchedAt    time.Time `json:"fetched_at"`
	Size         int64     `json:"size"`
}

type Fetcher struct {
	client *http.Client
	opts   Options
}

func New(opts Options) *Fetcher {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 10 << 20
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "any2proxy/0.1"
	}
	if opts.CacheDir == "" {
		opts.CacheDir = defaultCacheDir()
	}

	client := &http.Client{Timeout: opts.Timeout}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if !opts.AllowPrivate {
			if err := checkHost(req.URL); err != nil {
				return err
			}
		}
		return nil
	}

	return &Fetcher{client: client, opts: opts}
}

func (f *Fetcher) Get(ctx context.Context, rawURL string) ([]byte, Meta, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, Meta{}, fmt.Errorf("fetch: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, Meta{}, fmt.Errorf("fetch: unsupported scheme %q", parsed.Scheme)
	}

	hash := hashURL(rawURL)
	bodyPath := filepath.Join(f.opts.CacheDir, hash+".body")
	metaPath := filepath.Join(f.opts.CacheDir, hash+".json")

	if f.opts.Offline {
		body, err := os.ReadFile(bodyPath)
		if err != nil {
			return nil, Meta{}, fmt.Errorf("fetch: offline and no cache: %w", err)
		}
		return body, readMeta(metaPath), nil
	}

	if !f.opts.AllowPrivate {
		if err := checkHost(parsed); err != nil {
			return nil, Meta{}, err
		}
	}

	meta := readMeta(metaPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, Meta{}, fmt.Errorf("fetch: %w", err)
	}
	req.Header.Set("User-Agent", f.opts.UserAgent)
	if !f.opts.NoCache {
		if meta.ETag != "" {
			req.Header.Set("If-None-Match", meta.ETag)
		}
		if meta.LastModified != "" {
			req.Header.Set("If-Modified-Since", meta.LastModified)
		}
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, Meta{}, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		body, err := os.ReadFile(bodyPath)
		if err != nil {
			return nil, Meta{}, fmt.Errorf("fetch: cached body missing: %w", err)
		}
		return body, meta, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, Meta{}, fmt.Errorf("fetch: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, f.opts.MaxBytes+1))
	if err != nil {
		return nil, Meta{}, fmt.Errorf("fetch: %w", err)
	}
	if int64(len(body)) > f.opts.MaxBytes {
		return nil, Meta{}, fmt.Errorf("fetch: body exceeds %d bytes", f.opts.MaxBytes)
	}

	meta = Meta{
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		FetchedAt:    time.Now().UTC(),
		Size:         int64(len(body)),
	}
	if err := writeCache(f.opts.CacheDir, bodyPath, metaPath, body, meta); err != nil {
		return nil, Meta{}, err
	}
	return body, meta, nil
}

func defaultCacheDir() string {
	if home := os.Getenv("A2P_HOME"); home != "" {
		return filepath.Join(home, "cache", "subs")
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "any2proxy", "subs")
	}
	return filepath.Join(base, "any2proxy", "subs")
}

func hashURL(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(sum[:])
}

func readMeta(path string) Meta {
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return Meta{}
	}
	return meta
}

func writeCache(dir, bodyPath, metaPath string, body []byte, meta Meta) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	if err := writeFile(bodyPath, body); err != nil {
		return err
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return writeFile(metaPath, data)
}

func writeFile(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("fetch: %w", err)
	}
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("fetch: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	return nil
}

func checkHost(parsed *url.URL) error {
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("fetch: empty host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivate(ip) {
			return fmt.Errorf("fetch: private address blocked")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("fetch: resolve %s: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivate(ip) {
			return fmt.Errorf("fetch: private address blocked")
		}
	}
	return nil
}

func isPrivate(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
