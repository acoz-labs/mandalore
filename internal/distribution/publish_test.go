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

// A simulated provider, never a live release or product-acceptance fixture.
type publisherFixture struct {
	dir, identity                                                                    string
	m                                                                                Manifest
	p                                                                                *Publisher
	release                                                                          map[string]any
	assets                                                                           []map[string]any
	data                                                                             map[string][]byte
	requests                                                                         []string
	writes                                                                           int
	policy                                                                           string
	tag                                                                              string
	failCreate, failUpload, failPublish, corruptDownload, redirect, publishedMutable bool
	before                                                                           func(*http.Request)
	after                                                                            func(*http.Request, *http.Response) (*http.Response, error)
}

func newPublisherFixture(t *testing.T) *publisherFixture {
	t.Helper()
	dir, m, raw := candidateFixture(t)
	f := &publisherFixture{dir: dir, m: m, identity: (ParsedManifest{m, Digest(raw)}).Identity(), data: map[string][]byte{}, policy: `{"enabled":true}`}
	f.p = newPublisher("release-token", "policy-token", roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if f.before != nil {
			f.before(r)
		}
		f.requests = append(f.requests, r.Method+" "+r.URL.String())
		if r.Header.Get("Cookie") != "" {
			t.Fatal("cookie leaked")
		}
		path := strings.TrimPrefix(r.URL.Path, "/repos/acoz-labs/mandalore")
		auth := r.Header.Get("Authorization")
		if path == "/immutable-releases" {
			if auth != "Bearer policy-token" {
				t.Fatal("policy credential scope")
			}
		} else if r.URL.Host == "release-assets.githubusercontent.com" {
			if auth != "" {
				t.Fatal("credential forwarded to CDN")
			}
		} else if auth != "Bearer release-token" && auth != "" {
			t.Fatal("wrong release credential")
		}
		status, b, header := 200, []byte(nil), http.Header{}
		encode := func(v any) { b, _ = json.Marshal(v) }
		switch {
		case path == "/immutable-releases" && r.Method == "GET":
			b = []byte(f.policy)
		case path == "/git/ref/tags/"+m.Tag && r.Method == "GET":
			if f.tag == "" {
				status = 404
			} else {
				encode(map[string]any{"ref": "refs/tags/" + m.Tag, "object": map[string]any{"type": "commit", "sha": f.tag}})
			}
		case strings.HasPrefix(path, "/git/tags/") && r.Method == "GET":
			status = 404
		case path == "/releases" && r.Method == "GET":
			if auth == "" {
				t.Fatal("draft discovery must be authenticated")
			}
			if f.release == nil {
				b = []byte(`[]`)
			} else {
				f.release["assets"] = f.assets
				encode([]any{f.release})
			}
		case path == "/releases" && r.Method == "POST":
			if f.release != nil {
				t.Fatal("duplicate release creation")
			}
			f.writes++
			if err := json.NewDecoder(r.Body).Decode(&f.release); err != nil {
				t.Fatal(err)
			}
			if f.release["draft"] != true || len(f.release) != 6 {
				t.Fatal("creation must only stage a draft with the explicit candidate metadata")
			}
			f.release["id"] = int64(42)
			f.release["immutable"] = false
			f.release["assets"] = f.assets
			status = 201
			encode(f.release)
			if f.failCreate {
				f.failCreate = false
				return nil, errors.New("provider-secret-error")
			}
		case path == "/releases/42/assets" && r.Method == "POST" && r.URL.Host == "uploads.github.com":
			f.writes++
			name := r.URL.Query().Get("name")
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			if r.ContentLength != int64(len(data)) {
				t.Fatal("upload content length does not bind the owned bytes")
			}
			id := int64(100 + len(f.assets))
			a := map[string]any{"id": id, "name": name, "size": len(data), "digest": "sha256:" + Digest(data), "state": "uploaded"}
			f.assets = append(f.assets, a)
			f.data[fmt.Sprint(id)] = data
			status = 201
			encode(a)
			if f.failUpload {
				f.failUpload = false
				status = 502
				b = []byte("provider-secret-error")
			}
		case path == "/releases/42" && r.Method == "PATCH":
			var patch map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil || len(patch) != 2 || patch["draft"] != false || patch["make_latest"] != "legacy" {
				t.Fatal("publication must not rewrite candidate identity or assets", patch, err)
			}
			f.writes++
			f.release["draft"] = false
			f.release["immutable"] = !f.publishedMutable
			f.release["published_at"] = "2026-09-14T00:00:00Z"
			f.tag = m.SourceCommit
			encode(f.release)
			if f.failPublish {
				f.failPublish = false
				return nil, errors.New("provider-secret-error")
			}
		case (path == "/releases/42" || path == "/releases/tags/"+m.Tag) && r.Method == "GET":
			if f.release == nil {
				status = 404
			} else {
				f.release["assets"] = f.assets
				encode(f.release)
			}
		case strings.HasPrefix(path, "/releases/assets/") && r.Method == "GET":
			id := strings.TrimPrefix(path, "/releases/assets/")
			b = f.data[id]
			if b == nil {
				status = 404
			}
			if f.corruptDownload {
				b = []byte("changed")
			}
			if f.redirect {
				status = 302
				header.Set("Location", "https://release-assets.githubusercontent.com/"+id+"?signature=opaque")
			}
		case r.URL.Host == "release-assets.githubusercontent.com" && r.Method == "GET":
			b = f.data[strings.TrimPrefix(r.URL.Path, "/")]
		default:
			t.Fatalf("unexpected publisher request %s %s", r.Method, r.URL)
		}
		if r.Method != "GET" && auth != "Bearer release-token" {
			t.Fatal("unauthenticated write")
		}
		response := &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(bytes.NewReader(b)), ContentLength: int64(len(b)), Request: r}
		if f.after != nil {
			return f.after(r, response)
		}
		return response, nil
	}))
	return f
}

func TestPublisherStagesVerifiesPublishesAndReusesSameBytes(t *testing.T) {
	f := newPublisherFixture(t)
	f.redirect = true
	privateReads, publicReads := 0, 0
	f.before = func(r *http.Request) {
		if publicationAssetPath.MatchString(r.URL.Path) {
			if r.Header.Get("Authorization") == "" {
				publicReads++
			} else {
				privateReads++
			}
		}
		if r.Method == "PATCH" && privateReads != 8 {
			t.Fatalf("published before all remote bytes were checked: %d", privateReads)
		}
	}
	r, err := f.p.Publish(context.Background(), f.dir, f.identity)
	if err != nil {
		t.Fatal(err, r)
	}
	if r.Phase != "publication-verified" || r.ReleaseID != 42 || len(r.Assets) != 8 || r.Identity != f.identity || r.PendingOperation != "" || f.writes != 10 {
		t.Fatal("incomplete publication", r, f.writes)
	}
	if publicReads != 10 {
		t.Fatalf("public inspection must check manifest/checksums and download all eight assets: %d", publicReads)
	}
	for _, a := range r.Assets {
		b, _ := os.ReadFile(filepath.Join(f.dir, a.Name))
		if !bytes.Equal(b, f.data[fmt.Sprint(a.ID)]) {
			t.Fatal("published bytes changed")
		}
	}
	writes := f.writes
	r, err = f.p.Publish(context.Background(), f.dir, f.identity)
	if err != nil || r.Phase != "publication-verified" || f.writes != writes {
		t.Fatal("published retry mutated or failed", r, err)
	}
}

func TestPublisherRefusesBeforeWriting(t *testing.T) {
	for _, kind := range []string{"disabled", "missing", "null", "duplicate", "malformed", "wrong identity", "wrong tag", "corrupt local", "missing credential", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublisherFixture(t)
			ctx := context.Background()
			switch kind {
			case "disabled":
				f.policy = `{"enabled":false}`
			case "missing":
				f.policy = `{}`
			case "null":
				f.policy = `{"enabled":null}`
			case "duplicate":
				f.policy = `{"enabled":false,"enabled":true}`
			case "malformed":
				f.policy = `{"enabled":"true"}`
			case "wrong identity":
				f.identity += "x"
			case "wrong tag":
				f.tag = strings.Repeat("f", 40)
			case "corrupt local":
				if err := os.WriteFile(filepath.Join(f.dir, "install.sh"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing credential":
				f.p.token = ""
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			r, err := f.p.Publish(ctx, f.dir, f.identity)
			if err == nil || f.writes != 0 || r.PendingOperation != "" {
				t.Fatal("unsafe publication", r, err, f.writes)
			}
		})
	}
}

func TestPublisherRetriesUncertainWritesWithoutReplacement(t *testing.T) {
	for _, kind := range []string{"create", "upload", "publish"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublisherFixture(t)
			f.failCreate = kind == "create"
			f.failUpload = kind == "upload"
			f.failPublish = kind == "publish"
			r, err := f.p.Publish(context.Background(), f.dir, f.identity)
			if err == nil || r.PendingOperation == "" || kind != "create" && r.ReleaseID != 42 || strings.Contains(err.Error(), "secret") {
				t.Fatal("uncertain effects lost or leaked", r, err)
			}
			if kind == "create" && (r.ReleaseID != 0 || r.PendingOperation != "create-draft") {
				t.Fatal("unconfirmed draft identity claimed", r)
			}
			r, err = f.p.Publish(context.Background(), f.dir, f.identity)
			if err != nil || r.Phase != "publication-verified" || f.writes != 10 {
				t.Fatal("retry did not reuse writes", r, err, f.writes)
			}
		})
	}
}

func TestPublisherPreservesConflictingDrafts(t *testing.T) {
	for _, kind := range []string{"body", "target", "asset digest", "asset starter", "extra asset", "wrong release id", "duplicate asset id"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublisherFixture(t)
			f.failUpload = true
			if _, err := f.p.Publish(context.Background(), f.dir, f.identity); err == nil {
				t.Fatal("expected interrupted setup")
			}
			if f.release == nil || len(f.assets) != 1 {
				t.Fatal("fixture did not reach interrupted upload")
			}
			switch kind {
			case "body":
				f.release["body"] = "foreign"
			case "target":
				f.release["target_commitish"] = "main"
			case "asset digest":
				f.assets[0]["digest"] = "sha256:" + strings.Repeat("f", 64)
			case "asset starter":
				f.assets[0]["state"] = "starter"
			case "extra asset":
				f.assets[0]["name"] = "foreign.txt"
			case "wrong release id":
				f.release["id"] = int64(0)
			case "duplicate asset id":
				f.assets = append(f.assets, f.assets[0])
			}
			writes := f.writes
			if _, err := f.p.Publish(context.Background(), f.dir, f.identity); err == nil || f.writes != writes {
				t.Fatal("conflict was mutated or accepted", err)
			}
		})
	}
}

func TestPublisherNeverClaimsUnverifiedPublication(t *testing.T) {
	for _, kind := range []string{"remote bytes", "published mutable", "policy changes"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublisherFixture(t)
			if kind == "remote bytes" {
				f.corruptDownload = true
			}
			if kind == "published mutable" {
				f.publishedMutable = true
			}
			if kind == "policy changes" {
				f.before = func(r *http.Request) {
					if len(f.assets) == 8 {
						f.policy = `{"enabled":false}`
					}
				}
			}
			r, err := f.p.Publish(context.Background(), f.dir, f.identity)
			if err == nil || r.Phase == "publication-verified" {
				t.Fatal("unverified success", r, err)
			}
			if kind != "published mutable" && f.release["draft"] != true {
				t.Fatal("published before preconditions")
			}
		})
	}
}

func TestPublisherRejectsChangedLocalOrRemoteState(t *testing.T) {
	for _, kind := range []string{"local bytes", "local link", "tag", "existing remote bytes", "cancelled", "asset disappears"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublisherFixture(t)
			f.failUpload = true
			if r, err := f.p.Publish(context.Background(), f.dir, f.identity); err == nil || r.PendingOperation != "upload:SHA256SUMS" {
				t.Fatal("fixture did not stage first asset", r, err)
			}
			writes := f.writes
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.before = func(r *http.Request) {
				if r.Method == "GET" && publicationAssetPath.MatchString(r.URL.Path) {
					switch kind {
					case "local bytes":
						if err := os.WriteFile(filepath.Join(f.dir, "install.sh"), []byte("changed"), 0600); err != nil {
							t.Fatal(err)
						}
					case "local link":
						path := filepath.Join(f.dir, "install.sh")
						if err := os.Remove(path); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink("manifest.json", path); err != nil {
							t.Fatal(err)
						}
					case "tag":
						f.tag = strings.Repeat("f", 40)
					case "existing remote bytes":
						f.corruptDownload = true
					case "cancelled":
						cancel()
					}
				}
				if kind == "asset disappears" && len(f.assets) == 2 && r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/releases/42") {
					f.assets = f.assets[1:]
				}
			}
			r, err := f.p.Publish(ctx, f.dir, f.identity)
			if err == nil || f.release["draft"] != true || r.Phase == "publication-verified" {
				t.Fatal("changed state published", r, err)
			}
			if (kind == "existing remote bytes" || kind == "cancelled") && f.writes != writes {
				t.Fatal("mutated after first existing-byte read failed")
			}
		})
	}
}

func TestPublisherRefusesRedirectsInvalidMetadataAndFailedPolicyRead(t *testing.T) {
	for _, kind := range []string{"policy 403", "policy 404", "policy redirect", "list null", "list duplicate keys", "list duplicate tag", "list pagination bound", "list oversized", "metadata redirect", "upload redirect", "asset redirect"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublisherFixture(t)
			f.after = func(r *http.Request, response *http.Response) (*http.Response, error) {
				setBody := func(b string) {
					response.Body.Close()
					response.Body = io.NopCloser(strings.NewReader(b))
					response.ContentLength = int64(len(b))
				}
				redirect := func() {
					response.StatusCode = 302
					response.Header.Set("Location", "https://example.invalid/private?secret=do-not-log")
				}
				path := r.URL.Path
				if strings.HasSuffix(path, "/immutable-releases") {
					switch kind {
					case "policy 403":
						response.StatusCode = 403
					case "policy 404":
						response.StatusCode = 404
					case "policy redirect":
						redirect()
					}
				}
				if path == "/repos/acoz-labs/mandalore/releases" && r.Method == "GET" {
					switch kind {
					case "list null":
						setBody(`null`)
					case "list duplicate keys":
						setBody(`[{"tag_name":"x","tag_name":"y"}]`)
					case "list duplicate tag":
						setBody(`[{"tag_name":"` + f.m.Tag + `"},{"tag_name":"` + f.m.Tag + `"}]`)
					case "list pagination bound":
						setBody(`[` + strings.Repeat(`{},`, 99) + `{}]`)
					case "list oversized":
						setBody(strings.Repeat(" ", maxReleaseMetadata+1))
					case "metadata redirect":
						redirect()
					}
				}
				if kind == "upload redirect" && r.URL.Host == "uploads.github.com" {
					redirect()
				}
				if kind == "asset redirect" && publicationAssetPath.MatchString(path) {
					redirect()
				}
				return response, nil
			}
			r, err := f.p.Publish(context.Background(), f.dir, f.identity)
			if err == nil || strings.Contains(err.Error(), "do-not-log") || r.Phase == "publication-verified" {
				t.Fatal("unsafe response accepted or leaked", r, err)
			}
			if !strings.HasPrefix(kind, "upload") && !strings.HasPrefix(kind, "asset") && f.writes != 0 {
				t.Fatal("wrote despite failed read-only preflight", f.writes)
			}
			if kind == "upload redirect" && r.PendingOperation == "" {
				t.Fatal("uncertain upload lost")
			}
		})
	}
}

func TestPublicationTransportRestrictsCredentialAndMutationScope(t *testing.T) {
	for _, tc := range []struct {
		method, address string
		policy          bool
	}{
		{"DELETE", releaseAPI + "/releases/42", false},
		{"PATCH", releaseAPI + "/releases/assets/100", false},
		{"PUT", releaseAPI + "/immutable-releases", true},
		{"GET", releaseAPI + "/releases/42", true},
		{"GET", releaseAPI + "/immutable-releases", false},
		{"GET", "https://api.github.com/repos/other/repository/releases/42", false},
		{"GET", "https://api.github.com:443/repos/acoz-labs/mandalore/releases/42", false},
		{"GET", releaseAPI + "/releases?per_page=100&page=1&page=2", false},
		{"POST", "https://uploads.github.com/repos/acoz-labs/mandalore/releases/42/assets?name=../secret", false},
		{"POST", "https://release-assets.githubusercontent.com/100", false},
	} {
		t.Run(tc.method+" "+tc.address, func(t *testing.T) {
			transport := publicationTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("untrusted request reached network")
				return nil, nil
			}), token: "secret", policy: tc.policy}
			r, err := http.NewRequest(tc.method, tc.address, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := transport.RoundTrip(r); err == nil {
				t.Fatal("untrusted endpoint accepted")
			}
		})
	}
}

func TestPublisherDoesNotTreatBrokenAnnotatedTagAsAbsent(t *testing.T) {
	f := newPublisherFixture(t)
	f.after = func(r *http.Request, response *http.Response) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/git/ref/tags/") {
			b := `{"ref":"refs/tags/` + f.m.Tag + `","object":{"type":"tag","sha":"` + strings.Repeat("b", 40) + `"}}`
			response.Body.Close()
			response.Body = io.NopCloser(strings.NewReader(b))
			response.ContentLength = int64(len(b))
			response.StatusCode = 200
		}
		return response, nil
	}
	if _, err := f.p.Publish(context.Background(), f.dir, f.identity); err == nil || f.writes != 0 {
		t.Fatal("broken existing tag was treated as missing", err)
	}
}
