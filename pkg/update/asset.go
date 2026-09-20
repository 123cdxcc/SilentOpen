package update

import "strings"

// Asset returns the release asset built for goos/goarch, or nil when no asset
// looks like a build for that platform.
//
// Matching is deliberately tolerant, because asset names are chosen by the
// release pipeline rather than by this package: a name qualifies when it
// contains the platform word and the architecture word, separated from the rest
// of the name by anything that is not a letter or a digit. Both
// "SilentOpen-1.2.0-linux-amd64.tar.gz" and "gh_2.101.0_linux_amd64.tar.gz" are
// therefore recognised, and so is the underscored "x86_64". An
// architecture-exact asset wins over a universal one.
//
// Two assets can still score the same — a release that attaches both an
// installer and a bare build for one platform. The first one wins, and GitHub
// returns assets in upload order, so the pipeline decides by uploading its
// preferred build first. Guessing further here (preferring ".exe" over ".msi",
// say) would encode one author's naming into the package. A release whose names
// this package does not recognise yields nil instead of a guess.
func (r *Release) Asset(goos, goarch string) *Asset {
	if r == nil {
		return nil
	}
	platform, ok := platformTokens(goos, goarch)
	if !ok {
		return nil
	}
	var best *Asset
	bestScore := 0
	for i := range r.Assets {
		asset := &r.Assets[i]
		if !isPayload(asset.Name) {
			continue
		}
		name := strings.ToLower(asset.Name)
		score := 0
		if hasAnyToken(name, platform.osTokens) {
			score += 2
		} else {
			continue
		}
		switch {
		case hasAnyToken(name, platform.archTokens):
			score += 2
		case hasAnyToken(name, platform.universalTokens):
			score++
		default:
			continue
		}
		if score > bestScore {
			best, bestScore = asset, score
		}
	}
	return best
}

// platformNameTokens describes what one GOOS/GOARCH pair is called in asset names.
type platformNameTokens struct {
	osTokens        []string
	archTokens      []string
	universalTokens []string
}

// platformTokens returns the asset-name tokens that identify goos/goarch. An
// unsupported pair reports false, which makes Asset report "nothing suitable"
// instead of matching a name by accident.
func platformTokens(goos, goarch string) (platformNameTokens, bool) {
	var platform platformNameTokens
	switch goos {
	case "darwin":
		platform.osTokens = []string{"darwin", "macos", "mac", "osx"}
		// A Wails macOS build is one lipo'd bundle that serves either chip.
		platform.universalTokens = []string{"universal"}
	case "windows":
		platform.osTokens = []string{"windows", "win"}
	case "linux":
		platform.osTokens = []string{"linux"}
	default:
		return platformNameTokens{}, false
	}
	switch goarch {
	case "amd64":
		platform.archTokens = []string{"amd64", "x86_64", "x64"}
	case "arm64":
		platform.archTokens = []string{"arm64", "aarch64"}
	default:
		return platformNameTokens{}, false
	}
	return platform, true
}

// nonPayloadSuffixes lists the companion files a release attaches next to its
// builds. They carry a platform word in their own name ("...-macos.sha256"), so
// they would otherwise be offered as a download.
var nonPayloadSuffixes = []string{
	".sha256", ".sha512", ".sha1", ".md5",
	".sig", ".asc", ".pem",
	".txt", ".json", ".yml", ".yaml",
	".blockmap", ".zsync",
}

// isPayload reports whether an asset name looks like a build rather than one of
// the companion files listed in nonPayloadSuffixes.
func isPayload(name string) bool {
	lowered := strings.ToLower(name)
	for _, suffix := range nonPayloadSuffixes {
		if strings.HasSuffix(lowered, suffix) {
			return false
		}
	}
	return true
}

// hasAnyToken reports whether name contains one of wanted as a whole word.
func hasAnyToken(name string, wanted []string) bool {
	for _, token := range wanted {
		if hasToken(name, token) {
			return true
		}
	}
	return false
}

// hasToken reports whether name contains token with a word boundary on both
// sides. Matching whole words rather than plain substrings is what keeps
// "macOS" from being found by the short token "mac" and "arm64" from being
// confused with "armv6"; treating "_" as a boundary as well is what makes
// "linux_amd64" and "x86_64" both readable.
func hasToken(name, token string) bool {
	for offset := 0; ; {
		found := strings.Index(name[offset:], token)
		if found < 0 {
			return false
		}
		start := offset + found
		if isWordBoundary(name, start-1) && isWordBoundary(name, start+len(token)) {
			return true
		}
		offset = start + 1
	}
}

// isWordBoundary reports whether position is outside name or holds a character
// that is neither a letter nor a digit.
func isWordBoundary(name string, position int) bool {
	if position < 0 || position >= len(name) {
		return true
	}
	character := name[position]
	alphanumeric := (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9')
	return !alphanumeric
}
