package process

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"syscall"
	"testing"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

func TestGroupListeners(t *testing.T) {
	listener := func(pid int32, ip string, port uint32, family uint32) gnet.ConnectionStat {
		return gnet.ConnectionStat{Pid: pid, Status: "LISTEN", Family: family, Laddr: gnet.Addr{IP: ip, Port: port}}
	}
	tests := []struct {
		name        string
		connections []gnet.ConnectionStat
		want        []ProcessInfo
		unowned     bool
	}{
		{"empty", nil, []ProcessInfo{}, false},
		{"ignore established", []gnet.ConnectionStat{{Pid: 9, Status: "ESTABLISHED", Laddr: gnet.Addr{Port: 3000}}}, []ProcessInfo{}, false},
		{
			"deduplicate endpoints and sort",
			[]gnet.ConnectionStat{
				listener(20, "::1", 8080, syscall.AF_INET6),
				listener(10, "127.0.0.1", 3000, syscall.AF_INET),
				listener(20, "127.0.0.1", 80, syscall.AF_INET),
				listener(20, "127.0.0.1", 8080, syscall.AF_INET),
				listener(20, "127.0.0.2", 8080, syscall.AF_INET),
				listener(20, "::1", 8080, syscall.AF_INET6),
			},
			[]ProcessInfo{
				{PID: 10, Listeners: []Listener{{IP: "127.0.0.1", Port: 3000}}},
				{PID: 20, Listeners: []Listener{{IP: "127.0.0.1", Port: 80}, {IP: "127.0.0.1", Port: 8080}, {IP: "127.0.0.2", Port: 8080}, {IP: "::1", Port: 8080}}},
			},
			false,
		},
		{
			"normalize wildcard addresses without inventing unknown IPs",
			[]gnet.ConnectionStat{
				listener(10, "*", 3000, syscall.AF_INET),
				listener(10, "*", 3000, syscall.AF_INET6),
				listener(10, "0.0.0.0", 3000, syscall.AF_INET),
				listener(10, "", 3000, syscall.AF_INET),
			},
			[]ProcessInfo{{PID: 10, Listeners: []Listener{{IP: "", Port: 3000}, {IP: "0.0.0.0", Port: 3000}, {IP: "::", Port: 3000}}}},
			false,
		},
		{"unknown owner", []gnet.ConnectionStat{listener(0, "127.0.0.1", 8080, syscall.AF_INET), listener(-1, "127.0.0.1", 9000, syscall.AF_INET), listener(10, "127.0.0.1", 3000, syscall.AF_INET)}, []ProcessInfo{{PID: 10, Listeners: []Listener{{IP: "127.0.0.1", Port: 3000}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, unowned := groupListeners(tt.connections)
			if !reflect.DeepEqual(got, tt.want) || unowned != tt.unowned {
				t.Fatalf("got %#v, unowned %v; want %#v, unowned %v", got, unowned, tt.want, tt.unowned)
			}
		})
	}
	data, err := json.Marshal(ProcessInfo{PID: 10, Listeners: []Listener{{IP: "127.0.0.1", Port: 3000}}})
	if err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(data, &row); err != nil {
		t.Fatal(err)
	}
	if row["startedAt"] != nil || row["ppid"] != nil || row["parentName"] != "" || row["cwd"] != "" {
		t.Fatalf("unreadable metadata must remain unknown: %s", data)
	}
}

// The frontend contract is the JSON shape, so a rename that changes it must
// fail here rather than silently in the UI.
func TestProcessInfoJSONShape(t *testing.T) {
	data, err := json.Marshal(ProcessInfo{})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	want := []string{"command", "cwd", "listeners", "name", "parentName", "pid", "ppid", "project", "startedAt"}
	if !slices.Equal(keys, want) {
		t.Fatalf("ProcessInfo JSON keys = %v, want %v", keys, want)
	}
}

func TestCollectCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	snapshot, err := Collect(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if !errors.Is(err, ErrSnapshotUnavailable) {
		t.Fatalf("got %v, want the sentinel to survive", err)
	}
	if snapshot != nil {
		t.Fatalf("got %+v, want no snapshot on error", snapshot)
	}
}

func TestDescribeExitedProcess(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^$")
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	row := ProcessInfo{PID: int32(child.Process.Pid), Listeners: []Listener{{IP: "127.0.0.1", Port: 3000}}}
	if _, err := describeProcess(context.Background(), row); !errors.Is(err, process.ErrorProcessNotRunning) {
		t.Fatalf("got %v, want process not running", err)
	}
}

func TestCollectFindsOwnListener(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := uint32(listener.Addr().(*net.TCPAddr).Port)
	snapshot, err := Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// The package contract guarantees this, and the check keeps the linter from
	// having to assume it.
	if snapshot == nil {
		t.Fatal("a nil snapshot with a nil error violates the return contract")
	}
	for _, row := range snapshot.Processes {
		if row.PID == int32(os.Getpid()) && slices.Contains(row.Listeners, Listener{IP: "127.0.0.1", Port: port}) {
			if row.Name == "" || row.Cwd == "" || row.Project == "" || row.Command == "" || row.StartedAt == nil || row.PPID == nil {
				t.Fatal("own process metadata is incomplete")
			}
			if *row.PPID != int32(os.Getppid()) || row.ParentName == "" {
				t.Fatalf("parent metadata is incomplete: PID %d, name %q", *row.PPID, row.ParentName)
			}
			if snapshot.CollectedAt <= 0 {
				t.Fatal("missing collection timestamp")
			}
			return
		}
	}
	t.Fatalf("own listener was not found: PID %d, port %d", os.Getpid(), port)
}
