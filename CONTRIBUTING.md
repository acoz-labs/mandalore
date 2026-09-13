# Contributing

Use discovery issues and proportional `docs/discovery/` pull requests for
ambiguous product decisions. Exact-head approval and merge authorize selected
or deliberately deferred delivery outcomes. Use bounded issues for lifecycle
state, planning pull requests for Solution Design, and implementation pull
requests for shipped changes.

Pull requests should include:

- linked issue
- linked approved planning PR, execution envelope, and current reconciliation
- durable documentation promotion and temporary plan-removal status
- summary
- tests/checks run
- documentation promotion destinations, ADRs, or a docs-not-needed rationale
- deploy impact
- known risks or rollback notes when relevant
- UI acceptance classification and an openable evidence manifest for meaningful
  rendered changes, following `docs/operations/ui-acceptance.md`

Use `Refs #<issue>` for changes whose issue remains open through staging,
product acceptance, or production release. Use `Closes #<issue>` only when
merge itself completes the issue under the repository's documented delivery
profile.
