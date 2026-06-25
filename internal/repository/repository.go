package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Store reads and writes app data as JSON files in a single directory.
// state.json is the source of truth; history.jsonl is an append-only event log.
type Store struct {
	dir string
}

// New returns a Store rooted at dir and makes sure the directory exists.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) statePath() string  { return filepath.Join(s.dir, "state.json") }
func (s *Store) configPath() string { return filepath.Join(s.dir, "config.json") }
func (s *Store) logPath() string    { return filepath.Join(s.dir, "history.jsonl") }

// LoadState reads state.json. If the file doesn't exist yet (first run), it writes a
// default state and returns it. Missing fields in an existing file keep their
// defaults, so adding fields in a new version won't break old saves.
func (s *Store) LoadState(now int64) (*State, error) {
	data, err := os.ReadFile(s.statePath())

	if errors.Is(err, os.ErrNotExist) {
		def := DefaultState(now)
		if err := s.SaveState(def); err != nil {
			return nil, err
		}
		return def, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}

	// Start from defaults, then overlay what's in the file. Fields absent from
	// the JSON keep their default values instead of becoming zero/broken.
	state := DefaultState(now)
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return state, nil
}

// SaveState writes state.json atomically: it writes to a temp file in the same
// directory, then renames it over the target. A crash mid-write leaves the old
// file intact — never a half-written one. This is what protects the bank.
func (s *Store) SaveState(state *State) error {
	state.SchemaVersion = StateSchemaVersion

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}

	// Temp file MUST be in the same directory so the rename stays on one
	// filesystem and is therefore atomic.
	tmp, err := os.CreateTemp(s.dir, "state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	// If anything below fails, don't leave the temp file lying around.
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	// Flush to disk before the rename so the data is really persisted.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	if err := os.Rename(tmpName, s.statePath()); err != nil {
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}

// AppendLog adds one event line to history.jsonl. The file is created on first
// write. Append of a small line is effectively atomic, so no temp dance needed.
func (s *Store) AppendLog(entry LogEntry) error {
	f, err := os.OpenFile(s.logPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	// Encoder.Encode appends a trailing newline, which is exactly the JSONL format.
	if err := json.NewEncoder(f).Encode(entry); err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	return nil
}

// ReadLog streams every event from history.jsonl, calling fn for each. Reading
// line by line keeps memory flat even as the log grows. A missing file is not
// an error — it just means no events yet.
func (s *Store) ReadLog(fn func(LogEntry) error) error {
	f, err := os.Open(s.logPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e LogEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return fmt.Errorf("parse log line: %w", err)
		}
		if err := fn(e); err != nil {
			return err
		}
	}
	return sc.Err()
}

// LoadConfig reads config.json. If the file doesn't exist yet (first run), it
// writes a default config and returns it. Missing fields in an existing file
// keep their defaults, so adding fields in a new version won't break old saves.
func (s *Store) LoadConfig() (*Config, error) {
	data, err := os.ReadFile(s.configPath())

	if errors.Is(err, os.ErrNotExist) {
		def := DefaultConfig()
		if err := s.SaveConfig(def); err != nil {
			return nil, err
		}
		return def, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Start from defaults, then overlay what's in the file. Fields absent from
	// the JSON keep their default values instead of becoming zero/broken.
	def := DefaultConfig()
	if err := json.Unmarshal(data, def); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return def, nil
}

// SaveConfig writes config.json atomically: it writes to a temp file in the same
// directory, then renames it over the target. A crash mid-write leaves the old
// file intact — never a half-written one.
func (s *Store) SaveConfig(config *Config) error {
	config.SchemaVersion = ConfigSchemaVersion

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	// Temp file MUST be in the same directory so the rename stays on one
	// filesystem and is therefore atomic.
	tmp, err := os.CreateTemp(s.dir, "config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	// If anything below fails, don't leave the temp file lying around.
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	// Flush to disk before the rename so the data is really persisted.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	if err := os.Rename(tmpName, s.configPath()); err != nil {
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}
