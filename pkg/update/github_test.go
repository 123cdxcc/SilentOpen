package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// releasePayload is a trimmed copy of what the GitHub releases API answers,
// including the fields this package ignores.
const releasePayload = `{
  "tag_name": "v1.2.0",
  "name": "SilentOpen 1.2.0",
  "body": "修好了两个问题。",
  "html_url": "https://github.com/example/SilentOpen/releases/tag/v1.2.0",
  "draft": false,
  "prerelease": false,
  "published_at": "2024-01-02T03:04:05Z",
  "assets": [
    {"name": "SilentOpen-1.2.0-macos-universal.zip", "browser_download_url": "https://example.test/macos.zip", "size": 11},
    {"name": "SilentOpen-1.2.0-windows-amd64.exe", "browser_download_url": "https://example.test/windows.exe", "size": 12}
  ]
}`

// githubServer serves one canned response and reports the request it received
// through the returned accessor, which is only valid after the call under test.
func githubServer(t *testing.T, status int, body string) (*httptest.Server, func() *http.Request) {
	t.Helper()
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = request.Clone(context.Background())
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, func() *http.Request { return received }
}

func TestGitHubReadsTheLatestRelease(t *testing.T) {
	server, received := githubServer(t, http.StatusOK, releasePayload)
	source := NewGitHub("example", "SilentOpen").WithUserAgent("SilentOpen/1.0.0")
	source.baseURL = server.URL

	release, err := source.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest returned %v", err)
	}
	if release.Version != "1.2.0" {
		t.Errorf("Version = %q, want the tag without its v prefix", release.Version)
	}
	if release.Notes != "修好了两个问题。" {
		t.Errorf("Notes = %q", release.Notes)
	}
	if release.PageURL != "https://github.com/example/SilentOpen/releases/tag/v1.2.0" {
		t.Errorf("PageURL = %q", release.PageURL)
	}
	if want := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC); !release.PublishedAt.Equal(want) {
		t.Errorf("PublishedAt = %v, want %v", release.PublishedAt, want)
	}
	if len(release.Assets) != 2 || release.Assets[0].Size != 11 || release.Assets[0].URL != "https://example.test/macos.zip" {
		t.Errorf("Assets = %+v", release.Assets)
	}
	asset := release.Asset("darwin", "arm64")
	if asset == nil || asset.Name != "SilentOpen-1.2.0-macos-universal.zip" {
		t.Errorf("Asset(darwin/arm64) = %v, want the macOS build", asset)
	}
	if received() == nil {
		t.Fatal("no request reached the server")
	}
	if got := received().URL.Path; got != "/repos/example/SilentOpen/releases/latest" {
		t.Errorf("requested %q", got)
	}
	if got := received().Header.Get("User-Agent"); got != "SilentOpen/1.0.0" {
		t.Errorf("User-Agent = %q", got)
	}
	if got := received().Header.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("Accept = %q", got)
	}
}

func TestGitHubMapsFailuresOntoTheDeclaredErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"no release", http.StatusNotFound, `{"message":"Not Found"}`, ErrNoRelease},
		{"rate limited", http.StatusForbidden, `{"message":"API rate limit exceeded"}`, ErrRateLimited},
		{"too many requests", http.StatusTooManyRequests, `{"message":"slow down"}`, ErrRateLimited},
		{"server error", http.StatusInternalServerError, `{}`, ErrResponse},
		{"not json", http.StatusOK, `{not json`, ErrResponse},
		{"no tag", http.StatusOK, `{"html_url":"https://example.test"}`, ErrResponse},
		{"unusable tag", http.StatusOK, `{"tag_name":"nightly"}`, ErrResponse},
		{"draft", http.StatusOK, `{"tag_name":"v1.2.0","draft":true}`, ErrNoRelease},
		{"pre-release", http.StatusOK, `{"tag_name":"v1.2.0","prerelease":true}`, ErrNoRelease},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, _ := githubServer(t, testCase.status, testCase.body)
			source := NewGitHub("example", "SilentOpen")
			source.baseURL = server.URL
			release, err := source.Latest(context.Background())
			if !errors.Is(err, testCase.want) {
				t.Fatalf("Latest returned %v, want %v", err, testCase.want)
			}
			if release != nil {
				t.Fatalf("Latest returned %+v together with an error", release)
			}
		})
	}
}

func TestGitHubRefusesAnUnconfiguredRepository(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	t.Cleanup(server.Close)
	for name, source := range map[string]*GitHub{
		"empty owner": NewGitHub("", "SilentOpen"),
		"empty repo":  NewGitHub("example", "  "),
		"nil source":  nil,
	} {
		if source != nil {
			source.baseURL = server.URL
		}
		release, err := source.Latest(context.Background())
		if !errors.Is(err, ErrRepoUnconfigured) {
			t.Errorf("%s: Latest returned %v, want ErrRepoUnconfigured", name, err)
		}
		if release != nil {
			t.Errorf("%s: Latest returned %+v together with an error", name, release)
		}
	}
	if requests != 0 {
		t.Fatalf("an unconfigured source made %d request(s)", requests)
	}
}

func TestGitHubReportsAnUnreachableServer(t *testing.T) {
	// Port 1 is reserved and refuses connections, which is what an offline
	// machine looks like to this code.
	source := NewGitHub("example", "SilentOpen")
	source.baseURL = "http://127.0.0.1:1"
	release, err := source.Latest(context.Background())
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("Latest returned %v, want ErrUnreachable", err)
	}
	if release != nil {
		t.Fatalf("Latest returned %+v together with an error", release)
	}
}

func TestGitHubKeepsTheCancellationCauseReachable(t *testing.T) {
	server, _ := githubServer(t, http.StatusOK, releasePayload)
	source := NewGitHub("example", "SilentOpen")
	source.baseURL = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	release, err := source.Latest(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Latest returned %v, want the cancellation to stay reachable", err)
	}
	if release != nil {
		t.Fatalf("Latest returned %+v together with an error", release)
	}
}

func TestGitHubReleasePayloadMappingIsOptionalFieldTolerant(t *testing.T) {
	// A release without notes, assets, or a publish date must still answer.
	server, _ := githubServer(t, http.StatusOK, `{"tag_name":"1.0.0"}`)
	source := NewGitHub("example", "SilentOpen")
	source.baseURL = server.URL
	release, err := source.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest returned %v", err)
	}
	if release.Version != "1.0.0" || release.Notes != "" || len(release.Assets) != 0 || !release.PublishedAt.IsZero() {
		t.Fatalf("Latest returned %+v", release)
	}
	if asset := release.Asset("linux", "amd64"); asset != nil {
		t.Fatalf("Asset(linux/amd64) = %v, want nil", asset)
	}
}
