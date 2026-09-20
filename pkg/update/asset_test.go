package update

import "testing"

func releaseWith(names ...string) *Release {
	assets := make([]Asset, 0, len(names))
	for _, name := range names {
		assets = append(assets, Asset{Name: name, URL: "https://example.test/" + name, Size: 1})
	}
	return &Release{Version: "1.2.0", Assets: assets}
}

func TestAssetPicksTheBuildForEachPlatform(t *testing.T) {
	release := releaseWith(
		"SilentOpen-1.2.0-macos-universal.zip",
		"SilentOpen-1.2.0-windows-amd64.exe",
		"SilentOpen-1.2.0-linux-amd64.tar.gz",
		"SilentOpen-1.2.0-linux-arm64.tar.gz",
		"checksums.txt",
	)
	cases := []struct {
		goos, goarch string
		want         string
	}{
		{"darwin", "arm64", "SilentOpen-1.2.0-macos-universal.zip"},
		{"darwin", "amd64", "SilentOpen-1.2.0-macos-universal.zip"},
		{"windows", "amd64", "SilentOpen-1.2.0-windows-amd64.exe"},
		{"linux", "amd64", "SilentOpen-1.2.0-linux-amd64.tar.gz"},
		{"linux", "arm64", "SilentOpen-1.2.0-linux-arm64.tar.gz"},
	}
	for _, testCase := range cases {
		asset := release.Asset(testCase.goos, testCase.goarch)
		if asset == nil {
			t.Errorf("Asset(%s/%s) = nil, want %s", testCase.goos, testCase.goarch, testCase.want)
			continue
		}
		if asset.Name != testCase.want {
			t.Errorf("Asset(%s/%s) = %q, want %q", testCase.goos, testCase.goarch, asset.Name, testCase.want)
		}
	}
}

func TestAssetPrefersAnExactArchitectureOverUniversal(t *testing.T) {
	release := releaseWith(
		"SilentOpen-1.2.0-macos-universal.zip",
		"SilentOpen-1.2.0-macos-arm64.zip",
	)
	if got := release.Asset("darwin", "arm64"); got == nil || got.Name != "SilentOpen-1.2.0-macos-arm64.zip" {
		t.Fatalf("got %v, want the arm64 build", got)
	}
}

func TestAssetAcceptsUnderscoredArchitectureNames(t *testing.T) {
	release := releaseWith("SilentOpen-1.2.0-linux-x86_64.tar.gz")
	if got := release.Asset("linux", "amd64"); got == nil || got.Name != "SilentOpen-1.2.0-linux-x86_64.tar.gz" {
		t.Fatalf("got %v, want the x86_64 build", got)
	}
}

func TestAssetRefusesToGuess(t *testing.T) {
	cases := map[string]*Release{
		"no assets":            releaseWith(),
		"platform only":        releaseWith("SilentOpen-macos.zip"),
		"architecture only":    releaseWith("SilentOpen-amd64.zip"),
		"companion files only": releaseWith("SilentOpen-1.2.0-macos-universal.zip.sha256", "SilentOpen-1.2.0-macos-universal.txt"),
		"another platform":     releaseWith("SilentOpen-1.2.0-windows-amd64.exe"),
		"nil release":          nil,
	}
	for name, release := range cases {
		if got := release.Asset("darwin", "arm64"); got != nil {
			t.Errorf("%s: got %q, want nil", name, got.Name)
		}
	}
}

// Release pipelines name assets in their own style. This list mirrors what
// goreleaser publishes (mixed case, underscores, checksum files next to the
// builds), so the matcher is locked against a naming convention this project
// does not control.
func TestAssetHandlesGoReleaserStyleNames(t *testing.T) {
	release := releaseWith(
		"gh_2.101.0_checksums.txt",
		"gh_2.101.0_linux_386.tar.gz",
		"gh_2.101.0_linux_amd64.tar.gz",
		"gh_2.101.0_linux_arm64.tar.gz",
		"gh_2.101.0_macOS_amd64.zip",
		"gh_2.101.0_macOS_arm64.zip",
		"gh_2.101.0_macOS_universal.pkg",
		"gh_2.101.0_windows_amd64.zip",
	)
	cases := []struct {
		goos, goarch string
		want         string
	}{
		{"darwin", "arm64", "gh_2.101.0_macOS_arm64.zip"},
		{"darwin", "amd64", "gh_2.101.0_macOS_amd64.zip"},
		{"linux", "amd64", "gh_2.101.0_linux_amd64.tar.gz"},
		{"linux", "arm64", "gh_2.101.0_linux_arm64.tar.gz"},
		{"windows", "amd64", "gh_2.101.0_windows_amd64.zip"},
	}
	for _, testCase := range cases {
		asset := release.Asset(testCase.goos, testCase.goarch)
		if asset == nil {
			t.Errorf("Asset(%s/%s) = nil, want %s", testCase.goos, testCase.goarch, testCase.want)
			continue
		}
		if asset.Name != testCase.want {
			t.Errorf("Asset(%s/%s) = %q, want %q", testCase.goos, testCase.goarch, asset.Name, testCase.want)
		}
	}
	// A 32-bit build must never be offered to a 64-bit machine.
	if asset := release.Asset("linux", "amd64"); asset != nil && asset.Name == "gh_2.101.0_linux_386.tar.gz" {
		t.Error("Asset(linux/amd64) picked the 386 build")
	}
}

func TestAssetIgnoresAnUnsupportedTarget(t *testing.T) {
	release := releaseWith("SilentOpen-1.2.0-freebsd-amd64.tar.gz")
	if got := release.Asset("freebsd", "amd64"); got != nil {
		t.Fatalf("got %q, want nil", got.Name)
	}
	if got := release.Asset("darwin", "riscv64"); got != nil {
		t.Fatalf("got %q, want nil", got.Name)
	}
}
