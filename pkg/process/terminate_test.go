package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func TestTerminate(t *testing.T) {
	if os.Getenv("SILENTOPEN_TERMINATE_TEST_CHILD") == "1" {
		time.Sleep(time.Minute)
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^TestTerminate$")
	child.Env = append(os.Environ(), "SILENTOPEN_TERMINATE_TEST_CHILD=1")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		_ = child.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		_ = child.Process.Kill()
		<-done
	})
	pid := int32(child.Process.Pid)
	p, err := process.NewProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	startedAt, err := p.CreateTime()
	if err != nil || startedAt <= 0 {
		t.Fatalf("cannot read child creation time: %d, %v", startedAt, err)
	}
	for _, tt := range []struct {
		name      string
		pid       int32
		startedAt int64
		want      string
		wantErr   error
	}{
		{"negative PID", -1, startedAt, "PID", ErrInvalidPID},
		{"zero PID", 0, startedAt, "PID", ErrInvalidPID},
		{"system PID", 1, startedAt, "PID", ErrInvalidPID},
		{"self", int32(os.Getpid()), startedAt, "自身", ErrSelf},
		{"missing timestamp", pid, 0, "缺少进程启动时间", ErrMissingStartTime},
		{"negative timestamp", pid, -1, "缺少进程启动时间", ErrMissingStartTime},
		{"reused PID", pid, startedAt + 1, "身份已变化", ErrIdentityChanged},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := Terminate(context.Background(), tt.pid, tt.startedAt)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want errors.Is %v", err, tt.wantErr)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Terminate(ctx, pid, startedAt); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := Terminate(ctx, pid, startedAt); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
	select {
	case <-done:
		t.Fatal("rejected requests must leave the child running")
	default:
	}
	if err := Terminate(context.Background(), pid, startedAt); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("child did not exit after termination")
	}
	if err := Terminate(context.Background(), pid, startedAt); !errors.Is(err, ErrProcessExited) {
		t.Fatalf("got %v, want ErrProcessExited", err)
	}
}
