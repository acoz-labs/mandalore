package distribution

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCandidateCredentialOnlyReachesExactMetadataGET(t *testing.T) {
	calls := 0
	tr := candidateMetadataTransport{token: "synthetic-workflow-token", base: transportRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer synthetic-workflow-token" || r.Header.Get("Cookie") != "" {
			t.Fatal("wrong metadata authentication")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	})}
	for _, path := range []string{"https://api.github.com/repos/acoz-labs/mandalore", "https://api.github.com/repos/acoz-labs/mandalore/actions/runs/1"} {
		r, _ := http.NewRequest("GET", path, nil)
		r.Header.Set("Cookie", "synthetic-cookie")
		response, err := tr.RoundTrip(r)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
	}
	for _, path := range []string{"http://api.github.com/repos/acoz-labs/mandalore", "https://untrusted.example/repos/acoz-labs/mandalore", "https://api.github.com/repos/other/project", "https://api.github.com/repos/acoz-labs/mandalore/releases", "https://api.github.com/repos/acoz-labs/mandalore/actions/artifacts/1/zip", "https://api.github.com/repos/acoz-labs/mandalore?token=bad"} {
		r, _ := http.NewRequest("GET", path, nil)
		if _, err := tr.RoundTrip(r); err == nil {
			t.Fatal("credential scope expanded", path)
		}
	}
	r, _ := http.NewRequest("DELETE", "https://api.github.com/repos/acoz-labs/mandalore/actions/artifacts/1", nil)
	if _, err := tr.RoundTrip(r); err == nil {
		t.Fatal("mutating credential use allowed")
	}
	if calls != 2 {
		t.Fatal("untrusted request reached transport", calls)
	}
}

func TestCandidateClientRefusesRedirectsWithoutForwardingCredential(t *testing.T) {
	c := NewCandidateClient("synthetic-workflow-token")
	calls := 0
	c.api.http.Transport = candidateMetadataTransport{token: "synthetic-workflow-token", base: transportRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://untrusted.example/redirect"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	_, err := c.InspectCandidate(context.Background(), CandidateSelection{SourceCommit: strings.Repeat("a", 40), RunID: 1, ArtifactID: 2})
	if err == nil || calls != 1 || strings.Contains(err.Error(), "synthetic-workflow-token") {
		t.Fatal("redirect/credential disclosure", calls, err)
	}
}
