package versioning

import "testing"

// Node.js releases an odd major every spring: it is a Current release, never LTS, so the node
// scheme offers it only to a version already on that major.
func TestNodeScheme(t *testing.T) {
	s, ok := Get("node")
	if !ok {
		t.Fatal("no node scheme")
	}
	cands := []string{"24.0.0", "24.11.0", "25.0.0", "25.2.0", "26.0.0", "26.1.0"}
	for _, tt := range []struct{ cur, want string }{
		{"24.0.0", "26.1.0"},  // even majors only
		{"25.0.0", "26.1.0"},  // a newer even major still wins
		{"25.2.0", "26.1.0"},  //
		{"26.0.0", "26.1.0"},  //
		{"23.0.0", "26.1.0"},  //
		{"24.11.0", "26.1.0"}, //
	} {
		if got, ok := s.Best(tt.cur, cands, nil, nil); !ok || got != tt.want {
			t.Errorf("Best(%s) = %q %v, want %q", tt.cur, got, ok, tt.want)
		}
	}
	// Without an even major to move to, an odd major stays on its own line.
	odd := []string{"25.0.0", "25.2.0", "27.0.0"}
	if got, ok := s.Best("25.0.0", odd, nil, nil); !ok || got != "25.2.0" {
		t.Errorf("Best(25.0.0) = %q %v, want 25.2.0", got, ok)
	}
	if _, ok := s.Best("24.0.0", []string{"25.2.0", "27.0.0"}, nil, nil); ok {
		t.Error("an even major moved to an odd one")
	}
	// Everything else is the docker scheme: tags keep their precision and suffix.
	if got, ok := s.Best("24-alpine", []string{"25-alpine", "26-alpine", "26.1.0-alpine", "26"}, nil, nil); !ok || got != "26-alpine" {
		t.Errorf("Best(24-alpine) = %q %v, want 26-alpine", got, ok)
	}
}
