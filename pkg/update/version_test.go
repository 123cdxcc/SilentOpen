package update

import "testing"

func TestParseVersionAcceptsReleaseTags(t *testing.T) {
	cases := map[string]Version{
		"1.2.3":         {Major: 1, Minor: 2, Patch: 3},
		"v1.2.3":        {Major: 1, Minor: 2, Patch: 3},
		"V1.2.3":        {Major: 1, Minor: 2, Patch: 3},
		"  v1.2.3  ":    {Major: 1, Minor: 2, Patch: 3},
		"1.2":           {Major: 1, Minor: 2},
		"2":             {Major: 2},
		"1.2.3-rc.1":    {Major: 1, Minor: 2, Patch: 3, Pre: "rc.1"},
		"v1.2.3+build7": {Major: 1, Minor: 2, Patch: 3},
	}
	for text, want := range cases {
		got, err := ParseVersion(text)
		if err != nil {
			t.Fatalf("ParseVersion(%q) returned %v", text, err)
		}
		if got != want {
			t.Errorf("ParseVersion(%q) = %+v, want %+v", text, got, want)
		}
	}
}

func TestParseVersionRejectsNonVersions(t *testing.T) {
	for _, text := range []string{"", "dev", "v", "1.2.3.4", "1.x", "1..2", "-1.2.3", "latest"} {
		if got, err := ParseVersion(text); err == nil {
			t.Errorf("ParseVersion(%q) = %+v, want an error", text, got)
		}
	}
}

func TestVersionStringRoundTrips(t *testing.T) {
	for _, text := range []string{"1.2.3", "0.1.0", "1.2.3-rc.1"} {
		parsed, err := ParseVersion(text)
		if err != nil {
			t.Fatalf("ParseVersion(%q) returned %v", text, err)
		}
		if got := parsed.String(); got != text {
			t.Errorf("ParseVersion(%q).String() = %q", text, got)
		}
	}
}

func TestCompareOrdersVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3", "v1.2.3", 0},
		{"1.2.3", "1.2.4", -1},
		{"1.2.4", "1.2.3", 1},
		{"1.3.0", "1.2.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.2", "1.2.0", 0},
		{"0.0.0", "0.0.1", -1},
		// A pre-release sorts below the release it leads to.
		{"1.2.0-rc.1", "1.2.0", -1},
		{"1.2.0", "1.2.0-rc.1", 1},
		{"1.2.0-rc.1", "1.2.0-rc.2", -1},
		{"1.2.0-rc.10", "1.2.0-rc.9", 1},
		{"1.2.0-rc.1", "1.2.0-beta.9", 1},
		{"1.2.0-alpha", "1.2.0-alpha.1", -1},
		// A pre-release of a lower number still loses to the higher number.
		{"1.2.1-rc.1", "1.2.0", 1},
	}
	for _, testCase := range cases {
		a, err := ParseVersion(testCase.a)
		if err != nil {
			t.Fatalf("ParseVersion(%q) returned %v", testCase.a, err)
		}
		b, err := ParseVersion(testCase.b)
		if err != nil {
			t.Fatalf("ParseVersion(%q) returned %v", testCase.b, err)
		}
		if got := Compare(a, b); got != testCase.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", testCase.a, testCase.b, got, testCase.want)
		}
		if got := Compare(b, a); got != -testCase.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", testCase.b, testCase.a, got, -testCase.want)
		}
	}
}
