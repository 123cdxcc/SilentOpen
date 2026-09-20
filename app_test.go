package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// startup is the only place that receives the application context, so a bound
// method called before it ran must degrade instead of panicking on a nil
// context. Collection itself must still succeed.
func TestGetProcessesBeforeStartup(t *testing.T) {
	snapshot, err := NewApp().GetProcesses()
	if err != nil {
		t.Fatalf("got %v, want a usable snapshot before startup", err)
	}
	// A nil error must come with a usable snapshot.
	if snapshot == nil {
		t.Fatal("a nil snapshot with a nil error violates the return contract")
	}
	if snapshot.CollectedAt <= 0 {
		t.Fatal("collection did not run before startup")
	}
}

func TestShutdownCancelsInFlightWork(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())
	app.shutdown(context.Background())
	if _, err := app.GetProcesses(); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestGetProcessIconBeforeStartup(t *testing.T) {
	if got := NewApp().GetProcessIcon(1, 1); got != "" {
		t.Fatalf("got %q, want no icon", got)
	}
}

func TestTerminateProcessGuardSurvivesFacade(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())
	err := app.TerminateProcess(1, 1)
	if err == nil || !strings.Contains(err.Error(), "PID") {
		t.Fatalf("got %v, want the PID guard error", err)
	}
}
