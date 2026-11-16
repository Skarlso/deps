package githubapi

import (
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// Entry is an entry of a directory listing: Type is "file", "dir", "symlink" or "submodule".
type Entry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Dir lists the directory path at ref ("" = default branch) in r; a missing directory wraps
// ErrNotFound. The contents API lists at most 1,000 entries.
func (c *Client) Dir(ctx context.Context, r Repo, path, ref string) ([]Entry, error) {
	var raw jsontext.Value
	if err := c.restDo(ctx, r, contentsPath(r, path, ref), false, false, func(b io.Reader) error {
		return json.UnmarshalRead(b, &raw)
	}); err != nil {
		return nil, err
	}
	if len(raw) == 0 || raw[0] != '[' {
		return nil, fmt.Errorf("%s: %s is not a directory", r.Path(), path)
	}
	var entries []Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("malformed response: %w", err)
	}
	return entries, nil
}

// contentsPath is the contents API path of path at ref ("" = default branch) in r.
func contentsPath(r Repo, path, ref string) string {
	segs := strings.Split(path, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	p := "/repos/" + r.Path() + "/contents/" + strings.Join(segs, "/")
	if ref != "" {
		p += "?ref=" + url.QueryEscape(ref)
	}
	return p
}

// File returns the content of path at ref ("" = default branch) in r; a missing file wraps
// ErrNotFound. A file over 1 MB takes a second request: the contents API leaves its content
// out of the JSON answer (fileJSON), and only the raw media type carries it (fileRaw).
func (c *Client) File(ctx context.Context, r Repo, path, ref string) ([]byte, error) {
	b, large, err := c.fileJSON(ctx, r, path, ref)
	if err != nil || !large {
		return b, err
	}
	return c.fileRaw(ctx, r, path, ref)
}

// fileJSON reads path at ref from the contents API, which answers JSON with the content
// base64 encoded. large reports an answer that leaves the content out, which GitHub sends
// for a file over 1 MB: `"encoding": "none"` with an empty content, up to the endpoint's
// limit of 100 MB. A directory, symlink or submodule is not a file, whatever its size.
func (c *Client) fileJSON(ctx context.Context, r Repo, path, ref string) ([]byte, bool, error) {
	if err := c.acquire(ctx); err != nil {
		return nil, false, err
	}
	defer c.release()
	resp, err := c.restGet(ctx, r, contentsPath(r, path, ref), false)
	if err != nil {
		return nil, false, err
	}
	defer closeBody(resp)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, false, statusError(resp)
	}
	var raw struct {
		Type     string `json:"type"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := json.UnmarshalRead(resp.Body, &raw); err != nil {
		return nil, false, fmt.Errorf("malformed response: %w", err)
	}
	if raw.Type == "file" && raw.Encoding == "none" {
		return nil, true, nil
	}
	if raw.Type != "file" || raw.Encoding != "base64" {
		return nil, false, fmt.Errorf("%s: %s is not a file", r.Path(), path)
	}
	b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(raw.Content, "\n", ""))
	if err != nil {
		return nil, false, fmt.Errorf("malformed response: %w", err)
	}
	return b, false, nil
}

// fileRaw reads path at ref from the contents API with the raw media type, whose answer is
// the file itself rather than JSON. It holds a request slot of its own, so File must not
// hold one.
func (c *Client) fileRaw(ctx context.Context, r Repo, path, ref string) ([]byte, error) {
	if err := c.acquire(ctx); err != nil {
		return nil, err
	}
	defer c.release()
	resp, err := c.restGetAccept(ctx, r, contentsPath(r, path, ref), false, "application/vnd.github.raw+json")
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, statusError(resp)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", r.Path(), path, err)
	}
	return b, nil
}
