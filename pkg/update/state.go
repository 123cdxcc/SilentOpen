package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// StateVersion is the schema version of the persisted state. It is written into
// every file so a future change of shape can reject an old file instead of
// misreading it.
const StateVersion = 1

// State is what survives between runs: when the last check happened, what it
// found, and which version the user asked not to be told about again.
type State struct {
	SchemaVersion   int    `json:"schemaVersion"`
	CheckedAt       int64  `json:"checkedAt"`
	LatestVersion   string `json:"latestVersion,omitempty"`
	LatestPageURL   string `json:"latestPageUrl,omitempty"`
	LatestAssetName string `json:"latestAssetName,omitempty"`
	Notes           string `json:"notes,omitempty"`
	SkippedVersion  string `json:"skippedVersion,omitempty"`
}

// Store persists State between runs. The interface exists so tests, and callers
// that do not want a file on disk at all, can supply their own.
type Store interface {
	// Load returns the persisted state, or the zero State when nothing usable
	// is stored. Reading deliberately has no error result: "never checked" and
	// "the file went away" are both normal, and both are answered the same way
	// — by asking the source again.
	Load() State

	// Save replaces the persisted state.
	Save(State) error
}

// FileStore keeps State in one JSON file.
type FileStore struct{ path string }

// NewFileStore returns a Store reading and writing path. An empty path — which
// DefaultStatePath reports when no user config directory can be determined —
// yields a store that has no state and refuses to save, so a caller that
// tolerates the write failure still gets a working, always-fresh check.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

// DefaultStatePath returns the per-user location of the state file:
// <user config dir>/SilentOpen/update.json.
func DefaultStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("无法确定用户配置目录：%w", err)
	}
	return filepath.Join(dir, "SilentOpen", "update.json"), nil
}

// Load implements Store. A missing, unreadable, malformed, or future-schema
// file all report the zero State: the only consequence is one extra check.
func (s *FileStore) Load() State {
	if s == nil || s.path == "" {
		return State{}
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return State{}
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil || state.SchemaVersion != StateVersion {
		return State{}
	}
	return state
}

// Save implements Store. The payload is written to a temporary file in the
// target directory and then renamed over the target, so an interrupted save
// never leaves a half-written file behind. Failures carry their cause
// unnormalised; the caller that knows the declared errors wraps them.
func (s *FileStore) Save(state State) error {
	if s == nil || s.path == "" {
		return errors.New("状态文件路径为空")
	}
	state.SchemaVersion = StateVersion
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, "update-*.json")
	if err != nil {
		return err
	}
	// A successful rename moves the file away, which makes this a no-op.
	defer os.Remove(temp.Name())
	if _, err := temp.Write(payload); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), s.path)
}
