package icon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/123cdxcc/SilentOpen/pkg/icondata"
)

// platformProvider resolves icons from the freedesktop desktop entries.
type platformProvider struct{}

// NewPlatformProvider returns the Linux Provider.
func NewPlatformProvider() Provider { return platformProvider{} }

// Icon matches executable against installed .desktop entries and returns the
// icon of the entry that launches it.
func (platformProvider) Icon(ctx context.Context, executable string) string {
	canonical, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return ""
	}
	roots := desktopRoots()
	var found string
	for _, root := range roots {
		entries, _ := os.ReadDir(filepath.Join(root, "applications"))
		for _, entry := range entries {
			if ctx.Err() != nil {
				return ""
			}
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".desktop" {
				continue
			}
			data, err := icondata.ReadFile(filepath.Join(root, "applications", entry.Name()))
			if err != nil {
				continue
			}
			command, iconName := desktopFields(string(data))
			candidate := desktopExecutable(command)
			if candidate == "" || iconName == "" {
				continue
			}
			if !filepath.IsAbs(candidate) {
				candidate, _ = exec.LookPath(candidate)
			}
			resolved, err := filepath.EvalSymlinks(candidate)
			if err == nil && resolved == canonical {
				if value := findDesktopIcon(roots, iconName); value != "" {
					if found != "" && value != found {
						return ""
					}
					found = value
				}
			}
		}
	}
	return found
}

func desktopRoots() []string {
	home := os.Getenv("XDG_DATA_HOME")
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".local", "share")
	}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if dirs == "" {
		dirs = "/usr/local/share:/usr/share"
	}
	roots := []string{home}
	for _, dir := range filepath.SplitList(dirs) {
		if filepath.IsAbs(dir) {
			roots = append(roots, dir)
		}
	}
	return roots
}

func desktopFields(data string) (command, iconName string) {
	inEntry := false
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
		} else if inEntry {
			key, value, _ := strings.Cut(line, "=")
			switch key {
			case "Exec":
				command = value
			case "Icon":
				iconName = value
			}
		}
	}
	return
}

// desktopExecutable extracts the program of an Exec line, or "" when the line
// needs a shell wrapper that this matcher does not resolve.
func desktopExecutable(command string) string {
	var token strings.Builder
	quoted, escaped := false, false
	command = strings.TrimSpace(command)
	for i, char := range command {
		if escaped {
			token.WriteRune(char)
			escaped = false
		} else if char == '\\' {
			escaped = true
		} else if char == '"' {
			quoted = !quoted
		} else if !quoted && (char == ' ' || char == '\t') {
			for _, arg := range strings.Fields(command[i:]) {
				switch arg {
				case "%f", "%F", "%u", "%U", "%i", "%c", "%k":
				default:
					return ""
				}
			}
			break
		} else {
			token.WriteRune(char)
		}
	}
	if quoted || escaped || strings.Contains(token.String(), "%") {
		return ""
	}
	return token.String()
}

// findDesktopIcon searches the icon themes for a desktop entry's icon name.
func findDesktopIcon(roots []string, name string) string {
	if filepath.IsAbs(name) {
		return icondata.FileURL(name)
	}
	if filepath.Base(name) != name {
		return ""
	}
	names := []string{name}
	if filepath.Ext(name) == "" {
		names = []string{name + ".png", name + ".svg"}
	}
	for _, root := range roots {
		for _, size := range []string{"64x64", "48x48", "32x32", "24x24", "16x16", "128x128", "256x256", "scalable"} {
			for _, file := range names {
				if iconURL := icondata.FileURL(filepath.Join(root, "icons", "hicolor", size, "apps", file)); iconURL != "" {
					return iconURL
				}
			}
		}
		for _, file := range names {
			if iconURL := icondata.FileURL(filepath.Join(root, "pixmaps", file)); iconURL != "" {
				return iconURL
			}
		}
	}
	return ""
}
