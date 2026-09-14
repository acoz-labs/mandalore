package distribution

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func transportFixture(t *testing.T) (CandidateSelection, map[string][]byte, []byte, time.Time) {
	t.Helper()
	dir, m, _ := candidateFixture(t)
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		f, err := w.Create(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	s := CandidateSelection{SourceCommit: m.SourceCommit, RunID: 101, ArtifactID: 202}
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	docs := map[string]any{
		"":                       map[string]any{"id": 303, "full_name": "acoz-labs/mandalore", "default_branch": "main"},
		"/actions/workflows/404": map[string]any{"id": 404, "path": CandidateWorkflow, "state": "active"},
		"/actions/runs/101":      map[string]any{"id": 101, "workflow_id": 404, "path": CandidateWorkflow, "event": "workflow_dispatch", "status": "completed", "conclusion": "success", "head_sha": s.SourceCommit, "head_branch": "main", "run_attempt": 1, "run_started_at": "2026-09-14T10:00:00Z", "updated_at": "2026-09-14T11:00:00Z", "repository": map[string]any{"id": 303, "full_name": "acoz-labs/mandalore"}, "head_repository": map[string]any{"id": 303, "full_name": "acoz-labs/mandalore"}},
		"/actions/artifacts/202": map[string]any{"id": 202, "name": "mandalore-candidate-" + s.SourceCommit + "-1", "size_in_bytes": archive.Len(), "digest": "sha256:" + Digest(archive.Bytes()), "expired": false, "created_at": "2026-09-14T10:30:00Z", "expires_at": "2026-12-13T10:30:00Z", "workflow_run": map[string]any{"id": 101, "repository_id": 303, "head_repository_id": 303, "head_sha": s.SourceCommit, "head_branch": "main"}},
	}
	raw := map[string][]byte{}
	for k, v := range docs {
		raw[k], _ = json.Marshal(v)
	}
	return s, raw, archive.Bytes(), now
}

type transportRoundTrip func(*http.Request) (*http.Response, error)

func (f transportRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func transportClient(t *testing.T, docs map[string][]byte) *ReleaseClient {
	t.Helper()
	return newReleaseClient(transportRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" || r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("unsafe candidate metadata request")
		}
		path := strings.TrimPrefix(r.URL.Path, "/repos/acoz-labs/mandalore")
		b, ok := docs[path]
		if !ok {
			t.Fatalf("unexpected metadata path %s", path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b)), Header: make(http.Header)}, nil
	}))
}

func TestCandidateTransportBindsTrustedOriginAndArchive(t *testing.T) {
	s, docs, archive, now := transportFixture(t)
	c := transportClient(t, docs)
	r, err := c.inspectCandidate(context.Background(), s, now)
	if err != nil {
		t.Fatal(err)
	}
	p, err := VerifyCandidateArchive(context.Background(), bytes.NewReader(archive), int64(len(archive)), r)
	if err != nil || p.Manifest.SourceCommit != s.SourceCommit || r.ArtifactID != s.ArtifactID || r.ArchiveSHA256 != Digest(archive) {
		t.Fatal("wrong candidate", p, r, err)
	}
}

func TestCandidateTransportRejectsWrongProvenance(t *testing.T) {
	for _, tc := range []struct {
		path, field string
		value       any
	}{
		{"", "full_name", "other/project"}, {"", "id", 0}, {"", "default_branch", "develop"},
		{"/actions/workflows/404", "path", ".github/workflows/ci.yml"}, {"/actions/workflows/404", "state", "disabled_manually"},
		{"/actions/runs/101", "id", 102}, {"/actions/runs/101", "head_sha", strings.Repeat("f", 40)},
		{"/actions/runs/101", "path", ".github/workflows/ci.yml"}, {"/actions/runs/101", "event", "pull_request"},
		{"/actions/runs/101", "status", "in_progress"}, {"/actions/runs/101", "conclusion", "failure"},
		{"/actions/runs/101", "run_attempt", 2}, {"/actions/runs/101", "run_started_at", "2026-09-14T10:45:00Z"},
		{"/actions/runs/101", "head_repository", map[string]any{"id": 999, "full_name": "other/fork"}},
		{"/actions/artifacts/202", "id", 203}, {"/actions/artifacts/202", "expired", true},
		{"/actions/artifacts/202", "expires_at", "2026-09-14T11:59:59Z"}, {"/actions/artifacts/202", "digest", ""},
		{"/actions/artifacts/202", "name", "another-artifact"}, {"/actions/artifacts/202", "size_in_bytes", 0},
		{"/actions/artifacts/202", "workflow_run", map[string]any{"id": 101, "repository_id": 303, "head_repository_id": 999, "head_sha": strings.Repeat("f", 40), "head_branch": "main"}},
	} {
		t.Run(tc.path+"/"+tc.field, func(t *testing.T) {
			s, docs, _, now := transportFixture(t)
			var v map[string]any
			json.Unmarshal(docs[tc.path], &v)
			v[tc.field] = tc.value
			docs[tc.path], _ = json.Marshal(v)
			if _, err := transportClient(t, docs).inspectCandidate(context.Background(), s, now); err == nil {
				t.Fatal("untrusted candidate accepted")
			}
		})
	}
}

func TestCandidateTransportRejectsMissingAndDuplicateRequiredMetadata(t *testing.T) {
	for _, field := range []string{"expired", "digest", "workflow_run", "created_at", "expires_at"} {
		s, docs, _, now := transportFixture(t)
		var v map[string]any
		json.Unmarshal(docs["/actions/artifacts/202"], &v)
		delete(v, field)
		docs["/actions/artifacts/202"], _ = json.Marshal(v)
		if _, err := transportClient(t, docs).inspectCandidate(context.Background(), s, now); err == nil {
			t.Fatal("missing field accepted", field)
		}
	}
	s, docs, _, now := transportFixture(t)
	docs[""] = []byte(`{"id":303,"id":303,"full_name":"acoz-labs/mandalore","default_branch":"main"}`)
	if _, err := transportClient(t, docs).inspectCandidate(context.Background(), s, now); err == nil {
		t.Fatal("duplicate metadata accepted")
	}
}

func TestCandidateArchiveRejectsChangedBytesSourceAndSpecialEntries(t *testing.T) {
	s, docs, archive, now := transportFixture(t)
	r, err := transportClient(t, docs).inspectCandidate(context.Background(), s, now)
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]byte(nil), archive...)
	bad[len(bad)/2] ^= 1
	if _, err := VerifyCandidateArchive(context.Background(), bytes.NewReader(bad), int64(len(bad)), r); err == nil {
		t.Fatal("changed transport accepted")
	}
	wrong := r
	wrong.SourceCommit = strings.Repeat("f", 40)
	if _, err := VerifyCandidateArchive(context.Background(), bytes.NewReader(archive), int64(len(archive)), wrong); err == nil {
		t.Fatal("wrong source accepted")
	}
	for _, name := range []string{"../escape", "/absolute", "nested/manifest.json", "manifest.json"} {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		z, _ := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		for _, f := range z.File {
			dst, _ := w.Create(f.Name)
			src, _ := f.Open()
			io.Copy(dst, src)
			src.Close()
		}
		dst, _ := w.Create(name)
		dst.Write([]byte("foreign"))
		w.Close()
		badReceipt := r
		badReceipt.ArchiveSHA256 = Digest(b.Bytes())
		badReceipt.ArchiveSize = int64(b.Len())
		if _, err := VerifyCandidateArchive(context.Background(), bytes.NewReader(b.Bytes()), int64(b.Len()), badReceipt); err == nil {
			t.Fatal("unsafe inventory accepted", name)
		}
	}
}

func TestCandidateArchiveBoundsDirectoryBeforeParsing(t *testing.T) {
	s, docs, archive, now := transportFixture(t)
	r, err := transportClient(t, docs).inspectCandidate(context.Background(), s, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"count", "directory-size", "disk", "offset"} {
		b := append([]byte(nil), archive...)
		end := b[len(b)-22:]
		switch mode {
		case "count":
			binary.LittleEndian.PutUint16(end[10:], 65535)
		case "directory-size":
			binary.LittleEndian.PutUint32(end[12:], 1<<24)
		case "disk":
			binary.LittleEndian.PutUint16(end[4:], 1)
		case "offset":
			binary.LittleEndian.PutUint32(end[16:], 0)
		}
		x := r
		x.ArchiveSHA256 = Digest(b)
		if _, err := VerifyCandidateArchive(context.Background(), bytes.NewReader(b), int64(len(b)), x); err == nil {
			t.Fatal("malformed ZIP accepted", mode)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := VerifyCandidateArchive(ctx, bytes.NewReader(archive), int64(len(archive)), r); err == nil {
		t.Fatal("cancelled archive check succeeded")
	}
}
