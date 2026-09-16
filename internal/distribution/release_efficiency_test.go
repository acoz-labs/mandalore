package distribution

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

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
