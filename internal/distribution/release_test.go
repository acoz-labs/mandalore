package distribution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type releaseFixture struct {
	client   *ReleaseClient
	body     map[string][]byte
	release  map[string]any
	assets   []map[string]any
	manifest Manifest
	requests []string
	redirect map[string]string
	status   map[string]int
}

func newReleaseFixture(t *testing.T) *releaseFixture {
	t.Helper()
	dir, m, _ := candidateFixture(t)
	f := &releaseFixture{body: map[string][]byte{}, redirect: map[string]string{}, status: map[string]int{}, manifest: m}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		id := int64(100 + i)
		f.assets = append(f.assets, map[string]any{"id": id, "name": e.Name(), "state": "uploaded", "size": len(b), "digest": "sha256:" + Digest(b)})
		f.body[fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", id)] = b
	}
	f.release = map[string]any{"id": int64(42), "tag_name": "v1.0.0", "draft": false, "prerelease": false, "immutable": true, "published_at": "2026-09-14T00:00:00Z", "assets": f.assets}
	f.body["/repos/acoz-labs/mandalore/git/ref/tags/v1.0.0"] = []byte(`{"ref":"refs/tags/v1.0.0","object":{"type":"commit","sha":"` + m.SourceCommit + `"}}`)
	f.client = newReleaseClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		f.requests = append(f.requests, r.URL.String())
		if r.Method != "GET" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("public release read borrowed authentication or mutated external state")
		}
		if !allowedReleaseURL(r.URL) {
			t.Fatal("request contacted an untrusted host")
		}
		code := f.status[r.URL.Path]
		if code == 0 {
			code = 200
		}
		header := http.Header{}
		if location := f.redirect[r.URL.Path]; location != "" {
			code = 302
			header.Set("Location", location)
		}
		b, ok := f.body[r.URL.Path]
		if !ok && r.URL.Host == "api.github.com" && (r.URL.Path == "/repos/acoz-labs/mandalore/releases/latest" || r.URL.Path == "/repos/acoz-labs/mandalore/releases/tags/v1.0.0" || r.URL.Path == "/repos/acoz-labs/mandalore/releases/42") {
			var err error
			b, err = json.Marshal(f.release)
			if err != nil {
				t.Fatal(err)
			}
			ok = true
		}
		if !ok && code == 200 {
			code = 404
		}
		return &http.Response{StatusCode: code, Header: header, Body: io.NopCloser(bytes.NewReader(b)), ContentLength: int64(len(b)), Request: r}, nil
	}))
	return f
}

func TestReleaseInspectionPinsCommitManifestAndAssets(t *testing.T) {
	for _, version := range []string{"", "1.0.0"} {
		f := newReleaseFixture(t)
		r, err := f.client.Inspect(context.Background(), version)
		if err != nil {
			t.Fatal(err)
		}
		if r.ID != 42 || r.Manifest.Manifest.SourceCommit != f.manifest.SourceCommit || len(r.Assets) != 8 || r.Manifest.SHA256 == "" {
			t.Fatalf("incomplete pinned inspection: %+v", r)
		}
		for _, a := range r.Assets {
			if a.Name == "manifest.json" && a.SHA256 != r.Manifest.SHA256 {
				t.Fatal("manifest identity mismatch")
			}
		}
	}
}

func TestReleaseInspectionRejectsUntrustedOrIncompleteMetadata(t *testing.T) {
	cases := map[string]func(*releaseFixture){
		"missing draft flag":   func(f *releaseFixture) { delete(f.release, "draft") },
		"null prerelease flag": func(f *releaseFixture) { f.release["prerelease"] = nil },
		"duplicate JSON": func(f *releaseFixture) {
			b, _ := json.Marshal(f.release)
			f.body["/repos/acoz-labs/mandalore/releases/latest"] = append([]byte(`{"id":42,`), b[1:]...)
		},
		"oversized metadata": func(f *releaseFixture) {
			f.body["/repos/acoz-labs/mandalore/releases/latest"] = bytes.Repeat([]byte(" "), maxReleaseMetadata+1)
		},
		"draft":             func(f *releaseFixture) { f.release["draft"] = true },
		"mutable":           func(f *releaseFixture) { f.release["immutable"] = false },
		"unpublished":       func(f *releaseFixture) { f.release["published_at"] = "" },
		"prerelease latest": func(f *releaseFixture) { f.release["prerelease"] = true },
		"tag mismatch":      func(f *releaseFixture) { f.release["tag_name"] = "v2.0.0" },
		"wrong commit": func(f *releaseFixture) {
			f.body["/repos/acoz-labs/mandalore/git/ref/tags/v1.0.0"] = []byte(`{"ref":"refs/tags/v1.0.0","object":{"type":"commit","sha":"` + strings.Repeat("f", 40) + `"}}`)
		},
		"missing asset":   func(f *releaseFixture) { f.release["assets"] = f.assets[1:] },
		"duplicate asset": func(f *releaseFixture) { f.assets[1] = f.assets[0] },
		"asset ID":        func(f *releaseFixture) { f.assets[0]["id"] = 0 },
		"asset state":     func(f *releaseFixture) { f.assets[0]["state"] = "new" },
		"asset digest":    func(f *releaseFixture) { f.assets[0]["digest"] = "invalid" },
		"asset size":      func(f *releaseFixture) { f.assets[0]["size"] = 1 },
		"asset corruption": func(f *releaseFixture) {
			for path, b := range f.body {
				if strings.Contains(path, "/assets/") {
					f.body[path] = append(b, 'x')
				}
			}
		},
		"redirect host": func(f *releaseFixture) {
			f.redirect["/repos/acoz-labs/mandalore/releases/latest"] = "https://untrusted.example/data"
		},
		"redirect http": func(f *releaseFixture) {
			f.redirect["/repos/acoz-labs/mandalore/releases/latest"] = "http://api.github.com/data"
		},
		"redirect loop": func(f *releaseFixture) {
			f.redirect["/repos/acoz-labs/mandalore/releases/latest"] = "https://api.github.com/repos/acoz-labs/mandalore/releases/latest"
		},
		"404":        func(f *releaseFixture) { f.status["/repos/acoz-labs/mandalore/releases/latest"] = 404 },
		"rate limit": func(f *releaseFixture) { f.status["/repos/acoz-labs/mandalore/releases/latest"] = 429 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			f := newReleaseFixture(t)
			change(f)
			if _, err := f.client.Inspect(context.Background(), ""); err == nil {
				t.Fatal("untrusted or incomplete release accepted")
			}
		})
	}
}

func TestReleaseAnnotatedTagAndExplicitPrerelease(t *testing.T) {
	f := newReleaseFixture(t)
	f.release["prerelease"] = true
	tagSHA := strings.Repeat("e", 40)
	f.body["/repos/acoz-labs/mandalore/git/ref/tags/v1.0.0"] = []byte(`{"ref":"refs/tags/v1.0.0","object":{"type":"tag","sha":"` + tagSHA + `"}}`)
	f.body["/repos/acoz-labs/mandalore/git/tags/"+tagSHA] = []byte(`{"sha":"` + tagSHA + `","object":{"type":"commit","sha":"` + f.manifest.SourceCommit + `"}}`)
	r, err := f.client.Inspect(context.Background(), "1.0.0")
	if err != nil || !r.Prerelease {
		t.Fatalf("explicit published prerelease failed: %v", err)
	}
}

func TestReleaseDownloadRevalidatesPinnedIdentityAndChecksBytes(t *testing.T) {
	f := newReleaseFixture(t)
	r, err := f.client.Inspect(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	a, err := r.Manifest.Manifest.Binary("linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := f.client.DownloadAsset(context.Background(), r, a.Name, &out); err != nil {
		t.Fatal(err)
	}
	if Digest(out.Bytes()) != a.SHA256 {
		t.Fatal("downloaded wrong bytes")
	}
	f.release["id"] = int64(43)
	out.Reset()
	if err := f.client.DownloadAsset(context.Background(), r, a.Name, &out); err == nil || out.Len() != 0 {
		t.Fatal("release identity changed but download proceeded")
	}
	f.release["id"] = int64(42)
	for _, remote := range r.Assets {
		if remote.Name == a.Name {
			f.body[fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", remote.ID)] = []byte("corrupted")
		}
	}
	if err := f.client.DownloadAsset(context.Background(), r, a.Name, &out); err == nil {
		t.Fatal("corrupted payload download succeeded")
	}
}

func TestReleaseInspectionCancellationAndInvalidVersionAvoidRequests(t *testing.T) {
	f := newReleaseFixture(t)
	if _, err := f.client.Inspect(context.Background(), "../latest"); err == nil {
		t.Fatal("invalid version accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.client.Inspect(ctx, ""); err == nil {
		t.Fatal("cancelled request succeeded")
	}
	if len(f.requests) != 0 {
		t.Fatal("invalid/cancelled inspection contacted network")
	}
}

func TestReleaseDownloadAllowsOnlyKnownCDNWithoutCredentials(t *testing.T) {
	t.Setenv("GH_TOKEN", "SYNTHETIC_TOKEN_MUST_NOT_BE_USED")
	f := newReleaseFixture(t)
	r, err := f.client.Inspect(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := r.Manifest.Manifest.Binary("darwin", "arm64")
	for _, remote := range r.Assets {
		if remote.Name == a.Name {
			path := fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", remote.ID)
			f.redirect[path] = "https://release-assets.githubusercontent.com/synthetic"
			f.body["/synthetic"] = f.body[path]
		}
	}
	var b bytes.Buffer
	if err := f.client.DownloadAsset(context.Background(), r, a.Name, &b); err != nil || Digest(b.Bytes()) != a.SHA256 {
		t.Fatalf("valid CDN download failed: %v", err)
	}
}

type cancelReleaseRead struct{ cancel context.CancelFunc }

func (r cancelReleaseRead) Read([]byte) (int, error) { r.cancel(); return 0, context.Canceled }

func TestReleaseBodyLimitsAndMidstreamCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newReleaseClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, ContentLength: -1, Body: io.NopCloser(cancelReleaseRead{cancel}), Request: r}, nil
	}))
	if _, err := c.get(ctx, releaseAPI+"/synthetic", "application/octet-stream", 1024, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("midstream cancellation misreported: %v", err)
	}
	c = newReleaseClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, ContentLength: -1, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 2048))), Request: r}, nil
	}))
	n, err := c.get(context.Background(), releaseAPI+"/synthetic", "application/octet-stream", 1024, io.Discard)
	if err == nil || n != 1025 {
		t.Fatalf("unknown-length body was not bounded: %d %v", n, err)
	}
}
