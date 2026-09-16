package distribution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const releaseAPI = "https://api.github.com/repos/acoz-labs/mandalore"
const maxReleaseMetadata = 128 << 10

var ErrNoRelease = errors.New("the selected published Mandalore release is not available")

// ReleaseAsset records immutable API identity, not a mutable or signed download URL.
type ReleaseAsset struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// ReleaseView has verified manifest/checksum bytes and cross-checked metadata for
// all assets. It does NOT claim all executable payloads have been downloaded.
type ReleaseView struct {
	ID         int64          `json:"release_id"`
	URL        string         `json:"release_url"`
	Prerelease bool           `json:"prerelease"`
	Manifest   ParsedManifest `json:"manifest"`
	Assets     []ReleaseAsset `json:"assets"`
}

// verifiedRelease exists only inside one live operation. In particular, it is
// never reconstructed from a caller's serialized ReleaseView or InstallPlan.
type verifiedRelease struct {
	view      ReleaseView
	manifest  []byte
	checksums []byte
}

type releaseMetadata struct {
	ID          int64  `json:"id"`
	Tag         string `json:"tag_name"`
	Draft       *bool  `json:"draft"`
	Prerelease  *bool  `json:"prerelease"`
	Immutable   *bool  `json:"immutable"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Size   int64  `json:"size"`
		Digest string `json:"digest"`
		State  string `json:"state"`
	} `json:"assets"`
}

type ReleaseClient struct{ http *http.Client }

func allowedReleaseURL(u *url.URL) bool {
	if u == nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" {
		return false
	}
	switch u.Host {
	case "api.github.com", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return true
	}
	return false
}

func NewReleaseClient() *ReleaseClient {
	t := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2: true, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 15 * time.Second, MaxResponseHeaderBytes: 64 << 10, DisableCompression: true,
		IdleConnTimeout: 30 * time.Second, MaxIdleConns: 8, MaxConnsPerHost: 4}
	return newReleaseClient(t)
}

func newReleaseClient(transport http.RoundTripper) *ReleaseClient {
	return &ReleaseClient{http: &http.Client{Transport: transport, Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || !allowedReleaseURL(req.URL) {
			return errors.New("release redirect rejected")
		}
		// No auth/cookie jar is used, including on redirects to a public asset CDN.
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
		return nil
	}}}
}

func (c *ReleaseClient) get(ctx context.Context, address, accept string, limit int64, dst io.Writer) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	u, err := url.Parse(address)
	if err != nil || !allowedReleaseURL(u) {
		return 0, errors.New("untrusted release URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return 0, errors.New("invalid release request")
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "mandalore-release-client")
	r, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, errors.New("release request failed, timed out or followed an untrusted redirect; no installation occurred")
	}
	defer r.Body.Close()
	if r.StatusCode == http.StatusNotFound {
		return 0, ErrNoRelease
	}
	if r.StatusCode != http.StatusOK {
		return 0, errors.New("release request was refused or rate-limited; no installation occurred")
	}
	if r.ContentLength > limit {
		return 0, errors.New("release response exceeds its size limit")
	}
	n, err := io.Copy(dst, io.LimitReader(r.Body, limit+1))
	if ctx.Err() != nil {
		return n, ctx.Err()
	}
	if err != nil || n > limit {
		return n, errors.New("release response was incomplete, exceeded its limit or could not be staged")
	}
	return n, nil
}

func (c *ReleaseClient) getJSON(ctx context.Context, path string, out any) error {
	var b bytes.Buffer
	if _, err := c.get(ctx, releaseAPI+path, "application/vnd.github+json", maxReleaseMetadata, &b); err != nil {
		return err
	}
	var object map[string]any
	if err := strictjson.Decode(b.Bytes(), &object, maxReleaseMetadata); err != nil {
		return errors.New("GitHub release metadata is invalid")
	}
	if err := json.Unmarshal(b.Bytes(), out); err != nil {
		return errors.New("GitHub release metadata has invalid field types")
	}
	return nil
}

func (c *ReleaseClient) asset(ctx context.Context, a ReleaseAsset, dst io.Writer) error {
	if a.ID <= 0 || a.Size < 1 || a.Size > MaxBinaryBytes || !validHex(a.SHA256, 64) {
		return errors.New("invalid release asset identity")
	}
	h := sha256.New()
	n, err := c.get(ctx, releaseAPI+"/releases/assets/"+strconv.FormatInt(a.ID, 10), "application/octet-stream", a.Size, io.MultiWriter(dst, h))
	if err != nil {
		return err
	}
	if n != a.Size || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
		return errors.New("downloaded release asset does not match its recorded size and digest")
	}
	return nil
}

func (c *ReleaseClient) commit(ctx context.Context, tag string) (string, error) {
	type object struct {
		Type string `json:"type"`
		SHA  string `json:"sha"`
	}
	var ref struct {
		Ref    string `json:"ref"`
		Object object `json:"object"`
	}
	if err := c.getJSON(ctx, "/git/ref/tags/"+url.PathEscape(tag), &ref); err != nil {
		return "", err
	}
	if ref.Ref != "refs/tags/"+tag {
		return "", errors.New("release tag reference does not match")
	}
	obj := ref.Object
	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		if !validHex(obj.SHA, 40) || seen[obj.SHA] {
			return "", errors.New("invalid or cyclic release tag identity")
		}
		if obj.Type == "commit" {
			return obj.SHA, nil
		}
		if obj.Type != "tag" {
			return "", errors.New("release tag does not identify a commit")
		}
		seen[obj.SHA] = true
		var next struct {
			SHA    string `json:"sha"`
			Object object `json:"object"`
		}
		if err := c.getJSON(ctx, "/git/tags/"+obj.SHA, &next); err != nil {
			return "", err
		}
		if next.SHA != obj.SHA {
			return "", errors.New("annotated release tag identity changed")
		}
		obj = next.Object
	}
	return "", errors.New("release tag chain exceeds its limit")
}

func (c *ReleaseClient) Inspect(ctx context.Context, version string) (ReleaseView, error) {
	v, err := c.inspectVersion(ctx, version)
	return v.view, err
}

func (c *ReleaseClient) inspectVersion(ctx context.Context, version string) (verifiedRelease, error) {
	if version != "" && !ValidVersion(version) {
		return verifiedRelease{}, errors.New("select a canonical release version or leave it empty for latest")
	}
	path := "/releases/latest"
	if version != "" {
		path = "/releases/tags/" + url.PathEscape("v"+version)
	}
	return c.inspectVerified(ctx, path, version, version == "")
}

func (c *ReleaseClient) inspect(ctx context.Context, path, version string, latest bool) (ReleaseView, error) {
	v, err := c.inspectVerified(ctx, path, version, latest)
	return v.view, err
}

func (c *ReleaseClient) inspectVerified(ctx context.Context, path, version string, latest bool) (verifiedRelease, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	var r releaseMetadata
	if err := c.getJSON(ctx, path, &r); err != nil {
		return verifiedRelease{}, err
	}
	v := strings.TrimPrefix(r.Tag, "v")
	_, dateErr := time.Parse(time.RFC3339, r.PublishedAt)
	base, _, _ := strings.Cut(v, "+")
	if r.ID <= 0 || r.Tag != "v"+v || !ValidVersion(v) || r.Draft == nil || r.Prerelease == nil || r.Immutable == nil || *r.Draft || !*r.Immutable || dateErr != nil || latest && (*r.Prerelease || strings.Contains(base, "-")) || version != "" && v != version {
		return verifiedRelease{}, errors.New("release must be published, immutable and match the explicitly selected version; latest excludes prereleases")
	}
	if len(r.Assets) != 8 {
		return verifiedRelease{}, errors.New("release asset inventory is incomplete or unexpected")
	}
	assets := map[string]ReleaseAsset{}
	ids := map[int64]bool{}
	for _, a := range r.Assets {
		digest := strings.TrimPrefix(a.Digest, "sha256:")
		if a.ID <= 0 || ids[a.ID] || a.Digest != "sha256:"+digest || !validHex(digest, 64) || a.State != "uploaded" || a.Size < 1 || a.Size > MaxBinaryBytes || assets[a.Name].ID != 0 {
			return verifiedRelease{}, errors.New("release asset identity, state, size or digest is invalid")
		}
		assets[a.Name] = ReleaseAsset{ID: a.ID, Name: a.Name, Size: a.Size, SHA256: digest}
		ids[a.ID] = true
	}
	manifestAsset, sumsAsset := assets["manifest.json"], assets["SHA256SUMS"]
	if manifestAsset.Size > MaxManifestBytes || sumsAsset.Size > MaxManifestBytes {
		return verifiedRelease{}, errors.New("release manifest or checksums exceed size limits")
	}
	var raw, sums bytes.Buffer
	if err := c.asset(ctx, manifestAsset, &raw); err != nil {
		return verifiedRelease{}, err
	}
	m, err := ParseManifest(raw.Bytes())
	if err != nil {
		return verifiedRelease{}, err
	}
	if m.Manifest.Tag != r.Tag || m.Manifest.Version != v {
		return verifiedRelease{}, errors.New("release and manifest version disagree")
	}
	commit, err := c.commit(ctx, r.Tag)
	if err != nil {
		return verifiedRelease{}, err
	}
	if m.Manifest.SourceCommit != commit {
		return verifiedRelease{}, errors.New("release tag and manifest source commit disagree")
	}
	if err := c.asset(ctx, sumsAsset, &sums); err != nil {
		return verifiedRelease{}, err
	}
	want, err := Checksums(raw.Bytes())
	if err != nil || !bytes.Equal(want, sums.Bytes()) {
		return verifiedRelease{}, errors.New("release checksums do not match its manifest")
	}
	for _, a := range m.Manifest.Assets {
		remote := assets[a.Name]
		if remote.ID == 0 || remote.Size != a.Size || remote.SHA256 != a.SHA256 {
			return verifiedRelease{}, errors.New("manifest and GitHub asset identities disagree")
		}
	}
	view := ReleaseView{ID: r.ID, URL: "https://github.com/acoz-labs/mandalore/releases/tag/" + url.PathEscape(r.Tag), Prerelease: *r.Prerelease, Manifest: m}
	if err := ctx.Err(); err != nil {
		return verifiedRelease{}, err
	}
	for _, a := range assets {
		view.Assets = append(view.Assets, a)
	}
	sort.Slice(view.Assets, func(i, j int) bool { return view.Assets[i].Name < view.Assets[j].Name })
	return verifiedRelease{view: view, manifest: raw.Bytes(), checksums: sums.Bytes()}, nil
}

// DownloadAsset refreshes the specific release ID, not latest, before streaming
// bytes. dst MUST be disposable staging: failed downloads may have written bytes.
// It never activates a runtime or trusts a serialized preview without rechecking.
func (c *ReleaseClient) DownloadAsset(ctx context.Context, expected ReleaseView, name string, dst io.Writer) error {
	if expected.ID <= 0 {
		return errors.New("invalid pinned release identity")
	}
	current, err := c.inspect(ctx, "/releases/"+strconv.FormatInt(expected.ID, 10), "", false)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, expected) {
		return errors.New("pinned release changed; inspect and approve a new plan")
	}
	for _, a := range current.Assets {
		if a.Name == name {
			return c.asset(ctx, a, dst)
		}
	}
	return errors.New("asset is not declared in the selected release")
}
