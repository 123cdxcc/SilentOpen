package update

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

// Version is a semantic version reduced to what comparing two releases needs.
type Version struct {
	Major int
	Minor int
	Patch int
	// Pre holds the pre-release identifiers of "1.2.3-rc.1" without the dash.
	// An empty Pre marks a release, which outranks any of its pre-releases.
	Pre string
}

// ParseVersion parses "1.2.3", "v1.2.3", "1.2" and "1.2.3-rc.1". Missing
// numeric segments count as zero and build metadata after "+" is ignored, so a
// tag like "v1.2.3+build.7" is accepted. An unparsable version is an error: the
// caller decides whether that means "no version to compare" or a broken tag.
func ParseVersion(text string) (Version, error) {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "v"), "V")
	if cut, _, found := strings.Cut(trimmed, "+"); found {
		trimmed = cut
	}
	core, pre, _ := strings.Cut(trimmed, "-")
	segments := strings.Split(core, ".")
	if len(segments) > 3 {
		return Version{}, fmt.Errorf("版本号 %q 的数字段过多", text)
	}
	parsed := Version{Pre: pre}
	numbers := []*int{&parsed.Major, &parsed.Minor, &parsed.Patch}
	for i, segment := range segments {
		value, err := strconv.Atoi(segment)
		if err != nil || value < 0 {
			return Version{}, fmt.Errorf("版本号 %q 含非数字段 %q", text, segment)
		}
		*numbers[i] = value
	}
	return parsed, nil
}

// String renders the version in its canonical form, without a leading "v".
func (v Version) String() string {
	text := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		text += "-" + v.Pre
	}
	return text
}

// Compare returns -1 when a is older than b, 0 when they are equal, and 1 when
// a is newer. Pre-release precedence follows semantic versioning 2.0.0 §11, so
// 1.2.0-rc.1 sorts below 1.2.0 and 1.2.0-rc.2 sorts above 1.2.0-rc.1.
func Compare(a, b Version) int {
	if c := cmp.Compare(a.Major, b.Major); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Minor, b.Minor); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Patch, b.Patch); c != 0 {
		return c
	}
	return comparePreRelease(a.Pre, b.Pre)
}

// comparePreRelease compares two pre-release strings. An empty string is a
// release, which always outranks a pre-release of the same numbers.
func comparePreRelease(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(left), len(right)) {
		if c := comparePreReleaseIdentifier(left[i], right[i]); c != 0 {
			return c
		}
	}
	// "1.0.0-alpha" sorts below "1.0.0-alpha.1": a shorter equal prefix loses.
	return cmp.Compare(len(left), len(right))
}

// comparePreReleaseIdentifier compares one dot-separated pre-release
// identifier: numeric ones compare numerically and outrank alphanumeric ones.
func comparePreReleaseIdentifier(a, b string) int {
	left, leftErr := strconv.Atoi(a)
	right, rightErr := strconv.Atoi(b)
	switch {
	case leftErr == nil && rightErr == nil:
		return cmp.Compare(left, right)
	case leftErr == nil:
		return -1
	case rightErr == nil:
		return 1
	}
	return strings.Compare(a, b)
}
