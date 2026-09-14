package distribution

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

const CandidateWorkflow = ".github/workflows/build-candidate.yml"
const MaxCandidateArchiveBytes = 520 << 20

type CandidateSelection struct {
	SourceCommit string `json:"source_commit"`
	RunID        int64  `json:"run_id"`
	ArtifactID   int64  `json:"artifact_id"`
}

func (s CandidateSelection) Validate() error {
	// Pinned GitHub actions parse IDs as JavaScript numbers. Do not permit an
	// otherwise valid Go integer to select a rounded, different artifact there.
	if !validHex(s.SourceCommit, 40) || s.RunID <= 0 || s.ArtifactID <= 0 || s.RunID > 1<<53-1 || s.ArtifactID > 1<<53-1 {
		return errors.New("candidate selection requires an exact source commit and positive run/artifact IDs")
	}
	return nil
}

// CandidateTransport records provenance inspected through the official API.
// A serialized receipt is not authority by itself: refresh the metadata before
// nomination/promotion and verify the downloaded archive independently.
type CandidateTransport struct {
	CandidateSelection
	FormatVersion int    `json:"format_version"`
	Repository    string `json:"repository"`
	RepositoryID  int64  `json:"repository_id"`
	DefaultBranch string `json:"default_branch"`
	Workflow      string `json:"workflow"`
	WorkflowID    int64  `json:"workflow_id"`
	RunAttempt    int64  `json:"run_attempt"`
	ArtifactName  string `json:"artifact_name"`
	ArchiveSize   int64  `json:"archive_size"`
	ArchiveSHA256 string `json:"archive_sha256"`
	CreatedAt     string `json:"created_at"`
	ExpiresAt     string `json:"expires_at"`
}

type candidateRepository struct {
	ID     int64  `json:"id"`
	Name   string `json:"full_name"`
	Branch string `json:"default_branch"`
}

func candidateName(source string, attempt int64) string {
	return "mandalore-candidate-" + source + "-" + strconv.FormatInt(attempt, 10)
}

func (t CandidateTransport) validate() error {
	created, e1 := time.Parse(time.RFC3339, t.CreatedAt)
	expires, e2 := time.Parse(time.RFC3339, t.ExpiresAt)
	if t.CandidateSelection.Validate() != nil || t.FormatVersion != 1 || t.Repository != "acoz-labs/mandalore" || t.RepositoryID <= 0 || t.DefaultBranch == "" || t.Workflow != CandidateWorkflow || t.WorkflowID <= 0 || t.RunAttempt <= 0 || t.ArtifactName != candidateName(t.SourceCommit, t.RunAttempt) || t.ArchiveSize <= 0 || t.ArchiveSize > MaxCandidateArchiveBytes || !validHex(t.ArchiveSHA256, 64) || e1 != nil || e2 != nil || !expires.After(created) {
		return errors.New("invalid candidate transport receipt")
	}
	return nil
}

// InspectCandidate is anonymous read-only metadata inspection for this public
// repository. It never downloads/executes artifacts or borrows native credentials.
// Archive transport is handled by the pinned workflow download action, not by
// trusting arbitrary URLs from metadata or forwarding a token to an asset host.
func (c *ReleaseClient) InspectCandidate(ctx context.Context, s CandidateSelection) (CandidateTransport, error) {
	t, err := c.inspectCandidate(ctx, s, time.Now().UTC())
	if err != nil {
		if errors.Is(err, ErrNoRelease) {
			return CandidateTransport{}, errors.New("candidate metadata unavailable; the run or artifact may be missing or inaccessible")
		}
		return CandidateTransport{}, err
	}
	expires, _ := time.Parse(time.RFC3339, t.ExpiresAt)
	if !expires.After(time.Now().UTC()) {
		return CandidateTransport{}, errors.New("candidate expired during metadata inspection")
	}
	return t, nil
}

func (c *ReleaseClient) inspectCandidate(ctx context.Context, s CandidateSelection, now time.Time) (CandidateTransport, error) {
	if err := s.Validate(); err != nil {
		return CandidateTransport{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	var repo candidateRepository
	if err := c.getJSON(ctx, "", &repo); err != nil {
		return CandidateTransport{}, err
	}
	if repo.ID <= 0 || repo.Name != "acoz-labs/mandalore" || repo.Branch == "" {
		return CandidateTransport{}, errors.New("candidate repository identity is not trusted")
	}
	var run struct {
		ID             int64               `json:"id"`
		WorkflowID     int64               `json:"workflow_id"`
		Path           string              `json:"path"`
		Event          string              `json:"event"`
		Status         string              `json:"status"`
		Conclusion     string              `json:"conclusion"`
		Source         string              `json:"head_sha"`
		Branch         string              `json:"head_branch"`
		Attempt        int64               `json:"run_attempt"`
		Started        string              `json:"run_started_at"`
		Updated        string              `json:"updated_at"`
		Repository     candidateRepository `json:"repository"`
		HeadRepository candidateRepository `json:"head_repository"`
	}
	if err := c.getJSON(ctx, "/actions/runs/"+strconv.FormatInt(s.RunID, 10), &run); err != nil {
		return CandidateTransport{}, err
	}
	started, e1 := time.Parse(time.RFC3339, run.Started)
	updated, e2 := time.Parse(time.RFC3339, run.Updated)
	if run.ID != s.RunID || run.WorkflowID <= 0 || run.Path != CandidateWorkflow || (run.Event != "workflow_dispatch" && run.Event != "push") || run.Status != "completed" || run.Conclusion != "success" || run.Source != s.SourceCommit || run.Branch != repo.Branch || run.Attempt <= 0 || run.Repository.ID != repo.ID || run.HeadRepository.ID != repo.ID || run.Repository.Name != repo.Name || run.HeadRepository.Name != repo.Name || e1 != nil || e2 != nil || updated.Before(started) || updated.After(now) {
		return CandidateTransport{}, errors.New("candidate was not built by the selected successful same-repository default-branch workflow run")
	}
	var workflow struct {
		ID    int64  `json:"id"`
		Path  string `json:"path"`
		State string `json:"state"`
	}
	if err := c.getJSON(ctx, "/actions/workflows/"+strconv.FormatInt(run.WorkflowID, 10), &workflow); err != nil {
		return CandidateTransport{}, err
	}
	if workflow.ID != run.WorkflowID || workflow.Path != CandidateWorkflow || workflow.State != "active" {
		return CandidateTransport{}, errors.New("candidate workflow identity is not active or does not match")
	}
	var artifact struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Size    int64  `json:"size_in_bytes"`
		Digest  string `json:"digest"`
		Expired *bool  `json:"expired"`
		Created string `json:"created_at"`
		Expires string `json:"expires_at"`
		Run     struct {
			ID               int64  `json:"id"`
			RepositoryID     int64  `json:"repository_id"`
			HeadRepositoryID int64  `json:"head_repository_id"`
			Source           string `json:"head_sha"`
			Branch           string `json:"head_branch"`
		} `json:"workflow_run"`
	}
	if err := c.getJSON(ctx, "/actions/artifacts/"+strconv.FormatInt(s.ArtifactID, 10), &artifact); err != nil {
		return CandidateTransport{}, err
	}
	created, e1 := time.Parse(time.RFC3339, artifact.Created)
	expires, e2 := time.Parse(time.RFC3339, artifact.Expires)
	if artifact.ID != s.ArtifactID || artifact.Name != candidateName(s.SourceCommit, run.Attempt) || artifact.Expired == nil || *artifact.Expired || artifact.Run.ID != run.ID || artifact.Run.RepositoryID != repo.ID || artifact.Run.HeadRepositoryID != repo.ID || artifact.Run.Source != s.SourceCommit || artifact.Run.Branch != repo.Branch || !strings.HasPrefix(artifact.Digest, "sha256:") || e1 != nil || e2 != nil || created.Before(started) || created.After(updated) || !expires.After(now) {
		return CandidateTransport{}, errors.New("candidate artifact is expired, from another run/attempt, or not bound to the selected source")
	}
	t := CandidateTransport{CandidateSelection: s, FormatVersion: 1, Repository: repo.Name, RepositoryID: repo.ID, DefaultBranch: repo.Branch, Workflow: workflow.Path, WorkflowID: workflow.ID, RunAttempt: run.Attempt, ArtifactName: artifact.Name, ArchiveSize: artifact.Size, ArchiveSHA256: strings.TrimPrefix(artifact.Digest, "sha256:"), CreatedAt: artifact.Created, ExpiresAt: artifact.Expires}
	if err := t.validate(); err != nil {
		return CandidateTransport{}, err
	}
	return t, nil
}

// VerifyCandidateArchive does not extract files or execute any payload. It binds
// exact archive bytes to a supplied inspected transport and verifies every member.
// Callers must refresh transport provenance; fabricated local receipts are not
// authenticated by this byte/content check and do not authorize nomination.
func VerifyCandidateArchive(ctx context.Context, reader io.ReaderAt, size int64, t CandidateTransport) (ParsedManifest, error) {
	if err := t.validate(); err != nil {
		return ParsedManifest{}, err
	}
	if size != t.ArchiveSize {
		return ParsedManifest{}, errors.New("candidate archive size differs from transport")
	}
	checkHash := func() error {
		h := sha256.New()
		n, err := io.Copy(h, installContextReader{ctx, io.NewSectionReader(reader, 0, size)})
		if err != nil {
			return err
		}
		if n != size || hex.EncodeToString(h.Sum(nil)) != t.ArchiveSHA256 {
			return errors.New("candidate archive digest differs from transport")
		}
		return nil
	}
	if err := checkHash(); err != nil {
		return ParsedManifest{}, err
	}
	if err := candidateZipLayout(reader, size); err != nil {
		return ParsedManifest{}, err
	}
	z, err := zip.NewReader(reader, size)
	if err != nil || len(z.File) != 8 {
		return ParsedManifest{}, errors.New("candidate archive requires exactly eight regular payload members")
	}
	files := map[string]*zip.File{}
	for _, f := range z.File {
		if f.Name == "" || strings.ContainsAny(f.Name, "/\\") || files[f.Name] != nil || !f.Mode().IsRegular() || f.Flags&1 != 0 || f.UncompressedSize64 == 0 || f.UncompressedSize64 > MaxBinaryBytes {
			return ParsedManifest{}, errors.New("candidate archive contains duplicate, nested, special or oversized members")
		}
		files[f.Name] = f
	}
	read := func(name string, limit int64) ([]byte, error) {
		f := files[name]
		if f == nil || f.UncompressedSize64 > uint64(limit) {
			return nil, errors.New("candidate archive member missing or oversized")
		}
		r, err := f.Open()
		if err != nil {
			return nil, errors.New("candidate archive member cannot be read")
		}
		defer r.Close()
		b, err := io.ReadAll(io.LimitReader(installContextReader{ctx, r}, limit+1))
		if err != nil || int64(len(b)) > limit || uint64(len(b)) != f.UncompressedSize64 {
			return nil, errors.New("candidate archive member is incomplete or corrupt")
		}
		return b, nil
	}
	b, err := read("manifest.json", MaxManifestBytes)
	if err != nil {
		return ParsedManifest{}, err
	}
	p, err := ParseManifest(b)
	if err != nil {
		return ParsedManifest{}, err
	}
	if p.Manifest.SourceCommit != t.SourceCommit || p.Manifest.GoVersion != "go1.26.4" {
		return ParsedManifest{}, errors.New("candidate manifest differs from selected source or pinned toolchain")
	}
	sums, _ := Checksums(b)
	got, err := read("SHA256SUMS", MaxManifestBytes)
	if err != nil || !bytes.Equal(sums, got) {
		return ParsedManifest{}, errors.New("candidate archive checksum inventory does not match")
	}
	for _, a := range p.Manifest.Assets {
		f := files[a.Name]
		if f == nil || f.UncompressedSize64 != uint64(a.Size) {
			return ParsedManifest{}, errors.New("candidate archive payload inventory or size differs from manifest")
		}
		r, err := f.Open()
		if err != nil {
			return ParsedManifest{}, errors.New("candidate payload cannot be read")
		}
		h := sha256.New()
		n, readErr := io.Copy(h, io.LimitReader(installContextReader{ctx, r}, a.Size+1))
		closeErr := r.Close()
		if readErr != nil || closeErr != nil || n != a.Size || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
			return ParsedManifest{}, errors.New("candidate archive payload is incomplete or differs from manifest")
		}
		if a.Kind == "codex-plugin" {
			b, err := read(a.Name, MaxPluginBytes)
			if err != nil {
				return ParsedManifest{}, err
			}
			if err := VerifyPlugin(b, p.Manifest.Version, p.Manifest.PluginSHA256); err != nil {
				return ParsedManifest{}, err
			}
		}
	}
	if err := checkHash(); err != nil {
		return ParsedManifest{}, err
	}
	return p, nil
}

// Bound central-directory parsing before archive/zip allocates entry objects.
// This eight-file, <520 MiB transport never needs ZIP64 or multi-disk archives.
func candidateZipLayout(r io.ReaderAt, size int64) error {
	invalid := errors.New("candidate ZIP directory is malformed, oversized or unsupported")
	if size < 22 {
		return invalid
	}
	n := min(size, 65535+22)
	tail := make([]byte, int(n))
	if _, err := r.ReadAt(tail, size-n); err != nil {
		return invalid
	}
	for i := len(tail) - 22; i >= 0; i-- {
		if binary.LittleEndian.Uint32(tail[i:]) != 0x06054b50 || int(binary.LittleEndian.Uint16(tail[i+20:])) != len(tail)-i-22 {
			continue
		}
		end := tail[i:]
		length := int64(binary.LittleEndian.Uint32(end[12:]))
		offset := int64(binary.LittleEndian.Uint32(end[16:]))
		if binary.LittleEndian.Uint16(end[4:]) != 0 || binary.LittleEndian.Uint16(end[6:]) != 0 || binary.LittleEndian.Uint16(end[8:]) != 8 || binary.LittleEndian.Uint16(end[10:]) != 8 || length <= 0 || length > 65536 || offset+length != size-n+int64(i) {
			return invalid
		}
		return nil
	}
	return invalid
}
