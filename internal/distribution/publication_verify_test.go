package distribution

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestPublishedVerificationIsReadOnlyAndChecksEveryAsset(t *testing.T) {
	f := newPublisherFixture(t)
	if _, err := f.p.Publish(context.Background(), f.dir, f.identity); err != nil {
		t.Fatal(err)
	}
	writes := f.writes
	reads := 0
	f.before = func(r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Authorization") != "" {
			t.Fatal("verification must be anonymous and read-only")
		}
		if publicationAssetPath.MatchString(r.URL.Path) {
			reads++
		}
	}
	r, err := f.p.public.VerifyPublication(context.Background(), "1.0.0", f.identity)
	if err != nil || r.Phase != "publication-verified" || r.Identity != f.identity || r.ReleaseID != 42 || len(r.Assets) != 8 || f.writes != writes || reads != 10 {
		t.Fatal("incomplete published verification", r, err, reads)
	}
}

func TestPublishedVerificationRefusesIncompleteOrWrongRelease(t *testing.T) {
	for _, kind := range []string{"wrong identity", "body", "mutable", "draft", "missing asset", "bytes", "changed after download", "tag changed after download", "invalid version", "invalid identity"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublisherFixture(t)
			if _, err := f.p.Publish(context.Background(), f.dir, f.identity); err != nil {
				t.Fatal(err)
			}
			writes := f.writes
			version, identity := "1.0.0", f.identity
			switch kind {
			case "wrong identity":
				identity = "mandalore:" + strings.Repeat("f", 40) + ":sha256:" + strings.Repeat("f", 64)
			case "body":
				f.release["body"] = "foreign ledger"
			case "mutable":
				f.release["immutable"] = false
			case "draft":
				f.release["draft"] = true
			case "missing asset":
				f.assets = f.assets[1:]
			case "bytes":
				f.corruptDownload = true
			case "changed after download", "tag changed after download":
				reads := 0
				f.before = func(r *http.Request) {
					if publicationAssetPath.MatchString(r.URL.Path) {
						reads++
						if reads == 10 {
							if kind == "tag changed after download" {
								f.tag = strings.Repeat("f", 40)
							} else {
								f.release["body"] = "changed"
							}
						}
					}
				}
			case "invalid version":
				version = "latest"
			case "invalid identity":
				identity = "unverified"
			}
			r, err := f.p.public.VerifyPublication(context.Background(), version, identity)
			if err == nil || r.Phase == "publication-verified" || f.writes != writes {
				t.Fatal("invalid release verified", r, err)
			}
		})
	}
}
