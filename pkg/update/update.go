// Package update answers whether a newer release of this application has been
// published, and where to get it. It knows nothing about Wails, about the
// transport the answer travels on, or about the frontend that renders it.
//
// # Scope
//
// The package deliberately stops at "a newer version exists, here is its page
// and the asset built for this platform". It never downloads or installs
// anything: the three platforms ship different payloads (a .app bundle on
// macOS, an executable or installer on Windows, a single binary on Linux) and
// replacing a running application is a platform-specific operation with its own
// failure modes. Callers that want it add it explicitly.
//
// # Return contract
//
// Functions that return a value together with an error follow the project-wide
// rule:
//
//   - a struct result is always a pointer, never a struct value;
//   - a nil error means the value is non-nil;
//   - a non-nil error means the value is nil.
package update

import (
	"context"
	"errors"
	"time"
)

// Release is the part of a published release this application cares about.
type Release struct {
	// Version is the release tag without its leading "v".
	Version     string
	Notes       string
	PageURL     string
	PublishedAt time.Time
	Assets      []Asset
}

// Asset is one downloadable file attached to a release.
type Asset struct {
	Name string
	URL  string
	Size int64
}

// Source reports the newest release of one repository. The interface exists so
// the checking logic does not depend on the network: tests inject a fake and
// the real implementation reads the GitHub REST API.
type Source interface {
	// Latest returns the newest published release. A repository that has no
	// release at all returns ErrNoRelease, which callers treat as "nothing
	// newer is published" rather than as a failure.
	Latest(ctx context.Context) (*Release, error)
}

// Failures are declared once so callers can match them with errors.Is instead
// of comparing strings. The messages are the exact text the UI displays.
var (
	// ErrNoRelease reports a repository without any published release.
	ErrNoRelease = errors.New("该仓库尚未发布任何版本")
	// ErrRepoUnconfigured reports a build that has no repository to check.
	ErrRepoUnconfigured = errors.New("尚未配置发布仓库，无法检查更新")
	// ErrUnreachable reports a request that never produced a response.
	ErrUnreachable = errors.New("无法连接更新服务器")
	// ErrRateLimited reports a lookup the server refused to serve.
	ErrRateLimited = errors.New("更新检查过于频繁，请稍后再试")
	// ErrResponse reports a response that cannot be understood.
	ErrResponse = errors.New("更新服务器返回了无法识别的数据")
	// ErrStateUnwritable reports state that could not be persisted.
	ErrStateUnwritable = errors.New("无法保存更新设置")
)
