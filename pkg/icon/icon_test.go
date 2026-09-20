package icon

import (
	"context"
	"sync"
	"testing"
)

// fakeProvider records what it was asked to resolve, so cache behaviour is
// observable without reaching into resolver internals.
type fakeProvider struct {
	icon string

	mu    sync.Mutex
	calls map[string]int
}

func (f *fakeProvider) Icon(_ context.Context, executable string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls == nil {
		f.calls = make(map[string]int)
	}
	f.calls[executable]++
	return f.icon
}

func (f *fakeProvider) callCount(executable string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[executable]
}

func TestResolverReturnsProviderIcon(t *testing.T) {
	provider := &fakeProvider{icon: "data:image/png;base64,AAAA"}
	resolver := NewResolver(provider)
	if got := resolver.Icon(context.Background(), "/usr/bin/editor"); got != provider.icon {
		t.Fatalf("got %q, want the provider's icon", got)
	}
}

func TestResolverMemoizesLookups(t *testing.T) {
	provider := &fakeProvider{icon: "data:image/png;base64,AAAA"}
	resolver := NewResolver(provider)
	for range 2 {
		resolver.Icon(context.Background(), "/usr/bin/editor")
	}
	if calls := provider.callCount("/usr/bin/editor"); calls != 1 {
		t.Fatalf("provider calls = %d, want 1", calls)
	}
}

func TestResolverMemoizesMissingIcons(t *testing.T) {
	provider := &fakeProvider{}
	resolver := NewResolver(provider)
	if got := resolver.Icon(context.Background(), "/opt/missing"); got != "" {
		t.Fatalf("got %q, want no icon", got)
	}
	if calls := provider.callCount("/opt/missing"); calls != 1 {
		t.Fatalf("provider calls = %d, want 1: a negative result must be cached", calls)
	}
}

func TestResolverRejectsEmptyPathAndCanceledContext(t *testing.T) {
	provider := &fakeProvider{icon: "data:image/png;base64,AAAA"}
	resolver := NewResolver(provider)
	if got := resolver.Icon(context.Background(), ""); got != "" {
		t.Fatalf("got %q, want no icon for an empty path", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := resolver.Icon(ctx, "/usr/bin/editor"); got != "" {
		t.Fatalf("got %q, want no icon for a canceled context", got)
	}
	if calls := provider.callCount("/usr/bin/editor"); calls != 0 {
		t.Fatalf("provider calls = %d, want 0: a canceled lookup must not resolve", calls)
	}
}

func TestResolverWithoutProviderReportsNoIcon(t *testing.T) {
	if got := NewResolver(nil).Icon(context.Background(), "/usr/bin/editor"); got != "" {
		t.Fatalf("got %q, want no icon", got)
	}
}
