package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// githubAPIBase is the GitHub REST endpoint releases are read from.
	githubAPIBase = "https://api.github.com"
	// githubTimeout bounds one release lookup. The check runs behind the UI, so
	// waiting longer than this only delays an answer nobody is blocking on.
	githubTimeout = 10 * time.Second
	// githubResponseLimit caps how much of a response is read, so a misbehaving
	// endpoint cannot make the application allocate without bound.
	githubResponseLimit = 1 << 20
)

// GitHub reads the newest release of one repository from the GitHub REST API.
type GitHub struct {
	owner     string
	repo      string
	baseURL   string
	client    *http.Client
	userAgent string
}

// NewGitHub returns a Source for owner/repo. An empty owner or repo reports
// ErrRepoUnconfigured rather than requesting a URL that cannot exist.
func NewGitHub(owner, repo string) *GitHub {
	return &GitHub{
		owner:     strings.TrimSpace(owner),
		repo:      strings.TrimSpace(repo),
		baseURL:   githubAPIBase,
		client:    &http.Client{Timeout: githubTimeout},
		userAgent: "SilentOpen",
	}
}

// WithUserAgent sets the User-Agent header, which the GitHub API requires.
// A blank agent leaves the default in place.
func (g *GitHub) WithUserAgent(agent string) *GitHub {
	if strings.TrimSpace(agent) != "" {
		g.userAgent = agent
	}
	return g
}

// Latest implements Source by reading /repos/{owner}/{repo}/releases/latest,
// which GitHub answers with the newest release that is neither a draft nor a
// pre-release.
func (g *GitHub) Latest(ctx context.Context) (*Release, error) {
	if g == nil || g.owner == "" || g.repo == "" {
		return nil, ErrRepoUnconfigured
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases/latest",
		g.baseURL, url.PathEscape(g.owner), url.PathEscape(g.repo))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w：%w", ErrUnreachable, err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", g.userAgent)
	response, err := g.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w：%w", ErrUnreachable, err)
	}
	defer response.Body.Close()
	switch {
	case response.StatusCode == http.StatusNotFound:
		// GitHub answers 404 both for a missing repository and for one without
		// releases; neither is a failure the user needs to act on.
		return nil, ErrNoRelease
	case response.StatusCode == http.StatusForbidden, response.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case response.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%w：HTTP %d", ErrResponse, response.StatusCode)
	}
	var payload githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, githubResponseLimit)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w：%w", ErrResponse, err)
	}
	return payload.release()
}

// githubRelease mirrors the release fields this package reads. Everything else
// in the payload is deliberately ignored.
type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

// githubAsset mirrors one attached file.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// release maps the payload onto this package's types. Drafts and pre-releases
// are refused rather than announced: the endpoint is documented to filter them,
// and a user must not be pushed onto a build the author has not called stable.
// A tag that is not a version cannot answer a version comparison either, so it
// counts as an unreadable response.
func (p githubRelease) release() (*Release, error) {
	if p.Draft || p.Prerelease {
		return nil, ErrNoRelease
	}
	version, err := ParseVersion(p.TagName)
	if err != nil {
		return nil, fmt.Errorf("%w：%w", ErrResponse, err)
	}
	assets := make([]Asset, 0, len(p.Assets))
	for _, asset := range p.Assets {
		assets = append(assets, Asset{Name: asset.Name, URL: asset.BrowserDownloadURL, Size: asset.Size})
	}
	return &Release{
		Version:     version.String(),
		Notes:       p.Body,
		PageURL:     p.HTMLURL,
		PublishedAt: p.PublishedAt,
		Assets:      assets,
	}, nil
}
