package config

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	day = 24 * time.Hour
	// month is Renovate's month (lib/util/pretty-time.ts rewrites "<n> months" to "<n*30>
	// days" before parsing), year the one currentAge and abandonmentThreshold document (the
	// ms package Renovate parses the rest with counts 365.25 days).
	month = 30 * day
	year  = 365 * day
)

// agePartRe matches one part of an age: a number, decimals allowed, and a unit. Renovate
// splits an age into such parts and sums them (lib/util/pretty-time.ts toMs, "1 day 12 hours").
var agePartRe = regexp.MustCompile(`^(\d+(?:\.\d+)?|\.\d+)\s*([a-zA-Z]+)\s*`)

// ageUnits are the units Renovate accepts, lower-cased: those of the ms package it parses
// ages with, plus the months of its own pre-pass. ms has no "mo".
var ageUnits = map[string]time.Duration{
	"ms": time.Millisecond, "msec": time.Millisecond, "msecs": time.Millisecond,
	"millisecond": time.Millisecond, "milliseconds": time.Millisecond,
	"s": time.Second, "sec": time.Second, "secs": time.Second, "second": time.Second, "seconds": time.Second,
	"m": time.Minute, "min": time.Minute, "mins": time.Minute, "minute": time.Minute, "minutes": time.Minute,
	"h": time.Hour, "hr": time.Hour, "hrs": time.Hour, "hour": time.Hour, "hours": time.Hour,
	"d": day, "day": day, "days": day,
	"w": 7 * day, "week": 7 * day, "weeks": 7 * day,
	"month": month, "months": month,
	"y": year, "yr": year, "yrs": year, "year": year, "years": year,
}

// errAgeForm marks a string that is no number-and-unit age at all, so ParseAge can fall back
// to a Go duration.
var errAgeForm = errors.New("not an age")

// UnknownAge is the age of a current version whose release time is unknown: no currentAge
// rule matches it.
const UnknownAge time.Duration = -1

// ageRangeRe splits a currentAge into its operator and age ("> 1 year", "<=30 days").
var ageRangeRe = regexp.MustCompile(`^\s*(<=|>=|<|>)\s*(.+?)\s*$`)

// ageRange is a parsed currentAge (Renovate's matchCurrentAge): the age of the current
// version compared with an operator, so "> 1 year" matches versions released over a year ago.
type ageRange struct {
	op  string
	age time.Duration
}

// parseAgeRange parses a currentAge: an operator <, <=, > or >= and an age as for
// minimumReleaseAge.
func parseAgeRange(s string) (*ageRange, error) {
	m := ageRangeRe.FindStringSubmatch(s)
	if m == nil {
		return nil, fmt.Errorf(`invalid age range %q: want an operator <, <=, > or >= and an age, e.g. "> 1 year" or "< 30 days"`, s)
	}
	d, err := ParseAge(m[2])
	if err != nil {
		return nil, fmt.Errorf("invalid age range %q: %w", s, err)
	}
	return &ageRange{op: m[1], age: d}, nil
}

// parseAgeParts parses an age the way Renovate's toMs does: one or more <number><unit> parts,
// summed. It returns errAgeForm when s has another shape.
func parseAgeParts(s string) (time.Duration, error) {
	rest := strings.TrimSpace(s)
	if rest == "" {
		return 0, errAgeForm
	}
	var total float64
	for rest != "" {
		p := agePartRe.FindStringSubmatch(rest)
		if p == nil {
			return 0, errAgeForm
		}
		unit, ok := ageUnit(p[2])
		if !ok {
			return 0, errAgeForm
		}
		n, err := strconv.ParseFloat(p[1], 64)
		if err != nil {
			return 0, errAgeForm
		}
		if total += n * float64(unit); total >= float64(math.MaxInt64) {
			return 0, fmt.Errorf("age %q out of range", s)
		}
		rest = rest[len(p[0]):]
	}
	return time.Duration(total), nil
}

// ageUnit returns the length of unit u. Case is ignored, as in ms, except for a lone "M":
// that is Renovate's month shorthand, while "m" stays minutes.
func ageUnit(u string) (time.Duration, bool) {
	if u == "M" {
		return month, true
	}
	d, ok := ageUnits[strings.ToLower(u)]
	return d, ok
}

// matches reports whether age (>= 0) satisfies the range.
func (a *ageRange) matches(age time.Duration) bool {
	switch a.op {
	case ">":
		return age > a.age
	case ">=":
		return age >= a.age
	case "<":
		return age < a.age
	}
	return age <= a.age
}

// ParseAge parses an age as Renovate does (lib/util/pretty-time.ts): one or more "<n><unit>"
// parts, summed ("1 day 12 hours"), with the units of the ms package (ms, s, m, h, d, w, y
// and their spelled-out forms, case-insensitive) and months ("1 month", "3 M") of 30 days; n
// may have decimals ("0.5 days"). "0" and Go durations ("1h30m") are accepted too, negative
// ages are not.
func ParseAge(s string) (time.Duration, error) {
	if s == "0" {
		return 0, nil
	}
	switch d, err := parseAgeParts(s); {
	case err == nil:
		return d, nil
	case !errors.Is(err, errAgeForm):
		return 0, err
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf(`invalid age %q: want e.g. "3 days", "12h", "90m" or "0"`, s)
	}
	return d, nil
}

// FormatAge renders an age in the largest unit that divides it ("28 days", "12h", "90m"), or
// as a Go duration.
func FormatAge(d time.Duration) string {
	switch day := 24 * time.Hour; {
	case d == 0:
		return "0"
	case d == day:
		return "1 day"
	case d%day == 0:
		return strconv.FormatInt(int64(d/day), 10) + " days"
	case d%time.Hour == 0:
		return strconv.FormatInt(int64(d/time.Hour), 10) + "h"
	case d%time.Minute == 0:
		return strconv.FormatInt(int64(d/time.Minute), 10) + "m"
	default:
		return d.String()
	}
}
