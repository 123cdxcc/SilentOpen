// Package process collects the processes listening on TCP ports, describes
// them, and can terminate a single process. It is a reusable building block:
// it knows nothing about the application's UI, transport, or business rules.
//
// # Return contract
//
// Functions that return a value together with an error follow the project-wide
// rule:
//
//   - a struct result is always a pointer, never a struct value;
//   - a nil error means the value is non-nil;
//   - a non-nil error means the value is nil.
//
// A nil value with a nil error is a bug, not a state callers must handle.
package process

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// ProcessInfo describes one process and the endpoints it listens on. The
// exported fields are the transport contract: the frontend consumes them as
// JSON, so their names and shapes are stable.
type ProcessInfo struct {
	PID        int32      `json:"pid"`
	PPID       *int32     `json:"ppid"`
	ParentName string     `json:"parentName"`
	Name       string     `json:"name"`
	Project    string     `json:"project"`
	Cwd        string     `json:"cwd"`
	Listeners  []Listener `json:"listeners"`
	Command    string     `json:"command"`
	StartedAt  *int64     `json:"startedAt"`
}

// Listener is one TCP endpoint a process listens on.
type Listener struct {
	IP   string `json:"ip"`
	Port uint32 `json:"port"`
}

// ProcessSnapshot is one collection result plus the warnings that qualify it.
type ProcessSnapshot struct {
	Processes   []ProcessInfo `json:"processes"`
	CollectedAt int64         `json:"collectedAt"`
	Warnings    []string      `json:"warnings"`
}

// collectTimeout bounds a single collection so a slow platform call cannot
// hang the caller indefinitely.
const collectTimeout = 5 * time.Second

// ErrSnapshotUnavailable reports a collection that produced no snapshot. It
// exists so the documented return contract never has to be broken: a returned
// error always comes with a nil snapshot.
var ErrSnapshotUnavailable = errors.New("未能读取进程列表")

// Collect returns every process that currently listens on TCP, aggregated by
// PID. Unreadable fields stay empty and are reported through Warnings rather
// than failing the whole collection.
//
// The return contract is the project-wide one: a nil error means the snapshot
// is non-nil, and a non-nil error means it is nil.
func Collect(parent context.Context) (*ProcessSnapshot, error) {
	ctx, cancel := context.WithTimeout(parent, collectTimeout)
	defer cancel()
	connections, err := gnet.ConnectionsWithContext(ctx, "tcp")
	// ConnectionsWithContext runs lsof through exec.CommandContext, so a
	// canceled or expired context surfaces as an error we must not mask with a
	// partial result. The per-process calls below read /proc and sysctl; their
	// ctx parameter is ignored by gopsutil on every platform, so they cannot
	// report cancellation.
	if ctx.Err() != nil {
		return nil, cancellationError(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("读取监听端口失败：%w", err)
	}
	rows, unowned := groupListeners(connections)
	snapshot := &ProcessSnapshot{Processes: make([]ProcessInfo, 0, len(rows)), Warnings: []string{}}
	if unowned {
		snapshot.Warnings = append(snapshot.Warnings, "部分监听端口无法读取所属 PID，未列入进程表。")
	}
	partial := false
	for _, row := range rows {
		described, err := describeProcess(ctx, row)
		if errors.Is(err, process.ErrorProcessNotRunning) {
			continue
		}
		if err != nil || incomplete(described) {
			partial = true
		}
		snapshot.Processes = append(snapshot.Processes, described)
	}
	if partial {
		snapshot.Warnings = append(snapshot.Warnings, "部分进程信息不可读；未知字段可能受权限限制或进程退出影响。")
	}
	snapshot.CollectedAt = time.Now().UnixMilli()
	return snapshot, nil
}

// cancellationError keeps the sentinel promise intact: callers see the actual
// context cause, and errors.Is still matches it.
func cancellationError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w：%w", ErrSnapshotUnavailable, err)
	}
	return ErrSnapshotUnavailable
}

// incomplete reports whether any field the frontend relies on could not be read.
func incomplete(info ProcessInfo) bool {
	switch {
	case info.Name == "":
		return true
	case info.Cwd == "":
		return true
	case info.Command == "":
		return true
	case info.PPID == nil:
		return true
	case info.StartedAt == nil:
		return true
	// PID 1 has no parent name to read.
	case *info.PPID > 0 && info.ParentName == "":
		return true
	}
	return false
}

// groupListeners aggregates listening sockets by owning PID. Rows are sorted by
// PID and each row's listeners by port then IP, so a snapshot is stable across
// calls. The second result reports sockets whose owning PID was unreadable.
func groupListeners(connections []gnet.ConnectionStat) ([]ProcessInfo, bool) {
	listenersByPID := make(map[int32][]Listener)
	unowned := false
	for _, connection := range connections {
		if connection.Status != "LISTEN" {
			continue
		}
		if connection.Pid <= 0 {
			unowned = true
			continue
		}
		ip := connection.Laddr.IP
		if ip == "*" {
			switch connection.Family {
			case syscall.AF_INET:
				ip = "0.0.0.0"
			case syscall.AF_INET6:
				ip = "::"
			}
		}
		listenersByPID[connection.Pid] = append(listenersByPID[connection.Pid], Listener{IP: ip, Port: connection.Laddr.Port})
	}
	rows := make([]ProcessInfo, 0, len(listenersByPID))
	for pid, listeners := range listenersByPID {
		slices.SortFunc(listeners, func(a, b Listener) int {
			if order := cmp.Compare(a.Port, b.Port); order != 0 {
				return order
			}
			return cmp.Compare(a.IP, b.IP)
		})
		rows = append(rows, ProcessInfo{PID: pid, Listeners: slices.Compact(listeners)})
	}
	slices.SortFunc(rows, func(a, b ProcessInfo) int { return cmp.Compare(a.PID, b.PID) })
	return rows, unowned
}

// describeProcess fills in a row's metadata. Missing or inaccessible fields stay
// empty; the caller decides how to report that through the snapshot warnings.
func describeProcess(ctx context.Context, info ProcessInfo) (ProcessInfo, error) {
	p, err := process.NewProcessWithContext(ctx, info.PID)
	if err != nil {
		return info, err
	}
	info.Name, _ = p.NameWithContext(ctx)
	info.Command, _ = p.CmdlineWithContext(ctx)
	info.Cwd, _ = p.CwdWithContext(ctx)
	if info.Cwd != "" {
		// The working directory's last element labels the project; a real
		// repository lookup can replace this if that proves insufficient.
		info.Project = filepath.Base(info.Cwd)
	}
	if ppid, err := p.PpidWithContext(ctx); err == nil {
		info.PPID = &ppid
		if ppid > 0 {
			if parent, err := process.NewProcessWithContext(ctx, ppid); err == nil {
				info.ParentName, _ = parent.NameWithContext(ctx)
			}
		}
	}
	if started, err := p.CreateTimeWithContext(ctx); err == nil && started > 0 {
		info.StartedAt = &started
	}
	return info, nil
}
