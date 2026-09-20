package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Terminate failures are declared once so callers can match them with
// errors.Is instead of comparing strings. The wrapped cause stays available
// through Unwrap, and the messages are the exact text the UI displays.
var (
	// ErrInvalidPID rejects PIDs that must never be signalled.
	ErrInvalidPID = errors.New("不允许结束 PID 小于或等于 1 的进程")
	// ErrSelf reports an attempt to signal the running application itself.
	ErrSelf = errors.New("不允许结束 SilentOpen 自身进程")
	// ErrMissingStartTime rejects a request without its identity guard.
	ErrMissingStartTime = errors.New("缺少进程启动时间，无法安全结束进程")
	// ErrStartTimeUnreadable reports a failed identity check.
	ErrStartTimeUnreadable = errors.New("无法读取进程启动时间，已拒绝结束进程")
	// ErrIdentityChanged guards against a PID reused since the last refresh.
	ErrIdentityChanged = errors.New("进程身份已变化，请刷新列表后重试")
	// ErrProcessExited reports a target that is already gone.
	ErrProcessExited = errors.New("进程已退出，请刷新列表")
	// ErrPermission reports a target this user may not signal.
	ErrPermission = errors.New("权限不足")
	// ErrTerminateFailed reports a signal the system refused.
	ErrTerminateFailed = errors.New("系统未能结束进程")
)

// terminateTimeout bounds one termination request.
const terminateTimeout = 5 * time.Second

// Terminate requests termination of one process, never its descendants.
// startedAt is the creation time the caller observed during collection; it is
// re-verified here so a reused PID is never signalled by mistake. POSIX sends
// SIGTERM and Windows terminates the process directly, so success means the
// request was delivered, not that the process has exited.
func Terminate(parent context.Context, pid int32, startedAt int64) error {
	if pid <= 1 {
		return ErrInvalidPID
	}
	if pid == int32(os.Getpid()) {
		return ErrSelf
	}
	if startedAt <= 0 {
		return ErrMissingStartTime
	}
	ctx, cancel := context.WithTimeout(parent, terminateTimeout)
	defer cancel()
	// A new Process avoids reusing the creation time cached during collection.
	p, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return terminationError(ctx, pid, err)
	}
	created, err := p.CreateTimeWithContext(ctx)
	if ctx.Err() != nil {
		return terminationError(ctx, pid, ctx.Err())
	}
	if err != nil {
		return fmt.Errorf("%w：%w", ErrStartTimeUnreadable, err)
	}
	if created <= 0 {
		return ErrStartTimeUnreadable
	}
	if created != startedAt {
		return ErrIdentityChanged
	}
	if err := p.TerminateWithContext(ctx); err != nil {
		return terminationError(ctx, pid, err)
	}
	return nil
}

// terminationError maps a platform failure onto the declared errors while
// keeping the underlying cause reachable.
func terminationError(ctx context.Context, pid int32, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("结束进程请求已取消或超时：%w", ctx.Err())
	}
	if errors.Is(err, process.ErrorProcessNotRunning) || errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("%w：%w", ErrProcessExited, process.ErrorProcessNotRunning)
	}
	if exists, checkErr := process.PidExistsWithContext(ctx, pid); checkErr == nil && !exists {
		return fmt.Errorf("%w：%w", ErrProcessExited, process.ErrorProcessNotRunning)
	}
	if errors.Is(err, os.ErrPermission) || errors.Is(err, process.ErrorNotPermitted) {
		return fmt.Errorf("%w：%w", ErrPermission, err)
	}
	return fmt.Errorf("%w：%w", ErrTerminateFailed, err)
}
