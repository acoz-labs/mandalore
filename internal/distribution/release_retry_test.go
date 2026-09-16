package distribution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type refusedBody struct{ t *testing.T }

func (b refusedBody) Read([]byte) (int, error) {
	b.t.Fatal("refusal body must not be read")
	return 0, io.EOF
}

func TestReleaseRateLimitHeaderBoundaries(t *testing.T) {
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		status  int
		h       http.Header
		limited bool
		kind    string
		delay   int64
		reset   string
	}{
		{"429 unknown", 429, nil, true, "unspecified", 0, ""},
		{"403 unknown", 403, nil, false, "", 0, ""},
		{"403 reset alone", 403, http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Add(time.Hour).Unix(), 10)}}, false, "", 0, ""},
		{"403 primary", 403, http.Header{"X-Ratelimit-Remaining": {"0"}}, true, "primary", 0, ""},
		{"403 retry", 403, http.Header{"Retry-After": {"1"}}, true, "unspecified", 1, ""},
		{"maximum delay", 429, http.Header{"Retry-After": {"86400"}}, true, "unspecified", 86400, ""},
		{"zero delay", 429, http.Header{"Retry-After": {"0"}}, true, "unspecified", 0, ""},
		{"excessive delay", 429, http.Header{"Retry-After": {"86401"}}, true, "unspecified", 0, ""},
		{"primary with future reset", 403, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {strconv.FormatInt(now.Add(time.Hour).Unix(), 10)}}, true, "primary", 0, "2026-09-16T01:00:00Z"},
		{"both lower bounds", 429, http.Header{"Retry-After": {"60"}, "X-Ratelimit-Reset": {strconv.FormatInt(now.Add(time.Hour).Unix(), 10)}}, true, "unspecified", 60, "2026-09-16T01:00:00Z"},
		{"maximum reset", 429, http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Add(24*time.Hour).Unix(), 10)}}, true, "unspecified", 0, "2026-09-17T00:00:00Z"},
		{"excessive reset", 429, http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Add(24*time.Hour+time.Second).Unix(), 10)}}, true, "unspecified", 0, ""},
		{"stale reset", 429, http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(now.Unix(), 10)}}, true, "unspecified", 0, ""},
		{"other status", 500, http.Header{"Retry-After": {"60"}, "X-Ratelimit-Remaining": {"0"}}, false, "", 0, ""},
		{"duplicate remaining", 403, http.Header{"X-Ratelimit-Remaining": {"0", "0"}}, false, "", 0, ""},
		{"case duplicated", 403, http.Header{"Retry-After": {"60"}, "retry-after": {"60"}}, false, "", 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := releaseRateLimit(tc.status, tc.h, now)
			if (e != nil) != tc.limited {
				t.Fatalf("classification: %+v", e)
			}
			if e == nil {
				return
			}
			if e.Retry.HTTPStatus != tc.status || e.Retry.Kind != tc.kind || e.Retry.RetryAfterSeconds != tc.delay || e.Retry.ResetAt != tc.reset {
				t.Fatalf("advice: %+v", e.Retry)
			}
		})
	}
	for _, field := range []string{"Retry-After", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		for _, value := range []string{"", "-1", "+1", "1.5", " 0", "0 ", "0,0", "0\r\nprivate", "0\x00", "１２", "18446744073709551616", strings.Repeat("0", 65), "Wed, 16 Sep 2026 01:00:00 GMT"} {
			h := http.Header{}
			h.Set(field, value)
			if _, ok := releaseHeaderNumber(h, field); ok {
				t.Fatalf("accepted invalid %s=%q", field, value)
			}
			e := releaseRateLimit(429, h, now)
			if e.Retry.RetryAfterSeconds != 0 || e.Retry.ResetAt != "" || e.Retry.Kind != "unspecified" || strings.Contains(e.Error(), "private") {
				t.Fatal("invalid header escaped into advice", e)
			}
		}
	}
}

func TestReleaseRefusalCancellationTakesPrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newReleaseClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		cancel()
		return &http.Response{StatusCode: 429, Header: http.Header{}, Body: refusedBody{t}, Request: r}, nil
	}))
	_, err := c.get(ctx, releaseAPI+"/synthetic", "application/json", 100, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation misreported as quota", err)
	}
}

func TestReleaseRateLimitExplicitRetryPreservesPriorRuntime(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprint("update=", update), func(t *testing.T) {
			f := newReleaseFixture(t)
			o := InstallOptions{Prefix: filepath.Join(t.TempDir(), "prefix"), Version: "1.0.0"}
			var previous string
			if update {
				old, err := planInstall(context.Background(), InstallOptions{Prefix: o.Prefix, Candidate: nextInstallCandidate(t)}, f.client)
				if err != nil {
					t.Fatal(err)
				}
				r, err := applyInstall(context.Background(), old, f.client, inertInstallVerifier, nil)
				if err != nil {
					t.Fatal(err)
				}
				previous = r.Runtime
			}
			p, err := planInstall(context.Background(), o, f.client)
			if err != nil {
				t.Fatal(err)
			}
			var refused string
			for _, a := range p.Source.Published.Assets {
				if a.Name == p.Binary.Name {
					refused = fmt.Sprintf("/repos/acoz-labs/mandalore/releases/assets/%d", a.ID)
				}
			}
			f.status[refused] = 429
			before := len(f.requests)
			r, err := applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil)
			var quota *ReleaseRateLimitError
			if !errors.As(err, &quota) || r.Phase != "staging" || r.DestinationChanged || r.Installed || len(f.requests)-before != 5 {
				t.Fatal("refusal effects/retry count", r, err, len(f.requests)-before)
			}
			if update {
				target, err := os.Readlink(p.Launcher)
				if err != nil || target != previous {
					t.Fatal("old launcher changed", target, err)
				}
			} else if _, err := os.Lstat(p.Prefix); !os.IsNotExist(err) {
				t.Fatal("fresh prefix changed", err)
			}
			delete(f.status, refused)
			r, err = applyInstall(context.Background(), p, f.client, inertInstallVerifier, nil)
			if err != nil || !r.Installed {
				t.Fatal("explicit retry failed", r, err)
			}
		})
	}
}
func (b refusedBody) Close() error { return nil }

func TestReleaseRefusalGuidanceAndNoRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		header http.Header
		want   string
	}{
		{"ordinary forbidden", 403, http.Header{}, "release request was refused"},
		{"unknown quota", 429, http.Header{}, "Retry timing is unavailable"},
		{"retry delay", 403, http.Header{"Retry-After": {"60"}}, "Wait at least 60 seconds"},
		{"primary", 403, http.Header{"X-Ratelimit-Remaining": {"0"}}, "rate-limited"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			c := newReleaseClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: tc.status, Header: tc.header, Body: refusedBody{t}, Request: r}, nil
			}))
			_, err := c.get(context.Background(), releaseAPI+"/synthetic", "application/json", 100, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("missing guidance %q: %v", tc.want, err)
			}
			if strings.Contains(err.Error(), "no installation occurred") || strings.Contains(err.Error(), "Nothing was installed") {
				t.Fatal("request error invented operation effects", err)
			}
			if tc.status == 403 && len(tc.header) == 0 && strings.Contains(err.Error(), "rate-limited") {
				t.Fatal("ordinary refusal misclassified", err)
			}
			if requests != 1 {
				t.Fatal("refusal retried without caller action", requests)
			}
		})
	}
}
