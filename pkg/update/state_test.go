package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStoreRoundTripsState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "update.json")
	store := NewFileStore(path)
	// Saving into a directory that does not exist yet must create it: the file
	// lives under the user config directory, which a fresh install lacks.
	want := State{
		CheckedAt:       1735689600,
		LatestVersion:   "1.2.0",
		LatestPageURL:   "https://github.com/example/SilentOpen/releases/tag/v1.2.0",
		LatestAssetName: "SilentOpen-1.2.0-macos-universal.zip",
		Notes:           "修好了两个问题。",
		SkippedVersion:  "1.1.0",
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save returned %v", err)
	}
	want.SchemaVersion = StateVersion
	if got := store.Load(); got != want {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir returned %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "update.json" {
		t.Fatalf("the directory holds %v, want only update.json", entries)
	}
}

func TestFileStoreDegradesInsteadOfFailing(t *testing.T) {
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.json")
	if err := os.WriteFile(malformed, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("WriteFile returned %v", err)
	}
	stale := filepath.Join(dir, "stale.json")
	if err := os.WriteFile(stale, []byte(`{"schemaVersion":0,"latestVersion":"9.9.9"}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned %v", err)
	}
	cases := map[string]*FileStore{
		"missing file":   NewFileStore(filepath.Join(dir, "absent.json")),
		"malformed file": NewFileStore(malformed),
		"old schema":     NewFileStore(stale),
		"empty path":     NewFileStore(""),
		"nil store":      nil,
	}
	for name, store := range cases {
		if got := store.Load(); got != (State{}) {
			t.Errorf("%s: Load() = %+v, want the zero State", name, got)
		}
	}
}

func TestFileStoreRefusesToSaveWithoutAPath(t *testing.T) {
	for name, store := range map[string]*FileStore{"empty path": NewFileStore(""), "nil store": nil} {
		if err := store.Save(State{CheckedAt: 1}); err == nil {
			t.Errorf("%s: Save reported success without a path", name)
		}
	}
}

func TestFileStoreSaveReplacesThePreviousFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.json")
	store := NewFileStore(path)
	if err := store.Save(State{CheckedAt: 1, LatestVersion: "1.0.0", SkippedVersion: "1.0.0"}); err != nil {
		t.Fatalf("Save returned %v", err)
	}
	// A later check that finds nothing must clear the remembered release while
	// keeping the version the user chose to ignore.
	if err := store.Save(State{CheckedAt: 2, SkippedVersion: "1.0.0"}); err != nil {
		t.Fatalf("Save returned %v", err)
	}
	got := store.Load()
	if got.CheckedAt != 2 || got.LatestVersion != "" || got.LatestPageURL != "" || got.SkippedVersion != "1.0.0" {
		t.Fatalf("Load() = %+v, want the replacement state", got)
	}
}

func TestDefaultStatePathLivesUnderTheUserConfigDirectory(t *testing.T) {
	path, err := DefaultStatePath()
	if err != nil {
		t.Fatalf("DefaultStatePath returned %v", err)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir returned %v", err)
	}
	if want := filepath.Join(dir, "SilentOpen", "update.json"); path != want {
		t.Fatalf("DefaultStatePath() = %q, want %q", path, want)
	}
	if !strings.HasSuffix(path, "update.json") {
		t.Fatalf("DefaultStatePath() = %q, want a JSON file", path)
	}
}
