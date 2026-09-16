package distribution

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestReleaseConcurrentOperationsKeepVerifiedStatePrivate(t *testing.T) {
	f := newReleaseFixture(t)
	transport := f.client.http.Transport
	var mu sync.Mutex
	// Serialize only the fixture's request-log writes, not either operation.
	f.client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		return transport.RoundTrip(r)
	})
	t.Run("operations", func(t *testing.T) {
		for _, name := range []string{"one", "two"} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				p, err := planInstall(context.Background(), InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix")}, f.client)
				if err != nil {
					t.Fatal(err)
				}
				r, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil)
				if err != nil || !r.Installed {
					t.Fatal(r, err)
				}
			})
		}
	})
	if len(f.requests) != 26 {
		t.Fatal("independent operations shared/skipped live verification", len(f.requests))
	}
}

func TestReleaseInstallRequestBudget(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprint("update=", update), func(t *testing.T) {
			f := newReleaseFixture(t)
			o := InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix"), Version: "1.0.0"}
			if update {
				old := InstallOptions{Prefix: o.Prefix, Candidate: nextInstallCandidate(t)}
				p, err := planInstall(context.Background(), old, f.client)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil); err != nil {
					t.Fatal(err)
				}
			}
			p, err := planInstall(context.Background(), o, f.client)
			if err != nil {
				t.Fatal(err)
			}
			planned := len(f.requests)
			r, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil)
			if err != nil || !r.Installed {
				t.Fatal(r, err)
			}
			applied := len(f.requests)
			if _, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil); err != nil {
				t.Fatal(err)
			}
			t.Logf("plan=%d apply=%d total=%d replay=%d", planned, applied-planned, applied, len(f.requests)-applied)
			if planned != 4 || applied != 13 || len(f.requests) != applied {
				t.Fatal("request budget or zero-request replay changed")
			}
		})
	}
}

func TestReleaseStagingRequiresFreshPrivateSource(t *testing.T) {
	for _, kind := range []string{"missing", "wrong view", "corrupt bytes", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			f := newReleaseFixture(t)
			var source verifiedRelease
			p, err := planInstallVerified(context.Background(), InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix")}, f.client, &source)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			v := &source
			switch kind {
			case "missing":
				v = nil
			case "wrong view":
				source.view.ID++
			case "corrupt bytes":
				source.manifest = []byte("corrupt")
			case "cancelled":
				cancel()
			}
			before := len(f.requests)
			if err := stageInstallSource(ctx, p, f.client, root, v); err == nil {
				t.Fatal("invalid source staged")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 || len(f.requests) != before {
				t.Fatal("invalid source caused writes or network", err)
			}
		})
	}
}

func TestEfficientReleaseApplyRejectsChangesAndPreservesDestination(t *testing.T) {
	for _, kind := range []string{"stale preview", "release during staging", "tag during staging", "asset during staging", "probe changes bytes", "corrupt binary"} {
		t.Run(kind, func(t *testing.T) {
			f := newReleaseFixture(t)
			p, err := planInstall(context.Background(), InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix")}, f.client)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "stale preview" {
				f.release["id"] = int64(43)
			}
			if kind == "corrupt binary" {
				for _, a := range f.assets {
					if a["name"] == p.Binary.Name {
						f.body[fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", a["id"])] = []byte("corrupt")
					}
				}
			}
			probe := func(ctx context.Context, path string, p InstallPlan) error {
				if kind == "probe changes bytes" {
					return os.WriteFile(path, []byte("changed by probe"), 0600)
				}
				return inertInstallVerifier(ctx, path, p)
			}
			after := func(phase string) error {
				if phase == "verified" {
					switch kind {
					case "release during staging":
						f.release["id"] = int64(43)
					case "tag during staging":
						f.body["/repos/acoz-labs/mandalore/git/ref/tags/v1.0.0"] = []byte(`{"ref":"refs/tags/v1.0.0","object":{"type":"commit","sha":"` + strings.Repeat("f", 40) + `"}}`)
					case "asset during staging":
						f.assets[0]["digest"] = "sha256:" + strings.Repeat("f", 64)
					}
				}
				return nil
			}
			r, err := applyInstall(context.Background(), p, f.client, probe, after)
			if err == nil || r.Installed || r.DestinationChanged {
				t.Fatal("changed source installed", r, err)
			}
			if _, err := os.Lstat(p.Prefix); !os.IsNotExist(err) {
				t.Fatal("refusal changed destination", err)
			}
		})
	}
}

func TestReleaseInstallRetainsOriginalManifestBytes(t *testing.T) {
	f := newReleaseFixture(t)
	var original []byte
	for _, a := range f.assets {
		if a["name"] == "manifest.json" {
			path := fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", a["id"])
			original = append([]byte(" \n"), f.body[path]...)
			f.body[path] = original
			a["size"], a["digest"] = len(original), "sha256:"+Digest(original)
		}
	}
	sums, err := Checksums(original)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range f.assets {
		if a["name"] == "SHA256SUMS" {
			f.body[fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", a["id"])] = sums
			a["size"], a["digest"] = len(sums), "sha256:"+Digest(sums)
		}
	}
	p, err := planInstall(context.Background(), InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix")}, f.client)
	if err != nil {
		t.Fatal(err)
	}
	r, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(r.Runtime), "manifest.json"))
	if err != nil || !bytes.Equal(b, original) {
		t.Fatal("manifest was reconstructed rather than retained", err)
	}
}

func TestReleaseVerificationRequestBudget(t *testing.T) {
	f := newPublisherFixture(t)
	if _, err := f.p.Publish(context.Background(), f.dir, f.identity); err != nil {
		t.Fatal(err)
	}
	requests := 0
	assets := map[string]int{}
	f.before = func(r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("unexpected authenticated or mutating request")
		}
		requests++
		if publicationAssetPath.MatchString(r.URL.Path) {
			assets[r.URL.Path]++
		}
	}
	if _, err := f.p.public.VerifyPublication(context.Background(), "1.0.0", f.identity); err != nil {
		t.Fatal(err)
	}
	t.Logf("requests=%d distinct_assets=%d", requests, len(assets))
	if requests != 13 || len(assets) != 8 {
		t.Fatal("full verification budget or asset coverage changed")
	}
	for path, count := range assets {
		if count != 1 {
			t.Fatalf("asset %s fetched %d times", path, count)
		}
	}
}
