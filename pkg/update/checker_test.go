package update

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeSource counts how often the checker actually asked it, which is what the
// caching behaviour is about.
type fakeSource struct {
	release *Release
	err     error
	calls   int
}

func (f *fakeSource) Latest(context.Context) (*Release, error) {
	f.calls++
	switch {
	case f.err != nil:
		return nil, f.err
	case f.release == nil:
		return nil, ErrNoRelease
	}
	return f.release, nil
}

// fakeStore keeps state in memory and can be made to fail writes.
type fakeStore struct {
	state State
	saves int
	err   error
}

func (f *fakeStore) Load() State { return f.state }

func (f *fakeStore) Save(state State) error {
	if f.err != nil {
		return f.err
	}
	f.saves++
	f.state = state
	return nil
}

func sampleRelease() *Release {
	return &Release{
		Version:     "1.2.0",
		Notes:       "修好了两个问题。",
		PageURL:     "https://github.com/example/SilentOpen/releases/tag/v1.2.0",
		PublishedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		Assets: []Asset{
			{Name: "SilentOpen-1.2.0-macos-universal.zip", URL: "https://example.test/macos.zip"},
			{Name: "SilentOpen-1.2.0-windows-amd64.exe", URL: "https://example.test/windows.exe"},
			{Name: "SilentOpen-1.2.0-linux-amd64.tar.gz", URL: "https://example.test/linux.tar.gz"},
		},
	}
}

func TestCheckReportsANewerRelease(t *testing.T) {
	source, store := &fakeSource{release: sampleRelease()}, &fakeStore{}
	current := time.Unix(1735689600, 0)
	checker := NewChecker(source, store, "1.0.0").
		WithPlatform("darwin", "arm64").
		WithClock(func() time.Time { return current })

	result, err := checker.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if result == nil {
		t.Fatal("Check returned a nil result with a nil error")
	}
	if !result.UpdateAvailable || result.LatestVersion != "1.2.0" || result.CurrentVersion != "1.0.0" {
		t.Errorf("Check returned %+v", result)
	}
	if result.AssetName != "SilentOpen-1.2.0-macos-universal.zip" {
		t.Errorf("Check picked %q, want the macOS build", result.AssetName)
	}
	if result.PageURL == "" || result.Notes == "" {
		t.Errorf("Check returned %+v, want the page and the notes", result)
	}
	if result.CheckedAt != current.Unix() {
		t.Errorf("CheckedAt = %d, want %d", result.CheckedAt, current.Unix())
	}
	if result.Skipped {
		t.Error("a release nobody ignored must not report as skipped")
	}
	if store.saves != 1 || store.state.LatestVersion != "1.2.0" || store.state.LatestAssetName != result.AssetName {
		t.Errorf("stored %+v after %d save(s)", store.state, store.saves)
	}
}

func TestCheckDoesNotAnnounceEqualOrOlderReleases(t *testing.T) {
	for _, currentVersion := range []string{"1.2.0", "1.3.0", "2.0.0"} {
		checker := NewChecker(&fakeSource{release: sampleRelease()}, &fakeStore{}, currentVersion)
		result, err := checker.Check(context.Background(), true)
		if err != nil {
			t.Fatalf("Check returned %v", err)
		}
		if result.UpdateAvailable {
			t.Errorf("current %s: Check announced %s", currentVersion, result.LatestVersion)
		}
	}
}

// A build from a pre-release tag is upgraded by the release it leads to.
func TestCheckAnnouncesTheReleaseAPreReleaseLeadsTo(t *testing.T) {
	checker := NewChecker(&fakeSource{release: sampleRelease()}, &fakeStore{}, "1.2.0-rc.1")
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if !result.UpdateAvailable {
		t.Errorf("Check returned %+v, want 1.2.0 to upgrade 1.2.0-rc.1", result)
	}
}

func TestCheckNeverAnnouncesUpdatesToALocalBuild(t *testing.T) {
	checker := NewChecker(&fakeSource{release: sampleRelease()}, &fakeStore{}, "dev")
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	// The published version is still reported, so the UI can show what is out
	// there; it just must not claim an update is available.
	if result.UpdateAvailable {
		t.Errorf("Check announced an update to a dev build: %+v", result)
	}
	if result.LatestVersion != "1.2.0" {
		t.Errorf("LatestVersion = %q, want the published version", result.LatestVersion)
	}
}

func TestCheckReusesAStoredAnswerUntilItExpires(t *testing.T) {
	source, store := &fakeSource{release: sampleRelease()}, &fakeStore{}
	now := time.Unix(1735689600, 0)
	checker := NewChecker(source, store, "1.0.0").WithClock(func() time.Time { return now })

	cached := store.Load()
	cached.CheckedAt = now.Unix()
	cached.LatestVersion = "1.1.0"
	cached.LatestPageURL = "https://example.test/1.1.0"
	store.state = cached

	result, err := checker.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if source.calls != 0 {
		t.Errorf("a fresh answer still asked the source %d time(s)", source.calls)
	}
	if result.LatestVersion != "1.1.0" {
		t.Errorf("LatestVersion = %q, want the stored answer", result.LatestVersion)
	}

	// A forced check ignores the cache.
	if _, err := checker.Check(context.Background(), true); err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if source.calls != 1 || store.state.LatestVersion != "1.2.0" {
		t.Errorf("a forced check asked %d time(s) and stored %+v", source.calls, store.state)
	}

	// Once the answer is older than the TTL it is no longer reused.
	now = now.Add(cacheTTL + time.Minute)
	if _, err := checker.Check(context.Background(), false); err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if source.calls != 2 {
		t.Errorf("an expired answer asked the source %d time(s), want 2", source.calls)
	}
}

func TestCheckAsksEveryTimeWithoutACache(t *testing.T) {
	source := &fakeSource{release: sampleRelease()}
	checker := NewChecker(source, &fakeStore{}, "1.0.0").WithCacheTTL(0)
	for range 3 {
		if _, err := checker.Check(context.Background(), false); err != nil {
			t.Fatalf("Check returned %v", err)
		}
	}
	if source.calls != 3 {
		t.Errorf("the source was asked %d time(s), want 3", source.calls)
	}
}

func TestCheckTreatsAMissingReleaseAsNothingNewer(t *testing.T) {
	store := &fakeStore{state: State{SkippedVersion: "1.1.0"}}
	checker := NewChecker(&fakeSource{}, store, "1.0.0")
	result, err := checker.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("Check returned %v, want a repository without releases to be normal", err)
	}
	if result.UpdateAvailable || result.LatestVersion != "" {
		t.Errorf("Check returned %+v", result)
	}
	if result.CheckedAt == 0 {
		t.Error("a completed check must carry its timestamp, so the cache can hold")
	}
	if store.state.SkippedVersion != "1.1.0" {
		t.Errorf("stored %+v, want the ignored version kept", store.state)
	}
}

func TestCheckStopsAnnouncingAWithdrawnRelease(t *testing.T) {
	store := &fakeStore{state: State{
		CheckedAt:      time.Now().Unix(),
		LatestVersion:  "1.2.0",
		LatestPageURL:  "https://example.test/1.2.0",
		SkippedVersion: "1.1.0",
	}}
	checker := NewChecker(&fakeSource{}, store, "1.0.0").WithCacheTTL(0)
	result, err := checker.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if result.LatestVersion != "" || result.PageURL != "" {
		t.Errorf("Check returned %+v, want the withdrawn release forgotten", result)
	}
}

func TestCheckReportsSourceFailuresAndStoresNothing(t *testing.T) {
	failure := errors.New("boom")
	store := &fakeStore{state: State{CheckedAt: 1, LatestVersion: "1.1.0"}}
	checker := NewChecker(&fakeSource{err: failure}, store, "1.0.0").WithCacheTTL(0)
	result, err := checker.Check(context.Background(), false)
	if !errors.Is(err, failure) {
		t.Fatalf("Check returned %v, want the source failure", err)
	}
	if result != nil {
		t.Fatalf("Check returned %+v together with an error", result)
	}
	// The last good answer survives, so an offline machine keeps showing it.
	if store.saves != 0 || store.state.LatestVersion != "1.1.0" {
		t.Errorf("stored %+v after %d save(s), want no write", store.state, store.saves)
	}
}

func TestCheckSurvivesAnUnwritableStore(t *testing.T) {
	store := &fakeStore{err: errors.New("read-only")}
	checker := NewChecker(&fakeSource{release: sampleRelease()}, store, "1.0.0")
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check returned %v, want the answer even when it cannot be stored", err)
	}
	if !result.UpdateAvailable {
		t.Errorf("Check returned %+v", result)
	}
}

func TestCheckWorksWithoutAStore(t *testing.T) {
	checker := NewChecker(&fakeSource{release: sampleRelease()}, nil, "1.0.0")
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if !result.UpdateAvailable {
		t.Errorf("Check returned %+v", result)
	}
}

func TestCheckReportsAnUnconfiguredSource(t *testing.T) {
	result, err := NewChecker(nil, &fakeStore{}, "1.0.0").Check(context.Background(), false)
	if !errors.Is(err, ErrRepoUnconfigured) {
		t.Fatalf("Check returned %v, want ErrRepoUnconfigured", err)
	}
	if result != nil {
		t.Fatalf("Check returned %+v together with an error", result)
	}
	var checker *Checker
	if result, err := checker.Check(context.Background(), false); !errors.Is(err, ErrRepoUnconfigured) || result != nil {
		t.Fatalf("a nil checker returned %+v, %v", result, err)
	}
}

func TestCheckPicksTheAssetForTheConfiguredPlatform(t *testing.T) {
	checker := NewChecker(&fakeSource{release: sampleRelease()}, &fakeStore{}, "1.0.0").WithPlatform("windows", "amd64")
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if result.AssetName != "SilentOpen-1.2.0-windows-amd64.exe" {
		t.Errorf("AssetName = %q, want the Windows build", result.AssetName)
	}
}

func TestCheckClipsLongNotesAtARuneBoundary(t *testing.T) {
	release := sampleRelease()
	release.Notes = "  " + strings.Repeat("版", notesLimit+50) + "  "
	checker := NewChecker(&fakeSource{release: release}, &fakeStore{}, "1.0.0")
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if runes := []rune(result.Notes); len(runes) != notesLimit+1 {
		t.Fatalf("notes are %d runes, want %d plus the ellipsis", len(runes), notesLimit)
	}
	if !strings.HasSuffix(result.Notes, "…") {
		t.Errorf("clipped notes do not end in an ellipsis: %q", result.Notes[max(0, len(result.Notes)-3):])
	}
}

func TestSkipRemembersAndClearsAVersion(t *testing.T) {
	store := &fakeStore{}
	checker := NewChecker(&fakeSource{release: sampleRelease()}, store, "1.0.0")
	if err := checker.Skip(" 1.2.0 "); err != nil {
		t.Fatalf("Skip returned %v", err)
	}
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if !result.Skipped || !result.UpdateAvailable {
		t.Errorf("Check returned %+v, want an available but skipped update", result)
	}
	if err := checker.Skip(""); err != nil {
		t.Fatalf("Skip returned %v", err)
	}
	if result, err = checker.Check(context.Background(), false); err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if result.Skipped {
		t.Errorf("clearing the ignored version left %+v", result)
	}
}

func TestSkipReportsAnUnwritableStore(t *testing.T) {
	if err := NewChecker(&fakeSource{}, &fakeStore{err: errors.New("read-only")}, "1.0.0").Skip("1.2.0"); !errors.Is(err, ErrStateUnwritable) {
		t.Fatalf("Skip returned %v, want ErrStateUnwritable", err)
	}
	if err := NewChecker(&fakeSource{}, nil, "1.0.0").Skip("1.2.0"); !errors.Is(err, ErrStateUnwritable) {
		t.Fatalf("Skip returned %v, want ErrStateUnwritable", err)
	}
	var checker *Checker
	if err := checker.Skip("1.2.0"); !errors.Is(err, ErrStateUnwritable) {
		t.Fatalf("a nil checker returned %v, want ErrStateUnwritable", err)
	}
}

func TestSkipIsNotUndoneByACheckThatFindsNothing(t *testing.T) {
	store := &fakeStore{}
	checker := NewChecker(&fakeSource{}, store, "1.0.0").WithCacheTTL(0)
	if err := checker.Skip("1.2.0"); err != nil {
		t.Fatalf("Skip returned %v", err)
	}
	if _, err := checker.Check(context.Background(), false); err != nil {
		t.Fatalf("Check returned %v", err)
	}
	if store.state.SkippedVersion != "1.2.0" {
		t.Fatalf("stored %+v, want the ignored version kept", store.state)
	}
}
