# Solution-Design Plans

Use one temporary directory per issue while a substantial change is in
`Solution Design`:

```text
docs/plans/<issue-number>-<short-slug>/
```

Copy `_template/` for the standard planning pack. A narrow change may combine
the material into one `plan.md`; a broad design may split `03-design.md` into
smaller views when that improves review. Preserve the same questions and final
metadata either way.

The contributor authors one planning-only pull request. Maintainers review it
with ordinary inline comments and review summaries. Intermediate findings are
resolved inside that pull request; the product authority receives one clean
final design gate rather than one approval request per document.

The final planning pull request must:

- reference its issue with `Refs #<number>`;
- change only its issue plan directory;
- record the source commit and explicit execution envelope;
- contain no unresolved blockers or placeholders;
- be independently approved and merged before the issue moves to `Ready`.

During implementation, the plan is the intent baseline. Before the
implementation pull request leaves draft, reconcile the actual diff with the
plan, promote the shipped contract into durable architecture, contract, ADR,
development, deployment, runbook, or user documentation, and delete the
temporary issue plan. The merged planning pull request and Git history preserve
the design journey; active repository docs describe what actually ships.

Do not archive completed plans under `docs/plans/`. If implementation is
abandoned, remove the plan in a small pull request and record the reason on the
issue.
