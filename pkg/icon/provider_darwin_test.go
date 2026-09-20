package icon

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNestedAppBundles(t *testing.T) {
	exe := "/Applications/Editor.app/Contents/Frameworks/Helper.app/Contents/MacOS/Helper"
	want := []string{"/Applications/Editor.app/Contents/Frameworks/Helper.app", "/Applications/Editor.app"}
	if got := appBundles(exe); !reflect.DeepEqual(got, want) {
		t.Fatalf("appBundles() = %v, want %v", got, want)
	}
	if got := appBundles("/usr/local/bin/node"); len(got) != 0 {
		t.Fatal("CLI executable must not be associated with an unrelated app")
	}
}

func TestReadSystemApplicationIcon(t *testing.T) {
	exe := "/System/Applications/Utilities/Terminal.app/Contents/MacOS/Terminal"
	if _, err := os.Stat(exe); err != nil {
		t.Skip("Terminal.app is not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if got := NewPlatformProvider().Icon(ctx, exe); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("could not read the installed Terminal icon (context: %v)", ctx.Err())
	}
}
