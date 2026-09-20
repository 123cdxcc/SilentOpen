package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/123cdxcc/SilentOpen/pkg/update"
)

// stubSource answers with one canned release, or with a failure, and can
// observe the context it was given.
type stubSource struct {
	release *update.Release
	fail    error
	calls   int
}

func (s *stubSource) Latest(ctx context.Context) (*update.Release, error) {
	s.calls++
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w：%w", update.ErrUnreachable, err)
	}
	if s.fail != nil {
		return nil, s.fail
	}
	if s.release == nil {
		return nil, update.ErrNoRelease
	}
	return s.release, nil
}

// stubStore keeps the state in memory instead of in the user's config directory.
type stubStore struct {
	state update.State
	saves int
}

func (s *stubStore) Load() update.State { return s.state }

func (s *stubStore) Save(state update.State) error {
	s.saves++
	s.state = state
	return nil
}

func appWithUpdates(source *stubSource, store *stubStore, current string) *App {
	checker := update.NewChecker(source, store, current).WithPlatform("darwin", "arm64")
	return &App{updates: checker}
}

// released is the canned answer the stub source serves.
func released() *update.Release {
	return &update.Release{
		Version: "1.2.0",
		Notes:   "修好了两个问题。",
		PageURL: "https://github.com/example/SilentOpen/releases/tag/v1.2.0",
		Assets: []update.Asset{
			{Name: "SilentOpen-1.2.0-macos-universal.zip", URL: "https://github.com/example/SilentOpen/releases/download/v1.2.0/macos.zip"},
		},
	}
}

// A build that has no repository to check must say so rather than report that
// everything is up to date.
func TestCheckUpdateReportsAnUnconfiguredRepository(t *testing.T) {
	configured := releaseRepository
	releaseRepository = ""
	t.Cleanup(func() { releaseRepository = configured })

	result, err := NewApp().CheckUpdate(true)
	if !errors.Is(err, update.ErrRepoUnconfigured) {
		t.Fatalf("CheckUpdate returned %v, want ErrRepoUnconfigured", err)
	}
	if result != nil {
		t.Fatalf("CheckUpdate returned %+v together with an error", result)
	}
}

func TestCheckUpdateReturnsTheAnswer(t *testing.T) {
	source, store := &stubSource{release: released()}, &stubStore{}
	app := appWithUpdates(source, store, "1.0.0")

	result, err := app.CheckUpdate(true)
	if err != nil {
		t.Fatalf("CheckUpdate returned %v", err)
	}
	if result == nil {
		t.Fatal("CheckUpdate returned a nil result with a nil error")
	}
	if !result.UpdateAvailable || result.LatestVersion != "1.2.0" || result.CurrentVersion != "1.0.0" {
		t.Errorf("CheckUpdate returned %+v", result)
	}
	if result.AssetName != "SilentOpen-1.2.0-macos-universal.zip" {
		t.Errorf("CheckUpdate picked %q", result.AssetName)
	}
	if store.saves != 1 {
		t.Errorf("the answer was stored %d time(s), want 1", store.saves)
	}
	// A second, unforced call reuses the stored answer instead of asking again.
	if _, err := app.CheckUpdate(false); err != nil {
		t.Fatalf("CheckUpdate returned %v", err)
	}
	if source.calls != 1 {
		t.Errorf("the source was asked %d time(s), want 1", source.calls)
	}
}

func TestCheckUpdateFailsOnceTheApplicationShutDown(t *testing.T) {
	app := appWithUpdates(&stubSource{release: released()}, &stubStore{}, "1.0.0")
	app.startup(context.Background())
	app.shutdown(context.Background())

	result, err := app.CheckUpdate(true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("CheckUpdate returned %v, want the cancellation to stay reachable", err)
	}
	if result != nil {
		t.Fatalf("CheckUpdate returned %+v together with an error", result)
	}
}

func TestCheckUpdateReportsARepositoryWithoutReleases(t *testing.T) {
	app := appWithUpdates(&stubSource{}, &stubStore{}, "1.0.0")

	result, err := app.CheckUpdate(true)
	if err != nil {
		t.Fatalf("CheckUpdate returned %v, want a repository without releases to be normal", err)
	}
	if result.UpdateAvailable || result.LatestVersion != "" {
		t.Errorf("CheckUpdate returned %+v", result)
	}
}

func TestSkipVersionReachesTheStore(t *testing.T) {
	store := &stubStore{}
	app := appWithUpdates(&stubSource{release: released()}, store, "1.0.0")

	if err := app.SkipVersion("1.2.0"); err != nil {
		t.Fatalf("SkipVersion returned %v", err)
	}
	if store.state.SkippedVersion != "1.2.0" {
		t.Fatalf("stored %+v, want the version to be ignored", store.state)
	}
	result, err := app.CheckUpdate(false)
	if err != nil {
		t.Fatalf("CheckUpdate returned %v", err)
	}
	if !result.Skipped {
		t.Errorf("CheckUpdate returned %+v, want the ignored version reported as skipped", result)
	}
}

func TestGetVersionReturnsTheBuildVersion(t *testing.T) {
	built := version
	version = "1.2.3"
	t.Cleanup(func() { version = built })

	if got := NewApp().GetVersion(); got != "1.2.3" {
		t.Fatalf("GetVersion() = %q, want the injected version", got)
	}
}

// The browser is only ever handed a GitHub https address: the target arrives
// from a network response, so anything else must be refused before it reaches
// the platform opener.
func TestOpenReleasePageOnlyAcceptsGitHubHTTPS(t *testing.T) {
	accepted := []string{
		"https://github.com/example/SilentOpen/releases/tag/v1.2.0",
		"https://github.com/example/SilentOpen/releases/download/v1.2.0/SilentOpen.zip",
		"https://www.github.com/example/SilentOpen",
		"https://GitHub.com/example/SilentOpen",
	}
	for _, target := range accepted {
		if !isGitHubURL(target) {
			t.Errorf("isGitHubURL(%q) = false, want true", target)
		}
	}
	rejected := []string{
		"",
		"http://github.com/example/SilentOpen",
		"https://github.com.evil.test/example/SilentOpen",
		"https://evil.test/github.com",
		"https://gist.github.com/example",
		"https://objects.githubusercontent.com/example",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"github.com/example/SilentOpen",
	}
	for _, target := range rejected {
		if isGitHubURL(target) {
			t.Errorf("isGitHubURL(%q) = true, want false", target)
		}
	}
}

func TestOpenReleasePageRefusesAnythingElse(t *testing.T) {
	app := NewApp()
	// The refusal must happen before the platform opener runs, so this never
	// tries to launch a browser during a test run.
	err := app.OpenReleasePage("https://evil.test/payload")
	if !errors.Is(err, ErrUnsafeReleaseURL) {
		t.Fatalf("OpenReleasePage returned %v, want ErrUnsafeReleaseURL", err)
	}
}
