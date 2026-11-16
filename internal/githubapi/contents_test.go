package githubapi

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		switch r.URL.RequestURI() {
		case "/repos/o/r/contents/dir/a%20b.json?ref=v1%2B1":
			// GitHub wraps base64 content at 60 columns.
			_, _ = io.WriteString(w, `{"type":"file","encoding":"base64","content":"eyJleHRlbmRzIjpb\nXX0=\n"}`)
		case "/repos/o/r/contents/dir":
			_, _ = io.WriteString(w, `[{"type":"file"}]`)
		case "/repos/o/r/contents/sub":
			_, _ = io.WriteString(w, `{"type":"submodule"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New(srv.Client(), tokenAuth(), make(chan struct{}, 1), WithAPI(srv.URL, srv.URL+"/graphql"))

	b, err := c.File(t.Context(), oR, "dir/a b.json", "v1+1")
	if err != nil || string(b) != `{"extends":[]}` {
		t.Errorf("File = %q, %v", b, err)
	}
	if _, err := c.File(t.Context(), oR, "missing.json", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: err = %v, want ErrNotFound", err)
	}
	if _, err := c.File(t.Context(), oR, "sub", ""); err == nil || err.Error() != "o/r: sub is not a file" {
		t.Errorf("submodule: err = %v", err)
	}
	if _, err := c.File(t.Context(), oR, "dir", ""); err == nil {
		t.Error("directory listing: want error")
	}
}

// A file over 1 MB (a shared Renovate config, say) comes back from the contents API with an
// empty content and `"encoding": "none"`; only the raw media type carries it.
func TestFileOverOneMegabyte(t *testing.T) {
	want := strings.Repeat("{\"extends\":[]}\n", 100_000)
	var accepts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/repos/o/r/contents/big.json?ref=main" {
			http.NotFound(w, r)
			return
		}
		accepts = append(accepts, r.Header.Get("Accept"))
		if r.Header.Get("Accept") == "application/vnd.github.raw+json" {
			_, _ = io.WriteString(w, want)
			return
		}
		_, _ = io.WriteString(w, `{"type":"file","encoding":"none","size":1500000,"content":""}`)
	}))
	defer srv.Close()
	c := New(srv.Client(), tokenAuth(), make(chan struct{}, 1), WithAPI(srv.URL, srv.URL+"/graphql"))

	b, err := c.File(t.Context(), oR, "big.json", "main")
	if err != nil || string(b) != want {
		t.Fatalf("File = %d bytes, %v, want %d bytes", len(b), err, len(want))
	}
	if len(accepts) != 2 || accepts[0] != "application/vnd.github+json" || accepts[1] != "application/vnd.github.raw+json" {
		t.Errorf("Accept headers = %q", accepts)
	}
}

func TestDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/repos/o/r/contents/steps/script":
			_, _ = io.WriteString(w, `[{"name":"1.0.0","path":"steps/script/1.0.0","type":"dir"},{"name":"assets","type":"dir"},{"name":"README.md","type":"file"}]`)
		case "/repos/o/r/contents/steps/file?ref=main":
			_, _ = io.WriteString(w, `{"type":"file","encoding":"base64","content":""}`)
		case "/repos/o/r/contents/steps/bad":
			_, _ = io.WriteString(w, `[{"name":1}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New(srv.Client(), nil, make(chan struct{}, 1), WithAPI(srv.URL, srv.URL+"/graphql"))

	es, err := c.Dir(t.Context(), oR, "steps/script", "")
	if err != nil || len(es) != 3 || es[0] != (Entry{Name: "1.0.0", Type: "dir"}) || es[2] != (Entry{Name: "README.md", Type: "file"}) {
		t.Errorf("Dir = %+v, %v", es, err)
	}
	if _, err := c.Dir(t.Context(), oR, "steps/missing", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: err = %v, want ErrNotFound", err)
	}
	if _, err := c.Dir(t.Context(), oR, "steps/file", "main"); err == nil || err.Error() != "o/r: steps/file is not a directory" {
		t.Errorf("file: err = %v", err)
	}
	if _, err := c.Dir(t.Context(), oR, "steps/bad", ""); err == nil {
		t.Error("malformed: want error")
	}
}
