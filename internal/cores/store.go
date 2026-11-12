package cores

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var validID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type Entry struct {
	ID         string    `json:"id"`
	Version    string    `json:"version,omitempty"`
	SHA256     string    `json:"sha256"`
	Path       string    `json:"path"`
	ImportedAt time.Time `json:"imported_at"`
}

type Store struct {
	Path    string
	Dir     string
	entries []Entry
}

func DefaultPath() string {
	home := os.Getenv("A2P_HOME")
	if home == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			base = os.TempDir()
		}
		home = filepath.Join(base, "any2proxy")
	}
	return filepath.Join(home, "config", "cores.json")
}

func DefaultDir() string {
	home := os.Getenv("A2P_HOME")
	if home == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			base = os.TempDir()
		}
		home = filepath.Join(base, "any2proxy")
	}
	return filepath.Join(home, "cores")
}

func Open() (*Store, error) {
	return Load(DefaultPath(), DefaultDir())
}

func Load(path, dir string) (*Store, error) {
	store := &Store{Path: path, Dir: dir}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("cores: %w", err)
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("cores: %w", err)
	}
	store.entries = entries
	return store, nil
}

func (s *Store) Import(id, source, version string) (Entry, error) {
	if !validID.MatchString(id) {
		return Entry{}, fmt.Errorf("cores: invalid id %q", id)
	}
	info, err := os.Stat(source)
	if err != nil || !info.Mode().IsRegular() {
		return Entry{}, fmt.Errorf("cores: source is not a file")
	}

	targetDir := filepath.Join(s.Dir, id)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Entry{}, fmt.Errorf("cores: %w", err)
	}
	target := filepath.Join(targetDir, filepath.Base(source))
	if err := copyFile(source, target); err != nil {
		return Entry{}, err
	}
	sum, err := hashFile(target)
	if err != nil {
		return Entry{}, err
	}

	entry := Entry{
		ID:         id,
		Version:    version,
		SHA256:     sum,
		Path:       target,
		ImportedAt: time.Now().UTC(),
	}
	s.entries = replace(s.entries, entry)
	if err := s.save(); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

func (s *Store) List() []Entry {
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *Store) Get(id string) (Entry, error) {
	for _, entry := range s.entries {
		if entry.ID == id {
			return entry, nil
		}
	}
	return Entry{}, fmt.Errorf("cores: %q not found", id)
}

func (s *Store) Remove(id string) error {
	entry, err := s.Get(id)
	if err != nil {
		return err
	}
	s.entries = remove(s.entries, id)
	if err := os.Remove(entry.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cores: %w", err)
	}
	return s.save()
}

func (s *Store) Verify(id string) error {
	entry, err := s.Get(id)
	if err != nil {
		return err
	}
	sum, err := hashFile(entry.Path)
	if err != nil {
		return err
	}
	if sum != entry.SHA256 {
		return fmt.Errorf("cores: %s checksum mismatch", id)
	}
	return nil
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("cores: %w", err)
	}
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("cores: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.Path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("cores: %w", err)
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("cores: %w", err)
	}
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("cores: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("cores: %w", err)
	}
	return os.Rename(temp.Name(), s.Path)
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("cores: %w", err)
	}
	defer in.Close()

	temp, err := os.CreateTemp(filepath.Dir(target), ".tmp-*")
	if err != nil {
		return fmt.Errorf("cores: %w", err)
	}
	defer os.Remove(temp.Name())
	if _, err := io.Copy(temp, in); err != nil {
		_ = temp.Close()
		return fmt.Errorf("cores: %w", err)
	}
	if err := temp.Chmod(0o755); err != nil {
		_ = temp.Close()
		return fmt.Errorf("cores: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("cores: %w", err)
	}
	return os.Rename(temp.Name(), target)
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cores: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("cores: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func replace(entries []Entry, entry Entry) []Entry {
	for i := range entries {
		if entries[i].ID == entry.ID {
			entries[i] = entry
			return entries
		}
	}
	return append(entries, entry)
}

func remove(entries []Entry, id string) []Entry {
	out := entries[:0]
	for _, entry := range entries {
		if entry.ID != id {
			out = append(out, entry)
		}
	}
	return out
}
