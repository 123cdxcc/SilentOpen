package main

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/123cdxcc/SilentOpen/pkg/update"
)

// releaseRepository is the GitHub repository (owner/repo) that publishes this
// application and therefore the one a check reads. It stays empty until the
// project is published, and a build without it says so instead of pretending to
// have checked. The release pipeline overrides it per build with
// -ldflags "-X main.releaseRepository=<owner>/<repo>".
var releaseRepository = ""

// updateCheckTimeout bounds one check triggered from the UI. It covers the work
// around the request as well, since a bound method should always return.
const updateCheckTimeout = 20 * time.Second

// ErrUnsafeReleaseURL reports a link this application refuses to open.
var ErrUnsafeReleaseURL = errors.New("只允许打开 GitHub 上的链接")

// newUpdateChecker wires the GitHub source to the per-user state file. When no
// config directory can be determined the feature still works; only remembering
// the answer is lost, so the checker may ask again next start.
func newUpdateChecker() *update.Checker {
	owner, repo, _ := strings.Cut(releaseRepository, "/")
	source := update.NewGitHub(owner, repo).WithUserAgent("SilentOpen/" + version)
	statePath, err := update.DefaultStatePath()
	if err != nil {
		statePath = ""
	}
	return update.NewChecker(source, update.NewFileStore(statePath), version)
}

// CheckUpdate reports whether a release newer than the running build is
// published. force asks even when the stored answer is still fresh. A repository
// without any release answers "nothing newer" instead of failing. The error
// message is meant to be shown to the user as-is, so a background check should
// swallow it while a check the user asked for should display it.
func (a *App) CheckUpdate(force bool) (*update.Result, error) {
	ctx, cancel := context.WithTimeout(a.applicationContext(), updateCheckTimeout)
	defer cancel()
	// A nil checker is nil-safe: it reports the repository as unconfigured.
	return a.updates.Check(ctx, force)
}

// SkipVersion remembers a version the user does not want to be told about
// again. An empty version forgets the previous choice.
func (a *App) SkipVersion(target string) error {
	return a.updates.Skip(target)
}

// GetVersion returns the version of the running build, for display.
func (a *App) GetVersion() string {
	return version
}

// OpenReleasePage opens a release page in the system browser. Only https links
// on github.com are accepted: the address comes from a network response, and a
// bound method must not become a way to open an arbitrary page.
func (a *App) OpenReleasePage(target string) error {
	if !isGitHubURL(target) {
		return ErrUnsafeReleaseURL
	}
	runtime.BrowserOpenURL(a.applicationContext(), target)
	return nil
}

// isGitHubURL reports whether target is an https URL on github.com. Asset
// downloads are served from github.com and redirect to a CDN, which the browser
// follows on its own, so no other host needs to be accepted.
func isGitHubURL(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "github.com", "www.github.com":
		return true
	default:
		return false
	}
}
