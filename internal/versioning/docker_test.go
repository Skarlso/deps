package versioning

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The regexp implementation parseDocker replaced; kept as the reference for parity tests. It
// mirrors Renovate's docker versioning: the tag is split at the first "-" and the part before
// it must match the version pattern.
var (
	refDockerRe     = regexp.MustCompile(`^(v?)(\d+(?:\.\d+)*)(\w*)$`)
	refCommitHashRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

func refParseDocker(s string) (dockerTag, bool) {
	if refCommitHashRe.MatchString(s) && strings.ContainsAny(s, "abcdef") {
		return dockerTag{}, false
	}
	ver, suffix, _ := strings.Cut(s, "-")
	m := refDockerRe.FindStringSubmatch(ver)
	if m == nil {
		return dockerTag{}, false
	}
	t := dockerTag{prefix: m[1], pre: m[3], suffix: suffix}
	for p := range strings.SplitSeq(m[2], ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return dockerTag{}, false
		}
		t.parts = append(t.parts, n)
	}
	return t, true
}

func checkDockerParity(t *testing.T, s string) {
	t.Helper()
	want, wok := refParseDocker(s)
	got, gok := parseDocker(s)
	if wok != gok || want.prefix != got.prefix || want.pre != got.pre || want.suffix != got.suffix || !slices.Equal(want.parts, got.parts) {
		t.Errorf("parseDocker(%q) = %+v, %v; regexp = %+v, %v", s, got, gok, want, wok)
	}
}

var dockerCorpus = []string{
	"", "v", "vv1", "V1", "1", "v1", "01", "1.2", "1.2.", "1..2", ".1", "1.2.3", "v1.2.3-alpine",
	"1.2.3\n", "1\nfoo", "\n1", "1.2-rc.1", "1.x", "1.2.3.4.5.6", "latest", "-1", "+1", "1e5",
	"99999999999999999999", "1.99999999999999999999", "9223372036854775807", "9223372036854775808",
	"3a22eba", "3a22eb", "1234567", "abcdef0", "ABCDEF0", "3a22ebag", "0123456789abcdef0123456789abcdef01234567",
	"0123456789abcdef0123456789abcdef012345678", "1a2b3c4d", "1.2.3a", "v1.0\xff", "1\xff", "١٢٣", "1.٢",
	"20240101", "2024.01.01-r1", "v1.2.3+build", "1.2.3_4", "1 2", "1.2 ", "1.\n", "v\n1",
}

func TestParseDockerParity(t *testing.T) {
	for _, s := range dockerCorpus {
		checkDockerParity(t, s)
	}
	for _, s := range benchDockerTags() {
		checkDockerParity(t, s)
	}
}

func TestParseDocker(t *testing.T) {
	for _, tc := range []struct {
		in                  string
		ok                  bool
		prefix, pre, suffix string
		parts               []int
	}{
		{in: "v1.2.3-alpine", ok: true, prefix: "v", suffix: "alpine", parts: []int{1, 2, 3}},
		{in: "3.14.0rc2-slim", ok: true, pre: "rc2", suffix: "slim", parts: []int{3, 14, 0}},
		{in: "3.14.10-slim", ok: true, suffix: "slim", parts: []int{3, 14, 10}},
		{in: "1.2-alpine-3.20", ok: true, suffix: "alpine-3.20", parts: []int{1, 2}},
		{in: "1.2.", ok: false},
		{in: "1..2", ok: false},
		{in: "1234567", ok: true, parts: []int{1234567}},
		{in: "1a2b3c4", ok: false},
		{in: "1a2b3c", ok: true, pre: "a2b3c", parts: []int{1}},
		{in: "1.2\n", ok: false},
		{in: "1.99999999999999999999", ok: false},
		{in: "vv1", ok: false},
	} {
		got, ok := parseDocker(tc.in)
		if ok != tc.ok || got.prefix != tc.prefix || got.pre != tc.pre || got.suffix != tc.suffix || !slices.Equal(got.parts, tc.parts) {
			t.Errorf("parseDocker(%q) = %+v, %v", tc.in, got, ok)
		}
	}
}

// TestDockerPrereleaseUpdate covers Renovate's docker parsing of a prerelease tag: the
// prerelease is not part of the suffix, so a prerelease tag can update to the stable release.
func TestDockerPrereleaseUpdate(t *testing.T) {
	s, ok := Get("docker")
	if !ok {
		t.Fatal("no docker scheme")
	}
	cur, stable := parseMust(t, "3.14.0rc2-slim"), parseMust(t, "3.14.10-slim")
	if why := dockerCompat(cur, stable); why != "" {
		t.Errorf("dockerCompat(3.14.0rc2-slim, 3.14.10-slim) = %q, want compatible", why)
	}
	if why := dockerCompat(stable, cur); why != "prerelease" {
		t.Errorf("dockerCompat(3.14.10-slim, 3.14.0rc2-slim) = %q, want prerelease", why)
	}
	if why := dockerCompat(cur, parseMust(t, "3.15.0rc1-slim")); why != "prerelease" {
		t.Errorf("dockerCompat(3.14.0rc2-slim, 3.15.0rc1-slim) = %q, want prerelease", why)
	}
	tags := []string{"3.14.0rc1-slim", "3.14.0rc2-slim", "3.14.0rc3-slim", "3.14.10-slim", "3.14.10-alpine", "3.15.0rc1-slim", "latest"}
	if got, ok := s.Best("3.14.0rc2-slim", tags, nil, nil); !ok || got != "3.14.10-slim" {
		t.Errorf("Best = %q, %v; want 3.14.10-slim", got, ok)
	}
	// A release outranks its prereleases, unlike a bytewise suffix comparison.
	vs := []string{"3.13.0rc1", "3.13.0"}
	s.Sort(vs)
	if !slices.Equal(vs, []string{"3.13.0rc1", "3.13.0"}) {
		t.Errorf("Sort = %v, want [3.13.0rc1 3.13.0]", vs)
	}
}

func parseMust(t *testing.T, s string) dockerTag {
	t.Helper()
	v, ok := parseDocker(s)
	if !ok {
		t.Fatalf("parseDocker(%q) failed", s)
	}
	return v
}

func FuzzParseDocker(f *testing.F) {
	for _, s := range dockerCorpus {
		f.Add(s)
	}
	f.Fuzz(checkDockerParity)
}

func benchDockerTags() []string {
	suffixes := []string{"", "-alpine", "-slim", "-bookworm", "-alpine3.20", "-rc.1", ".windowsservercore"}
	tags := make([]string, 0, 2100)
	for major := range 10 {
		for minor := range 10 {
			for patch := range 10 {
				s := suffixes[(major+minor+patch)%len(suffixes)]
				tags = append(tags, fmt.Sprintf("%d.%d.%d%s", major, minor, patch, s))
			}
			tags = append(tags, fmt.Sprintf("v%d.%d", major, minor), fmt.Sprintf("%d.%d-%07x", major, minor, major*1000+minor))
		}
	}
	for i := range 200 {
		tags = append(tags, fmt.Sprintf("%07x", 0xa000000+i*7919), "latest", fmt.Sprintf("sha-%x", i))
	}
	return tags
}

func BenchmarkParseDocker(b *testing.B) {
	tags := benchDockerTags()
	b.ReportAllocs()
	for b.Loop() {
		for _, s := range tags {
			parseDocker(s)
		}
	}
}

func BenchmarkDockerBest(b *testing.B) {
	s, ok := Get("docker")
	if !ok {
		b.Fatal("no docker scheme")
	}
	tags := benchDockerTags()
	b.ReportAllocs()
	for b.Loop() {
		s.Best("1.2.0-alpine", tags, nil, nil)
	}
}
