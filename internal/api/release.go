package api

import (
	"context"

	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type ReleaseInspectInput struct {
	Version   string `json:"version,omitempty" jsonschema:"Explicit published release version; omitted selects latest stable. Mutually exclusive with candidate."`
	Candidate string `json:"candidate,omitempty" jsonschema:"Explicit local candidate directory; no network or executable invocation."`
}

type ReleaseInspection struct {
	Kind      string                       `json:"kind"`
	Candidate string                       `json:"candidate,omitempty"`
	Local     *distribution.ParsedManifest `json:"local,omitempty"`
	Published *distribution.ReleaseView    `json:"published,omitempty"`
	Notice    string                       `json:"notice"`
}

type releaseFailure struct{ err error }

func (e *releaseFailure) Error() string { return e.err.Error() }
func (e *releaseFailure) Unwrap() error { return e.err }

var releases = []Operation{
	operation("release_inspect", "Inspect a selected local candidate or official published release. CLI-only, no signet binding, execution, installation or memory changes. Published inspection verifies manifest/checksum bytes and asset metadata, not all executable payloads.", true,
		func(ctx context.Context, _ *memory.Service, in ReleaseInspectInput) (ReleaseInspection, error) {
			if in.Version != "" && (in.Candidate != "" || !distribution.ValidVersion(in.Version)) {
				return ReleaseInspection{}, strictjson.ErrInvalid
			}
			if in.Candidate != "" {
				p, err := distribution.VerifyDirectory(in.Candidate)
				if err != nil {
					return ReleaseInspection{}, &releaseFailure{err}
				}
				return ReleaseInspection{Kind: "local-candidate", Candidate: in.Candidate, Local: &p, Notice: "All declared local bytes and plugin content checked. Explicit local-source trust is still required; no executable, installation, network, signet or native behavior was tested or changed."}, nil
			}
			r, err := distribution.NewReleaseClient().Inspect(ctx, in.Version)
			if err != nil {
				return ReleaseInspection{}, &releaseFailure{err}
			}
			return ReleaseInspection{Kind: "github-release", Published: &r, Notice: "Official immutable release identity, tag commit, manifest/checksum bytes and declared asset metadata checked. Executable payloads were not downloaded or run. No installation or memory change occurred."}, nil
		}),
}

func init() {
	releases[0].CLIOnly = true
	releases[0].RequiresBinding = false
	releases[0].Network = true
}
