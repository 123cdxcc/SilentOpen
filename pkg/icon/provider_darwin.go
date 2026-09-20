package icon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/123cdxcc/SilentOpen/pkg/icondata"
)

// platformProvider resolves icons from macOS application bundles.
type platformProvider struct{}

// NewPlatformProvider returns the macOS Provider.
func NewPlatformProvider() Provider { return platformProvider{} }

// Icon finds the icon of the .app bundle that owns executable.
func (platformProvider) Icon(ctx context.Context, executable string) string {
	// Helpers may live in nested bundles with no icon, so try their outer app too.
	for _, bundle := range appBundles(executable) {
		for _, key := range []string{"CFBundleIconFile", "CFBundleIconName"} {
			if ctx.Err() != nil {
				return ""
			}
			out, err := exec.CommandContext(ctx, "/usr/libexec/PlistBuddy", "-c", "Print :"+key, filepath.Join(bundle, "Contents", "Info.plist")).Output()
			name := strings.TrimSpace(string(out))
			if err != nil || name == "" || len(name) > 512 || filepath.Base(name) != name {
				continue
			}
			for _, suffix := range []string{"", ".icns", ".png"} {
				source := filepath.Join(bundle, "Contents", "Resources", name+suffix)
				if info, err := os.Stat(source); err != nil || !info.Mode().IsRegular() {
					continue
				}
				if icon := convertIcon(ctx, source); icon != "" {
					return icon
				}
			}
		}
	}
	return ""
}

func appBundles(executable string) []string {
	var bundles []string
	for dir := filepath.Dir(executable); ; dir = filepath.Dir(dir) {
		if strings.EqualFold(filepath.Ext(dir), ".app") {
			bundles = append(bundles, dir)
		}
		if parent := filepath.Dir(dir); parent == dir {
			return bundles
		}
	}
}

func convertIcon(ctx context.Context, source string) string {
	temp, err := os.MkdirTemp("", "silentopen-icon-")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(temp)
	output := filepath.Join(temp, "icon.png")
	if err := exec.CommandContext(ctx, "/usr/bin/sips", "-s", "format", "png", "-Z", "64", source, "--out", output).Run(); err != nil {
		return ""
	}
	return icondata.FileURL(output)
}
