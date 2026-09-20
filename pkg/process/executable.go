package process

import (
	"github.com/shirou/gopsutil/v4/process"
)

// Executable returns the absolute path of the executable behind a process. The
// caller supplies the process creation time it observed during collection;
// passing 0 disables the identity check. startedAt matters because a PID can be
// reused between a refresh and the lookup.
//
// It takes no context on purpose. Reading a process's executable goes through
// gopsutil, whose context-aware variants ignore the context on every platform
// (process_darwin.go does not read its ctx argument at all), so accepting one
// would promise cancellation that cannot happen. The lookup reads kernel
// structures and does not block on the network.
//
// This is the only process fact the icon feature needs: resolving an icon from
// the path is the job of pkg/icon, which knows nothing about PIDs.
func Executable(pid int32, startedAt int64) string {
	if pid <= 0 {
		return ""
	}
	p, err := process.NewProcess(pid)
	if err != nil {
		return ""
	}
	if startedAt > 0 {
		created, err := p.CreateTime()
		if err != nil || created != startedAt {
			return ""
		}
	}
	executable, err := p.Exe()
	if err != nil {
		return ""
	}
	return executable
}
