package icondata

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "icon.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(FileURL(path), "data:image/png;base64,") {
		t.Fatal("valid PNG was not returned")
	}
	for _, fixture := range []struct {
		name, data string
		valid      bool
	}{
		{"icon.svg", `<svg xmlns="http://www.w3.org/2000/svg"/>`, true},
		{"fake.svg", `<html/>`, false},
		{"fake.png", "invalid", false},
		{"huge.png", strings.Repeat("x", MaxBytes+1), false},
	} {
		path := filepath.Join(dir, fixture.name)
		if err := os.WriteFile(path, []byte(fixture.data), 0600); err != nil {
			t.Fatal(err)
		}
		if got := FileURL(path); (got != "") != fixture.valid {
			t.Fatalf("unexpected result for %s", fixture.name)
		}
	}
	if got := FileURL(filepath.Join(dir, "does-not-exist")); got != "" {
		t.Fatalf("missing file must have no data URL, got %q", got)
	}
}
