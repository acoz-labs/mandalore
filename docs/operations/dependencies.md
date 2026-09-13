# Dependency Stewardship

Dependency updates are automated into pull requests, not straight into
production.

The managed baseline uses GitHub-native dependency tooling first:

- Dependabot raises version-update pull requests on a weekly schedule.
- Dependabot cooldown delays normal version updates so brand-new releases have
  time to be withdrawn, yanked, or flagged before they enter this repo.
- Dependabot security updates are treated as a fast lane and must not wait on
  the normal cooldown.
- Dependency review can run on pull requests that change manifests or lockfiles
  and block known high-risk additions when GitHub Code Security is available for
  the repository. Enable it by setting the repo variable
  `ENABLE_DEPENDENCY_REVIEW=true`.
- The dependency merge steward performs the final merge decision itself. It does
  not rely on GitHub native auto-merge, because private repositories on GitHub
  Free may not have server-side branch protection or required status checks.

## Update Classes

| Class | Cooldown | Automation | Review |
|---|---:|---|---|
| Security update | none | PR immediately | maintainer review if runtime, auth, deploy, or data-adjacent |
| Development patch | 7 days | eligible for steward merge after checks | optional |
| Development minor | 14 days | PR only by default | normal review until this repo proves boring |
| Runtime patch | 14 days | PR only by default | maintainer review |
| Runtime minor | 14 days | PR only by default | maintainer review |
| Major update | 30 days | PR only | issue required before merge |
| New dependency | issue required | no auto-merge | explicit trust and maintenance review |
| GitHub Actions update | 14 days | patch only may steward-merge | review workflow and permission impact |
| Docker/base image update | 14 days, 30 for major | no auto-merge | review build and deploy impact |

## Required Checks

Every dependency PR must pass:

- repo CI
- dependency review when available
- any repo-specific security scanner
- staging/release gates before production deploy, when this repo has production

If GitHub dependency review is unavailable for this private repository, leave
`ENABLE_DEPENDENCY_REVIEW` unset, record that as a repo-specific deviation in
`docs/operations/repo-standard.md`, and keep the PR manual-review rule in place.

## Merge Steward Rules

The merge steward is deliberately narrow:

- only Dependabot-authored PRs
- only semver patch version updates
- only after the configured cooldown has elapsed
- only after non-steward checks pass
- only for non-runtime dependency patches or GitHub Actions patch updates
- never for new dependencies, major updates, Docker/base images, or production
  deploy policy changes

The steward labels eligible PRs with `dependency:auto-merge-candidate`, then
re-evaluates them after CI finishes and on a daily GitHub Actions recovery
sweep. Event-driven evaluation remains the primary path; the scheduled sweep
exists only to recover candidates missed because an event was delayed. It
does a normal merge only after checking the PR author, candidate label, check
state, draft state, and mergeability.

Expanding steward merge beyond this scope requires an SDLC change and must
cascade to existing managed repos.

## Package Ecosystems

Keep `.github/dependabot.yml` matched to the repo's real package managers.
Remove unused ecosystems instead of leaving noisy broken checks behind.

Common ecosystems:

- `bundler` for Ruby apps with `Gemfile`
- `npm` for Node apps with `package.json`
- `github-actions` for workflow dependencies
- `docker` for production Dockerfiles and base images

## Supply Chain Notes

- Prefer first-party or widely maintained packages.
- Treat abandoned or deprecated packages as replacement work, not routine
  upgrade work.
- Prefer lockfiles for reproducible installs.
- Prefer short-lived OIDC credentials over long-lived deploy secrets when the
  target platform supports it.
- Pin GitHub Actions to immutable SHAs where the repo's maintenance burden
  justifies it; otherwise keep action updates visible through Dependabot and
  review workflow permission changes carefully.
