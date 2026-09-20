package main

import (
	"context"

	"github.com/123cdxcc/SilentOpen/pkg/icon"
	"github.com/123cdxcc/SilentOpen/pkg/process"
	"github.com/123cdxcc/SilentOpen/pkg/update"
)

// App is the Wails binding surface. Every exported method becomes an RPC call
// available to the frontend; the bound type and its package decide the
// generated binding path (wailsjs/go/main/App).
type App struct {
	// ctx is the application context handed over by startup. Wails calls bound
	// methods without a context argument, so the lifecycle hooks own it. It is
	// nil until startup runs, which the accessor below accounts for.
	ctx     context.Context
	cancel  context.CancelFunc
	icons   *icon.Resolver
	updates *update.Checker
}

// NewApp creates the application. The context arrives later, in startup.
func NewApp() *App {
	return &App{
		icons:   icon.NewResolver(icon.NewPlatformProvider()),
		updates: newUpdateChecker(),
	}
}

// startup is called at application startup and receives the application
// context. The derived context is cancelled by shutdown, so in-flight
// collection and termination stop when the window closes.
func (a *App) startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
}

// shutdown is called at application termination.
func (a *App) shutdown(context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
}

// applicationContext returns the context handed over by startup. A bound method
// is callable before startup (and in tests), so an unset context must degrade
// rather than panic deeper inside a platform call.
func (a *App) applicationContext() context.Context {
	if a.ctx == nil {
		return context.TODO()
	}
	return a.ctx
}

// GetProcesses returns the processes currently listening on TCP. It returns a
// nil snapshot together with the error when the collection fails.
func (a *App) GetProcesses() (*process.ProcessSnapshot, error) {
	return process.Collect(a.applicationContext())
}

// GetProcessIcon returns the application icon of a process as a data URL, or an
// empty string when this process has no readable icon. The process supplies the
// executable path; resolving an icon from that path is pkg/icon's job.
func (a *App) GetProcessIcon(pid int32, startedAt int64) string {
	return a.icons.Icon(a.applicationContext(), process.Executable(pid, startedAt))
}

// TerminateProcess requests termination of a single process. The returned error
// carries a message meant to be shown to the user as-is.
func (a *App) TerminateProcess(pid int32, startedAt int64) error {
	return process.Terminate(a.applicationContext(), pid, startedAt)
}
