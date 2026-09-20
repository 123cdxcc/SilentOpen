package update

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// cacheTTL is how long one answer is reused before the source is asked
	// again. Starting the application repeatedly must not mean one request per
	// start, and a user who opens it twice in a minute is asking one question.
	cacheTTL = 24 * time.Hour
	// notesLimit caps the release notes carried to the UI and to disk. Notes
	// are a convenience preview, and a release body is unbounded markdown.
	notesLimit = 4000
)

// Result is the answer of one check: the transport contract with the frontend,
// so the JSON keys are stable.
type Result struct {
	CurrentVersion string `json:"currentVersion"`
	// LatestVersion is empty when nothing newer is published.
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	// Skipped reports that the user asked not to be told about LatestVersion
	// again. The frontend decides what to do with it; this package keeps
	// reporting the version so a future "show ignored updates" view can use it.
	Skipped bool `json:"skipped"`
	// Notes is the release body, trimmed, not interpreted as markdown.
	Notes string `json:"notes"`
	// PageURL is the release page, the destination for a manual download.
	PageURL string `json:"pageUrl"`
	// AssetName is the release file built for the running platform, empty when
	// no attached file looks like one.
	AssetName string `json:"assetName"`
	// CheckedAt is when the answer was produced, in Unix seconds.
	CheckedAt int64 `json:"checkedAt"`
}

// Checker answers whether a newer release exists, reusing the stored answer for
// cacheTTL. Build one with NewChecker and reuse it; it is safe for concurrent
// use.
type Checker struct {
	source  Source
	store   Store
	current string
	goos    string
	goarch  string
	ttl     time.Duration
	now     func() time.Time
	mu      sync.Mutex
}

// NewChecker returns a Checker for the running version, reading releases from
// source and remembering answers in store. A current version that is not a
// version — the "dev" of a local build — never reports an update. A nil store
// means every check asks the source.
func NewChecker(source Source, store Store, current string) *Checker {
	return &Checker{
		source:  source,
		store:   store,
		current: strings.TrimSpace(current),
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
		ttl:     cacheTTL,
		now:     time.Now,
	}
}

// WithPlatform overrides the target used to pick a release asset. It exists for
// tests, which must not depend on the machine running them.
func (c *Checker) WithPlatform(goos, goarch string) *Checker {
	c.goos, c.goarch = goos, goarch
	return c
}

// WithCacheTTL overrides how long an answer is reused. A non-positive ttl makes
// every check ask the source.
func (c *Checker) WithCacheTTL(ttl time.Duration) *Checker {
	c.ttl = ttl
	return c
}

// WithClock overrides the clock, so tests can age a stored answer.
func (c *Checker) WithClock(now func() time.Time) *Checker {
	c.now = now
	return c
}

// Check reports whether a version newer than the running one is published.
// Unless force is set, an answer younger than the cache TTL is reused instead
// of asking the source.
//
// A repository without any release is not an error: it answers "nothing newer".
// Every other failure returns a nil Result, and the error message is the text
// the UI shows.
func (c *Checker) Check(ctx context.Context, force bool) (*Result, error) {
	if c == nil || c.source == nil {
		return nil, ErrRepoUnconfigured
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	stored := State{}
	if c.store != nil {
		stored = c.store.Load()
	}
	if !force && c.fresh(stored) {
		return c.result(stored), nil
	}
	release, err := c.source.Latest(ctx)
	switch {
	case errors.Is(err, ErrNoRelease):
		// Remember when the question was asked, but drop the release: a deleted
		// or withdrawn release must stop being announced. The ignored version
		// stays, so it is not re-announced when it comes back.
		next := State{CheckedAt: c.now().Unix(), SkippedVersion: stored.SkippedVersion}
		c.persist(next)
		return c.result(next), nil
	case err != nil:
		return nil, err
	}
	next := State{
		CheckedAt:      c.now().Unix(),
		LatestVersion:  release.Version,
		LatestPageURL:  release.PageURL,
		Notes:          clipNotes(release.Notes),
		SkippedVersion: stored.SkippedVersion,
	}
	if asset := release.Asset(c.goos, c.goarch); asset != nil {
		next.LatestAssetName = asset.Name
	}
	c.persist(next)
	return c.result(next), nil
}

// Skip remembers a version the user does not want to be told about again. An
// empty version clears the memory. Failures are reported as ErrStateUnwritable,
// whatever the store returned, so the caller has one error to show.
func (c *Checker) Skip(version string) error {
	if c == nil || c.store == nil {
		return fmt.Errorf("%w：未配置状态存储", ErrStateUnwritable)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	stored := c.store.Load()
	stored.SkippedVersion = strings.TrimSpace(version)
	if err := c.store.Save(stored); err != nil {
		return fmt.Errorf("%w：%w", ErrStateUnwritable, err)
	}
	return nil
}

// fresh reports whether a stored answer is young enough to reuse.
func (c *Checker) fresh(stored State) bool {
	if stored.CheckedAt <= 0 || c.ttl <= 0 {
		return false
	}
	// A timestamp from the future (a corrected clock) counts as fresh: asking
	// again would not produce a better answer.
	return c.now().Unix()-stored.CheckedAt < int64(c.ttl.Seconds())
}

// persist stores the answer. A failure is deliberately not reported: it only
// means the next check asks the source again, while the answer in hand is still
// correct, and there is nothing the user could do about it.
func (c *Checker) persist(state State) {
	if c.store == nil {
		return
	}
	_ = c.store.Save(state)
}

// result turns a stored answer into the contract with the frontend.
func (c *Checker) result(stored State) *Result {
	return &Result{
		CurrentVersion:  c.current,
		LatestVersion:   stored.LatestVersion,
		UpdateAvailable: c.newer(stored.LatestVersion),
		Skipped:         stored.LatestVersion != "" && stored.SkippedVersion == stored.LatestVersion,
		Notes:           stored.Notes,
		PageURL:         stored.LatestPageURL,
		AssetName:       stored.LatestAssetName,
		CheckedAt:       stored.CheckedAt,
	}
}

// newer reports whether a published version is one the running build should be
// told about. An unparsable running version never qualifies: a local build has
// nothing to compare against, and announcing updates to it would be noise.
func (c *Checker) newer(latest string) bool {
	if latest == "" {
		return false
	}
	current, err := ParseVersion(c.current)
	if err != nil {
		return false
	}
	published, err := ParseVersion(latest)
	if err != nil {
		return false
	}
	return Compare(published, current) > 0
}

// clipNotes bounds release notes, measuring in runes so a Chinese body is not
// cut in the middle of a character.
func clipNotes(notes string) string {
	trimmed := strings.TrimSpace(notes)
	runes := []rune(trimmed)
	if len(runes) <= notesLimit {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:notesLimit])) + "…"
}
