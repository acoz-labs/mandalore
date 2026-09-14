package distribution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// PublicationReceipt records the last verified phase and possible incomplete
// operation. It is recovery evidence, never nomination or acceptance authority.
type PublicationReceipt struct {
	Identity         string         `json:"identity"`
	SourceCommit     string         `json:"source_commit"`
	Phase            string         `json:"phase"`
	ReleaseID        int64          `json:"release_id,omitempty"`
	URL              string         `json:"release_url,omitempty"`
	Assets           []ReleaseAsset `json:"assets,omitempty"`
	PendingOperation string         `json:"pending_operation,omitempty"`
}

type publicationMetadata struct {
	releaseMetadata
	Name   string `json:"name"`
	Body   string `json:"body"`
	Target string `json:"target_commitish"`
}

// Publisher does not discover credentials, configure policy, nominate candidates
// or close issues. Its caller MUST establish trusted transport, exact independent
// acceptance and explicit publication authority before invoking Publish. Comparing
// an expected identity is byte binding, not evidence of that authorization.
type Publisher struct {
	token               string
	api, policy, public *ReleaseClient
}

// NewPublisher accepts explicit credentials only. policyToken needs repository
// Administration:read for the immutable-releases setting; the publisher needs
// Contents:write. An empty policy token uses the release token and fails closed
// if it cannot read the setting. No repository setting is changed automatically.
func NewPublisher(releaseToken, policyToken string) *Publisher {
	return newPublisher(releaseToken, policyToken, NewReleaseClient().http.Transport)
}

func newPublisher(token, policyToken string, base http.RoundTripper) *Publisher {
	if policyToken == "" {
		policyToken = token
	}
	api := newReleaseClient(publicationTransport{base, token, false})
	api.http.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		// Only binary GETs may redirect to the public asset CDN. Metadata and
		// mutations never redirect; credentials cannot be replayed on a new host.
		if len(via) > 5 || len(via) == 0 || via[0].Method != "GET" || via[0].Header.Get("Accept") != "application/octet-stream" || !publicationAssetPath.MatchString(via[0].URL.Path) || !publicationCDN(r.URL) {
			return errors.New("publisher redirect refused")
		}
		r.Header.Del("Authorization")
		r.Header.Del("Cookie")
		return nil
	}
	policy := newReleaseClient(publicationTransport{base, policyToken, true})
	policy.http.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("policy redirect refused") }
	return &Publisher{token: token, api: api, policy: policy, public: newReleaseClient(base)}
}

var publicationAssetPath = regexp.MustCompile(`^/repos/acoz-labs/mandalore/releases/assets/[1-9][0-9]*$`)
var publicationIDPath = regexp.MustCompile(`^/repos/acoz-labs/mandalore/releases/[1-9][0-9]*$`)
var publicationUploadPath = regexp.MustCompile(`^/repos/acoz-labs/mandalore/releases/[1-9][0-9]*/assets$`)
var publicationTagPath = regexp.MustCompile(`^/repos/acoz-labs/mandalore/git/(ref/tags/v[0-9A-Za-z.+-]+|tags/[0-9a-f]{40})$`)

func publicationCDN(u *url.URL) bool {
	return allowedReleaseURL(u) && (u.Host == "release-assets.githubusercontent.com" || u.Host == "objects.githubusercontent.com")
}

type publicationTransport struct {
	base   http.RoundTripper
	token  string
	policy bool
}

func (t publicationTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	u := r.URL
	allowed := false
	if u.Scheme == "https" && u.User == nil && u.Fragment == "" && u.RawPath == "" {
		if t.policy {
			allowed = r.Method == "GET" && u.Host == "api.github.com" && u.Path == "/repos/acoz-labs/mandalore/immutable-releases" && u.RawQuery == ""
		} else {
			switch u.Host {
			case "api.github.com":
				if u.RawQuery == "" {
					allowed = r.Method == "GET" && (publicationIDPath.MatchString(u.Path) || publicationAssetPath.MatchString(u.Path) || publicationTagPath.MatchString(u.Path)) || r.Method == "PATCH" && publicationIDPath.MatchString(u.Path) || r.Method == "POST" && u.Path == "/repos/acoz-labs/mandalore/releases"
				}
				if r.Method == "GET" && u.Path == "/repos/acoz-labs/mandalore/releases" {
					q := u.Query()
					page, err := strconv.Atoi(q.Get("page"))
					allowed = err == nil && page >= 1 && page <= 10 && len(q) == 2 && len(q["page"]) == 1 && q.Get("per_page") == "100" && len(q["per_page"]) == 1
				}
			case "uploads.github.com":
				q := u.Query()
				name := q.Get("name")
				allowed = r.Method == "POST" && publicationUploadPath.MatchString(u.Path) && len(q) == 1 && len(q["name"]) == 1 && name != "" && !strings.ContainsAny(name, "/\\\r\n")
			default:
				allowed = r.Method == "GET" && publicationCDN(u)
			}
		}
	}
	if !allowed {
		return nil, errors.New("publisher endpoint is not permitted")
	}
	r = r.Clone(r.Context())
	r.Header.Del("Authorization")
	r.Header.Del("Cookie")
	if !publicationCDN(u) && t.token != "" {
		r.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(r)
}

func (p *Publisher) requireImmutablePolicy(ctx context.Context) error {
	var settings struct {
		Enabled *bool `json:"enabled"`
	}
	if err := p.policy.getJSON(ctx, "/immutable-releases", &settings); err != nil {
		return errors.New("cannot verify immutable-release policy; a deliberately authorized Administration:read credential is required; no policy change was attempted")
	}
	if settings.Enabled == nil || !*settings.Enabled {
		return errors.New("immutable releases must be explicitly enabled before publication; no policy change was attempted")
	}
	return nil
}

func publicationBody(m ParsedManifest) string {
	return "Mandalore " + m.Manifest.Version + "\n\n- Commit: `" + m.Manifest.SourceCommit + "`\n- Artifact: `" + m.Identity() + "`\n"
}

func (p *Publisher) find(ctx context.Context, tag string) (publicationMetadata, error) {
	var selected publicationMetadata
	// Drafts are not reliably discoverable by the published-release tag endpoint.
	// Inspect a bounded authenticated list; reject ambiguity rather than choose one.
	for page := 1; page <= 10; page++ {
		var raw bytes.Buffer
		if _, err := p.api.get(ctx, releaseAPI+"/releases?per_page=100&page="+strconv.Itoa(page), "application/vnd.github+json", maxReleaseMetadata, &raw); err != nil {
			return selected, err
		}
		// strictjson's public contract is object-shaped. A single named array
		// preserves its duplicate-key, nesting, UTF-8 and trailing-data checks.
		var envelope struct {
			Releases []map[string]any `json:"releases"`
		}
		wrapped := append([]byte(`{"releases":`), raw.Bytes()...)
		wrapped = append(wrapped, '}')
		if err := strictjson.Decode(wrapped, &envelope, maxReleaseMetadata+13); err != nil || envelope.Releases == nil {
			return selected, errors.New("invalid publication list")
		}
		var releases []publicationMetadata
		if err := json.Unmarshal(raw.Bytes(), &releases); err != nil || len(releases) > 100 {
			return selected, errors.New("invalid publication list fields")
		}
		for _, r := range releases {
			if r.Tag == tag {
				if selected.Tag != "" {
					return selected, errors.New("multiple releases claim the selected tag")
				}
				selected = r
			}
		}
		if len(releases) < 100 {
			return selected, nil
		}
	}
	return selected, errors.New("release discovery exceeded its bounded history; no release was selected")
}

func (p *Publisher) write(ctx context.Context, method, address, contentType string, b []byte, status int, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, address, bytes.NewReader(b))
	if err != nil {
		return errors.New("invalid publication request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "mandalore-publisher")
	r, err := p.api.http.Do(req)
	if err != nil {
		return errors.New("publication request failed or timed out; remote effects may exist; inspect and retry the same identity")
	}
	defer r.Body.Close()
	if r.StatusCode != status || r.ContentLength > maxReleaseMetadata {
		return errors.New("publication response was refused or invalid; remote effects may exist; existing assets were not replaced")
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxReleaseMetadata+1))
	if err != nil {
		return errors.New("publication response was incomplete; remote effects may exist")
	}
	var object map[string]any
	if err := strictjson.Decode(raw, &object, maxReleaseMetadata); err != nil {
		return errors.New("invalid publication response; remote effects may exist")
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return errors.New("invalid publication response fields; remote effects may exist")
	}
	return ctx.Err()
}

func (p *Publisher) tag(ctx context.Context, m Manifest) error {
	// Only a missing reference permits GitHub to create the tag at publication.
	// A 404 later in an existing annotated-tag chain is a broken tag, not absence.
	var ref map[string]any
	err := p.api.getJSON(ctx, "/git/ref/tags/"+url.PathEscape(m.Tag), &ref)
	if errors.Is(err, ErrNoRelease) {
		return nil
	}
	if err != nil {
		return err
	}
	commit, err := p.api.commit(ctx, m.Tag)
	if err != nil {
		return err
	}
	if commit != m.SourceCommit {
		return errors.New("existing release tag identifies a different source; it was not changed")
	}
	return nil
}

func publicationAssets(r publicationMetadata, m ParsedManifest, want map[string]ReleaseAsset, complete bool) ([]ReleaseAsset, error) {
	base, _, _ := strings.Cut(m.Manifest.Version, "+")
	prerelease := strings.Contains(base, "-")
	if r.ID <= 0 || r.Tag != m.Manifest.Tag || r.Name != "Mandalore "+m.Manifest.Version || r.Body != publicationBody(m) || r.Target != m.Manifest.SourceCommit || r.Draft == nil || r.Prerelease == nil || *r.Prerelease != prerelease || r.Immutable == nil {
		return nil, errors.New("release identity or ownership conflicts with the selected candidate; no overwrite attempted")
	}
	if *r.Draft {
		if *r.Immutable || r.PublishedAt != "" {
			return nil, errors.New("draft publication state is inconsistent")
		}
	} else {
		if _, err := time.Parse(time.RFC3339, r.PublishedAt); err != nil || !*r.Immutable {
			return nil, errors.New("existing published release must already be immutable")
		}
	}
	if len(r.Assets) > len(want) || complete && len(r.Assets) != len(want) {
		return nil, errors.New("publication asset inventory is incomplete or unexpected")
	}
	ids, names := map[int64]bool{}, map[string]bool{}
	var assets []ReleaseAsset
	for _, a := range r.Assets {
		w, ok := want[a.Name]
		if !ok || a.ID <= 0 || ids[a.ID] || names[a.Name] || a.State != "uploaded" || a.Size != w.Size || a.Digest != "sha256:"+w.SHA256 {
			return nil, errors.New("existing publication asset conflicts or is incomplete; no deletion or replacement attempted")
		}
		w.ID = a.ID
		assets = append(assets, w)
		ids[a.ID] = true
		names[a.Name] = true
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Name < assets[j].Name })
	return assets, nil
}

// Publish stages only missing matching assets and publishes only after verifying
// every remote byte. A failure returns its recovery receipt, including uncertain
// writes. Never automatically delete, overwrite, repair policy, build, execute
// the candidate or complete the acceptance/issue ledger.
func (p *Publisher) Publish(ctx context.Context, path, expectedIdentity string) (receipt PublicationReceipt, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	if p.token == "" {
		return receipt, errors.New("publication requires an explicitly supplied credential")
	}
	m, err := VerifyDirectory(path)
	if err != nil {
		return receipt, err
	}
	if m.Identity() != expectedIdentity {
		return receipt, errors.New("candidate differs from the explicitly selected identity")
	}
	receipt = PublicationReceipt{Identity: m.Identity(), SourceCommit: m.Manifest.SourceCommit, Phase: "candidate-verified"}
	dir, err := openInstallDirectory(path)
	if err != nil {
		return receipt, err
	}
	defer dir.Close()
	raw, err := readCandidateFile(dir, "manifest.json", MaxManifestBytes)
	if err != nil || Digest(raw) != m.SHA256 {
		return receipt, errors.New("candidate manifest changed")
	}
	sums, err := Checksums(raw)
	if err != nil {
		return receipt, err
	}
	want := map[string]ReleaseAsset{"manifest.json": {Name: "manifest.json", Size: int64(len(raw)), SHA256: m.SHA256}, "SHA256SUMS": {Name: "SHA256SUMS", Size: int64(len(sums)), SHA256: Digest(sums)}}
	for _, a := range m.Manifest.Assets {
		want[a.Name] = ReleaseAsset{Name: a.Name, Size: a.Size, SHA256: a.SHA256}
	}
	if err := p.tag(ctx, m.Manifest); err != nil {
		return receipt, err
	}
	r, err := p.find(ctx, m.Manifest.Tag)
	if err != nil {
		return receipt, err
	}
	if r.Tag == "" {
		if err := p.requireImmutablePolicy(ctx); err != nil {
			return receipt, err
		}
		base, _, _ := strings.Cut(m.Manifest.Version, "+")
		body, _ := json.Marshal(map[string]any{"tag_name": m.Manifest.Tag, "target_commitish": m.Manifest.SourceCommit, "name": "Mandalore " + m.Manifest.Version, "body": publicationBody(m), "draft": true, "prerelease": strings.Contains(base, "-")})
		receipt.PendingOperation = "create-draft"
		if err := p.write(ctx, "POST", releaseAPI+"/releases", "application/json", body, 201, &r); err != nil {
			return receipt, err
		}
	}
	assets, err := publicationAssets(r, m, want, false)
	if err != nil {
		return receipt, err
	}
	receipt.ReleaseID = r.ID
	receipt.URL = "https://github.com/acoz-labs/mandalore/releases/tag/" + url.PathEscape(m.Manifest.Tag)
	receipt.PendingOperation = ""
	receipt.Assets = assets
	releasePath := "/releases/" + strconv.FormatInt(r.ID, 10)
	if *r.Draft {
		receipt.Phase = "draft-created"
		if err := p.requireImmutablePolicy(ctx); err != nil {
			return receipt, err
		}
		// Verify existing assets before any further mutation, not just metadata.
		present := map[string]bool{}
		for _, a := range assets {
			if err := p.api.asset(ctx, a, io.Discard); err != nil {
				return receipt, err
			}
			present[a.Name] = true
		}
		names := make([]string, 0, len(want))
		for name := range want {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if present[name] {
				continue
			}
			if err := ctx.Err(); err != nil {
				return receipt, err
			}
			a := want[name]
			// Own bounded bytes for the request, so concurrent local file edits
			// cannot change the upload after the digest has been checked.
			b, err := readCandidateFile(dir, name, a.Size)
			if err != nil || int64(len(b)) != a.Size || Digest(b) != a.SHA256 {
				return receipt, errors.New("candidate asset changed before upload; retained draft was not replaced")
			}
			receipt.PendingOperation = "upload:" + name
			var uploaded map[string]any
			if err := p.write(ctx, "POST", "https://uploads.github.com/repos/acoz-labs/mandalore"+releasePath+"/assets?name="+url.QueryEscape(name), "application/octet-stream", b, 201, &uploaded); err != nil {
				return receipt, err
			}
			// The complete authoritative inventory, not the upload response,
			// establishes the resulting asset IDs and detects concurrent changes.
			var current publicationMetadata
			if err := p.api.getJSON(ctx, releasePath, &current); err != nil {
				return receipt, err
			}
			currentAssets, err := publicationAssets(current, m, want, false)
			if err != nil {
				return receipt, err
			}
			if current.ID != r.ID || !*current.Draft || len(currentAssets) != len(assets)+1 {
				return receipt, errors.New("draft changed during upload; inspect partial publication")
			}
			for _, old := range assets {
				found := false
				for _, a := range currentAssets {
					if a == old {
						found = true
					}
				}
				if !found {
					return receipt, errors.New("existing asset changed during upload")
				}
			}
			found := false
			for _, a := range currentAssets {
				if a.Name == name {
					found = true
				}
			}
			if !found {
				return receipt, errors.New("uploaded asset is absent from the resulting draft")
			}
			r, assets = current, currentAssets
			receipt.Assets = assets
			receipt.PendingOperation = ""
		}
		receipt.Phase = "assets-staged"
		if _, err := publicationAssets(r, m, want, true); err != nil {
			return receipt, err
		}
		for _, a := range assets {
			if err := p.api.asset(ctx, a, io.Discard); err != nil {
				return receipt, err
			}
		}
		var current publicationMetadata
		if err := p.api.getJSON(ctx, releasePath, &current); err != nil {
			return receipt, err
		}
		if !reflect.DeepEqual(current, r) {
			return receipt, errors.New("draft changed during remote verification")
		}
		receipt.Phase = "assets-verified"
		if err := p.tag(ctx, m.Manifest); err != nil {
			return receipt, err
		}
		if err := p.requireImmutablePolicy(ctx); err != nil {
			return receipt, err
		}
		receipt.PendingOperation = "publish-draft"
		if err := p.write(ctx, "PATCH", releaseAPI+releasePath, "application/json", []byte(`{"draft":false,"make_latest":"legacy"}`), 200, &current); err != nil {
			return receipt, err
		}
		if _, err := publicationAssets(current, m, want, true); err != nil {
			return receipt, err
		}
		if current.ID != r.ID || *current.Draft {
			return receipt, errors.New("publication response did not identify the selected published release")
		}
		receipt.PendingOperation = ""
	}
	receipt.Phase = "published"
	// Verify the public, anonymous download surface, including every payload.
	view, err := p.public.inspect(ctx, releasePath, m.Manifest.Version, false)
	if err != nil {
		return receipt, err
	}
	if view.ID != r.ID || view.Manifest.Identity() != m.Identity() || !reflect.DeepEqual(view.Assets, receipt.Assets) {
		return receipt, errors.New("published candidate differs from the staged identity")
	}
	for _, a := range view.Assets {
		if err := p.public.asset(ctx, a, io.Discard); err != nil {
			return receipt, err
		}
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	receipt.Phase = "publication-verified"
	return receipt, nil
}
