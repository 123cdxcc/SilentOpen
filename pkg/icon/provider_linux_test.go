package icon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopExecutable(t *testing.T) {
	for command, want := range map[string]string{
		`/usr/bin/editor %U`:        "/usr/bin/editor",
		`"/opt/My App/editor" %F`:   "/opt/My App/editor",
		`"/opt/unterminated`:        "",
		`/opt/%invalid/application`: "",
		`node /opt/app/server.js`:   "",
		`editor --profile app`:      "",
	} {
		if got := desktopExecutable(command); got != want {
			t.Fatalf("desktopExecutable(%q) = %q, want %q", command, got, want)
		}
	}
}

func TestDesktopIconMatchesExecutable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_DATA_DIRS", dir)
	if err := os.Mkdir(filepath.Join(dir, "applications"), 0700); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "editor")
	iconPath := filepath.Join(dir, "editor.svg")
	for path, data := range map[string]string{
		exe:  "",
		iconPath: `<svg xmlns="http://www.w3.org/2000/svg"/>`,
		filepath.Join(dir, "applications", "editor.desktop"): "[Desktop Entry]\nExec=" + exe + " %F\nIcon=" + iconPath + "\n[Desktop Action Other]\nIcon=wrong-icon\n",
	} {
		if err := os.WriteFile(path, []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	provider := NewPlatformProvider()
	if got := provider.Icon(context.Background(), exe); !strings.HasPrefix(got, "data:image/svg+xml;base64,") {
		t.Fatal("desktop icon matching the executable was not found")
	}
	other := filepath.Join(dir, "other")
	if err := os.WriteFile(other, nil, 0700); err != nil {
		t.Fatal(err)
	}
	if provider.Icon(context.Background(), other) != "" {
		t.Fatal("unrelated executable must not receive the desktop icon")
	}
	otherIcon := filepath.Join(dir, "other.svg")
	if err := os.WriteFile(otherIcon, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><path/></svg>`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applications", "other.desktop"), []byte("[Desktop Entry]\nExec="+exe+"\nIcon="+otherIcon), 0600); err != nil {
		t.Fatal(err)
	}
	if provider.Icon(context.Background(), exe) != "" {
		t.Fatal("ambiguous application mappings must not choose an arbitrary icon")
	}
}
