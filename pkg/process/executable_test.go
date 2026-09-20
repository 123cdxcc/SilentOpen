package process

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shirou/gopsutil/v4/process"
)

func TestExecutableReturnsOwnPath(t *testing.T) {
	want, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// macOS resolves /var to /private/var, and gopsutil reports the canonical
	// path, so compare canonical forms on both sides.
	want, err = filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatal(err)
	}
	pid := int32(os.Getpid())
	p, err := process.NewProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	startedAt, err := p.CreateTime()
	if err != nil {
		t.Fatal(err)
	}
	got := Executable(pid, startedAt)
	if resolved, err := filepath.EvalSymlinks(got); err == nil {
		got = resolved
	}
	if got != want {
		t.Fatalf("Executable() = %q, want %q", got, want)
	}
	// A stale identity must not return a path for a reused PID.
	if got := Executable(pid, startedAt+1); got != "" {
		t.Fatalf("Executable() = %q, want no path for a changed identity", got)
	}
}

func TestExecutableRejectsInvalidPID(t *testing.T) {
	for _, pid := range []int32{0, -1} {
		if got := Executable(pid, 1); got != "" {
			t.Fatalf("Executable(%d) = %q, want no path", pid, got)
		}
	}
}
