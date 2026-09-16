# Request-count probe

Repository basis: `29c4d48a697f5ccdbddcd87b3f145ba2048df240`.
Temporarily place the following test in `internal/distribution/quota_probe_test.go`
and run `mise exec -- go test ./internal/distribution -run TestDiscoveryQuotaCounts -v -count=1`.
It uses existing in-memory release/publisher fixtures and disposable directories;
the production client code is unchanged. The inert runtime verifier is a test
seam, not proof of native execution. Update means a different retained digest,
not a real new public version. Counts exclude fixture preparation and include no
redirects/annotated tags. No live GitHub requests or publication occurred.

```go
package distribution

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
)

func TestDiscoveryQuotaCounts(t *testing.T) {
	for _, update := range []bool{false, true} {
		f := newReleaseFixture(t)
		o := InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix"), Version: "1.0.0"}
		if update {
			old := o
			old.Version = ""
			old.Candidate = nextInstallCandidate(t)
			p, err := planInstall(context.Background(), old, f.client)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil); err != nil {
				t.Fatal(err)
			}
		}
		start := len(f.requests)
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
		t.Logf("update=%t plan=%d apply=%d combined=%d exact_replay=%d", update, planned-start, applied-planned, applied-start, len(f.requests)-applied)
	}
	f := newPublisherFixture(t)
	if _, err := f.p.Publish(context.Background(), f.dir, f.identity); err != nil {
		t.Fatal(err)
	}
	reads, assets := 0, 0
	f.before = func(r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Authorization") != "" {
			t.Fatal("unexpected request")
		}
		reads++
		if publicationAssetPath.MatchString(r.URL.Path) {
			assets++
		}
	}
	if _, err := f.p.public.VerifyPublication(context.Background(), "1.0.0", f.identity); err != nil {
		t.Fatal(err)
	}
	t.Logf("full_verifier=%d asset_requests=%d", reads, assets)
}
```

Observed output on the pinned host toolchain:

```text
update=false plan=4 apply=18 combined=22 exact_replay=0
update=true plan=4 apply=18 combined=22 exact_replay=0
full_verifier=15 asset_requests=10
PASS
```

These are measured HTTP fixture requests, not a claim about GitHub's exact quota
accounting. The temporary executable test was removed after measurement; this
source is retained for review and reproduction. Product regression tests belong
to the subsequent implementation.

