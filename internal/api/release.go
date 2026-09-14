package api

import (
	"context"
	"encoding/json"

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

type releaseFailure struct {
	err    error
	result *distribution.InstallResult
}

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
					return ReleaseInspection{}, &releaseFailure{err: err}
				}
				return ReleaseInspection{Kind: "local-candidate", Candidate: in.Candidate, Local: &p, Notice: "All declared local bytes and plugin content checked. Explicit local-source trust is still required; no executable, installation, network, signet or native behavior was tested or changed."}, nil
			}
			r, err := distribution.NewReleaseClient().Inspect(ctx, in.Version)
			if err != nil {
				return ReleaseInspection{}, &releaseFailure{err: err}
			}
			return ReleaseInspection{Kind: "github-release", Published: &r, Notice: "Official immutable release identity, tag commit, manifest/checksum bytes and declared asset metadata checked. Executable payloads were not downloaded or run. No installation or memory change occurred."}, nil
		}),
	operation("release_plan", "Preview an exact CLI installation or retained-version selection. CLI-only and read-only; no binary execution, launcher activation, native connection changes or signet binding. Published selection requires public network access.", true,
		func(ctx context.Context, _ *memory.Service, in distribution.InstallOptions) (distribution.InstallPlan, error) {
			if err := in.Validate(); err != nil {
				return distribution.InstallPlan{}, strictjson.ErrInvalid
			}
			p, err := distribution.PlanInstall(ctx, in)
			if err != nil {
				return distribution.InstallPlan{}, &releaseFailure{err: err}
			}
			return p, nil
		}),
	operation("release_apply", "Apply an explicitly reviewed CLI installation plan. Revalidates source and destination, retains verified runtime bytes, and changes only an owned launcher/receipt. Matching pending activation can resume. No signet or native connection changes; CLI-only.", false,
		func(ctx context.Context, _ *memory.Service, in distribution.InstallPlan) (distribution.InstallResult, error) {
			raw, _ := json.Marshal(in)
			if _, err := distribution.ParseInstallPlan(raw); err != nil {
				return distribution.InstallResult{}, strictjson.ErrInvalid
			}
			r, err := distribution.ApplyInstall(ctx, in)
			if err != nil {
				return r, &releaseFailure{err: err, result: &r}
			}
			return r, nil
		}),
}

func init() {
	for i := range releases {
		releases[i].CLIOnly = true
		releases[i].RequiresBinding = false
		releases[i].Network = true
	}
}
