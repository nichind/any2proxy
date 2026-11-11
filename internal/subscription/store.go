package subscription

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nichind/any2proxy/internal/secretstore"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type Entry struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	AddedAt     time.Time `json:"added_at"`
	LastFetched time.Time `json:"last_fetched,omitempty"`
	LastStatus  string    `json:"last_status,omitempty"`
	Nodes       int       `json:"nodes,omitempty"`
}

type file struct {
	Version       int     `json:"version"`
	Subscriptions []Entry `json:"subscriptions"`
}

type Store struct {
	Path    string
	entries []Entry
}

func DefaultPath() string {
	if home := os.Getenv("A2P_HOME"); home != "" {
		return filepath.Join(home, "config", "subscriptions.json")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "any2proxy", "subscriptions.json")
}

func Load(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath()
	}
	store := &Store{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("subscription: %w", err)
	}
	data, err = secretstore.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("subscription: %w", err)
	}
	var doc file
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("subscription: %w", err)
	}
	store.entries = doc.Subscriptions
	return store, nil
}

func (s *Store) Add(name, rawURL string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("subscription: invalid name %q", name)
	}
	if err := validURL(rawURL); err != nil {
		return err
	}
	for _, entry := range s.entries {
		if entry.Name == name {
			return fmt.Errorf("subscription: name %q exists", name)
		}
	}
	s.entries = append(s.entries, Entry{
		Name:    name,
		URL:     rawURL,
		AddedAt: time.Now().UTC(),
	})
	return s.save()
}

func (s *Store) Remove(name string) error {
	for i, entry := range s.entries {
		if entry.Name == name {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("subscription: %q not found", name)
}

func (s *Store) Get(name string) (Entry, error) {
	for _, entry := range s.entries {
		if entry.Name == name {
			return entry, nil
		}
	}
	return Entry{}, fmt.Errorf("subscription: %q not found", name)
}

func (s *Store) List() []Entry {
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *Store) Update(name string, nodes int, status string) error {
	for i := range s.entries {
		if s.entries[i].Name == name {
			s.entries[i].Nodes = nodes
			s.entries[i].LastStatus = status
			if status == "ok" {
				s.entries[i].LastFetched = time.Now().UTC()
			}
			return s.save()
		}
	}
	return fmt.Errorf("subscription: %q not found", name)
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("subscription: %w", err)
	}
	data, err := json.MarshalIndent(file{Version: 1, Subscriptions: s.entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("subscription: %w", err)
	}
	data, err = secretstore.Encode(data)
	if err != nil {
		return fmt.Errorf("subscription: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.Path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("subscription: %w", err)
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("subscription: %w", err)
	}
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("subscription: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("subscription: %w", err)
	}
	if err := os.Rename(temp.Name(), s.Path); err != nil {
		return fmt.Errorf("subscription: %w", err)
	}
	return nil
}

func validURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("subscription: invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("subscription: url must be http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("subscription: url has no host")
	}
	return nil
}

func Host(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}
