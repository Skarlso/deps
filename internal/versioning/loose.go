package versioning

import (
	"strconv"
	"strings"

	msemver "github.com/Masterminds/semver/v3"

	"git.jakob-moeller.cloud/jakobmoellerdev/deps/internal/dep"
)

// maxLooseParts is the most numeric parts a loose version may have (Renovate: 6).
const maxLooseParts = 6

// looseVersion is Renovate's loose versioning: 1–6 numeric dot-separated parts with an
// optional "v" prefix and an optional suffix after "-" or "_", which counts as a prerelease
// (1877.25, v1.2.3.4, 2.0-rc1, 5_beta). The parts live in an array, so parsing allocates
// nothing.
type looseVersion struct {
	parts  [maxLooseParts]int
	n      int    // number of parts
	v      bool   // "v" prefix
	suffix string // prerelease after the separator; "" = release
}

func parseLoose(s string) (looseVersion, bool) {
	var l looseVersion
	if s != "" && s[0] == 'v' {
		l.v, s = true, s[1:]
	}
	for {
		i := 0
		for i < len(s) && '0' <= s[i] && s[i] <= '9' {
			i++
		}
		if i == 0 || l.n == maxLooseParts {
			return looseVersion{}, false
		}
		v, err := strconv.Atoi(s[:i])
		if err != nil {
			return looseVersion{}, false // overflow
		}
		l.parts[l.n], l.n, s = v, l.n+1, s[i:]
		if s == "" {
			return l, true
		}
		switch s[0] {
		case '.':
			s = s[1:]
		case '-', '_':
			if !validSuffix(s[1:]) {
				return looseVersion{}, false
			}
			l.suffix = s[1:]
			return l, true
		default:
			return looseVersion{}, false
		}
	}
}

// validSuffix reports whether s is a non-empty run of [0-9A-Za-z._+-].
func validSuffix(s string) bool {
	return s != "" && !strings.ContainsFunc(s, func(r rune) bool {
		return !('0' <= r && r <= '9' || 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || strings.ContainsRune("._+-", r))
	})
}

// looseCompat requires the current "v" prefix style, rejects prereleases unless cur is one,
// and rejects a candidate that only writes the current version with more parts (1.2 → 1.2.0,
// which compareLoose orders above it so the order stays total): that is no update, and the
// schemes with a precision of their own (tag, docker, cargo, pep440, conda) reject it too.
func looseCompat(cur, next looseVersion) string {
	switch {
	case next.v != cur.v:
		return "prefix"
	case next.suffix != "" && cur.suffix == "":
		return "prerelease"
	case next.n != cur.n && next.parts == cur.parts && next.suffix == cur.suffix:
		return "precision"
	}
	return ""
}

// compareLoose orders the parts numerically (a missing part counts as 0), then a release
// above its prereleases, prereleases as Renovate's loose versioning compares their suffixes,
// and finally fewer parts below more (1.2 < 1.2.0) so the order is total.
func compareLoose(a, b looseVersion) int {
	for i := range maxLooseParts {
		if a.parts[i] != b.parts[i] {
			if a.parts[i] < b.parts[i] {
				return -1
			}
			return 1
		}
	}
	if c := compareSuffixNumeric(a.suffix, b.suffix); c != 0 {
		return c
	}
	return a.n - b.n
}

// compareSuffixNumeric orders prerelease text as Renovate's loose and generic versioning do
// with localeCompare(…, {numeric: true}): "" (a release) is highest, runs of digits compare by
// value (rc9 below rc10, which a bytewise comparison gets wrong) and the rest case-insensitively
// (Beta next to beta), with the byte order breaking a tie so the order stays total.
func compareSuffixNumeric(x, y string) int {
	switch {
	case x == y:
		return 0
	case x == "":
		return 1
	case y == "":
		return -1
	}
	for i, j := 0, 0; i < len(x) && j < len(y); {
		if isDigit(x[i]) && isDigit(y[j]) {
			p, q := i, j
			for i < len(x) && isDigit(x[i]) {
				i++
			}
			for j < len(y) && isDigit(y[j]) {
				j++
			}
			// Leading zeros do not change the value, only the tie broken below.
			if c := compareNum(strings.TrimLeft(x[p:i], "0"), strings.TrimLeft(y[q:j], "0")); c != 0 {
				return c
			}
			continue
		}
		if a, b := lowerASCII(x[i]), lowerASCII(y[j]); a != b {
			if a < b {
				return -1
			}
			return 1
		}
		i, j = i+1, j+1
	}
	return strings.Compare(x, y)
}

// looseKind classifies by position: part 0 major, part 1 minor, any later part or the
// suffix patch.
func looseKind(cur, next looseVersion) dep.UpdateType {
	switch {
	case cur.parts[0] != next.parts[0]:
		return dep.Major
	case cur.parts[1] != next.parts[1]:
		return dep.Minor
	}
	return dep.Patch
}

// semver builds MAJOR.MINOR.PATCH[-suffix] from the first three parts.
func (l looseVersion) semver() *msemver.Version {
	return msemver.New(uint64(l.parts[0]), uint64(l.parts[1]), uint64(l.parts[2]), l.suffix, "")
}

func (l looseVersion) major() string { return strconv.Itoa(l.parts[0]) }
