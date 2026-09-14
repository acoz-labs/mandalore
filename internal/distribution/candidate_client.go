package distribution

import (
	"context"
	"errors"
	"net/http"
	"regexp"
)

// CandidateClient accepts an explicitly supplied workflow credential only for
// exact metadata GETs. It has no upload, deletion, arbitrary URL or asset API.
// Public-user release clients remain anonymous and never read this credential.
type CandidateClient struct{ api *ReleaseClient }

func NewCandidateClient(token string) *CandidateClient {
	c := NewReleaseClient()
	c.http.Transport = candidateMetadataTransport{c.http.Transport, token}
	c.http.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("candidate metadata redirects are not permitted")
	}
	return &CandidateClient{c}
}

func (c *CandidateClient) InspectCandidate(ctx context.Context, s CandidateSelection) (CandidateTransport, error) {
	return c.api.InspectCandidate(ctx, s)
}

var candidateMetadataPath = regexp.MustCompile(`^/repos/acoz-labs/mandalore(/actions/(runs|workflows|artifacts)/[1-9][0-9]*)?$`)

type candidateMetadataTransport struct {
	base  http.RoundTripper
	token string
}

func (t candidateMetadataTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodGet || r.URL.Scheme != "https" || r.URL.Host != "api.github.com" || r.URL.User != nil || r.URL.RawQuery != "" || !candidateMetadataPath.MatchString(r.URL.Path) {
		return nil, errors.New("candidate metadata endpoint is not permitted")
	}
	r = r.Clone(r.Context())
	r.Header.Del("Authorization")
	r.Header.Del("Cookie")
	if t.token != "" {
		r.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(r)
}
