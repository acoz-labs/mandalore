package distribution

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type refusedBody struct{ t *testing.T }

func (b refusedBody) Read([]byte) (int, error) {
	b.t.Fatal("refusal body must not be read")
	return 0, io.EOF
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
