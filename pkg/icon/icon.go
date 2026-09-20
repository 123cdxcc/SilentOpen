// Package icon resolves the application icon of an executable file into a data
// URL. It knows nothing about processes: callers find the path, this package
// answers what icon the operating system associates with it.
package icon

import (
	"context"
	"sync"
)

// Provider resolves the application icon of one executable path. Platform
// implementations live in provider_<goos>.go; the interface exists so the
// lookup can be replaced in tests and from other packages.
type Provider interface {
	// Icon returns a data URL, or an empty string when the platform has no
	// icon for the path.
	Icon(ctx context.Context, executable string) string
}

// Resolver looks up icons through a Provider and caches results per
// executable, including negative ones. Build one with NewResolver and reuse it;
// it is safe for concurrent use.
type Resolver struct {
	provider Provider
	cached   map[string]string
	mu       sync.Mutex
}

// maxCachedIcons bounds the resolver's memory. Reaching it clears the cache
// rather than evicting a single entry.
const maxCachedIcons = 128

// NewResolver returns a Resolver backed by provider. A nil provider yields a
// resolver that always reports "no icon", so a partially configured caller
// degrades instead of panicking.
func NewResolver(provider Provider) *Resolver {
	if provider == nil {
		provider = noIcons{}
	}
	return &Resolver{provider: provider, cached: make(map[string]string)}
}

// Icon returns the cached or freshly resolved data URL for executable, or an
// empty string when there is no icon to report. A nil resolver reports no icon.
func (r *Resolver) Icon(ctx context.Context, executable string) string {
	if r == nil {
		return ""
	}
	if executable == "" || ctx.Err() != nil {
		return ""
	}
	r.mu.Lock()
	value, found := r.cached[executable]
	r.mu.Unlock()
	if found {
		return value
	}
	value = r.provider.Icon(ctx, executable)
	if ctx.Err() != nil {
		return ""
	}
	r.mu.Lock()
	if len(r.cached) >= maxCachedIcons {
		clear(r.cached)
	}
	r.cached[executable] = value
	r.mu.Unlock()
	return value
}

// noIcons is the Provider used when none was supplied.
type noIcons struct{}

func (noIcons) Icon(context.Context, string) string { return "" }
