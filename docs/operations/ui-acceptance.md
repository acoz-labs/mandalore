# UI Acceptance Evidence

## Purpose And Scope

Meaningful user-interface changes require evidence that another reviewer can
inspect after the test session ends. Functional browser tests, visual-regression
comparisons, and product judgment answer different questions:

- functional tests prove behavior and semantic contracts;
- visual regression detects unintended change from an approved baseline; and
- product review judges whether a new or materially changed experience is fit
  for its intended task.

No one layer substitutes for the others. This contract applies to browser and
native interfaces, including layout, interaction, responsive behavior,
accessibility, content hierarchy, and visual presentation.

## Classification And Product Judgment

Classify each change in the issue and implementation pull request:

- `no-rendered-impact` — explain why the UI evidence gate does not apply;
- `in-pattern-visual-change` — review against existing product patterns and any
  approved visual baseline; or
- `new-or-materially-changed-experience` — complete product-design review and
  judge the rendered result against its task, design intent, and acceptance
  criteria.

A first screenshot cannot approve itself. Automated screenshot comparison can
protect an already approved interface, but it cannot decide that a new interface
is understandable, appropriately effortless, or visually coherent.

## Scenario Matrix

Before implementation, define the smallest matrix that exposes the change's
material risks. Include, as applicable:

- routes, screens, and critical task journeys;
- loading, empty, partial, error, validation, success, and recovery states;
- representative roles, permissions, and content lengths;
- desktop, mobile, and risk-relevant intermediate breakpoints;
- themes, locales, text direction, reduced motion, zoom, and input modes; and
- browsers or native platforms whose rendering or integration can differ.

Record omitted surfaces and why they are lower risk. Do not multiply screenshots
that provide no distinct evidence.

## Implementation Evidence Gate

Contributor verification is bound to the exact implementation pull-request
head. Before a meaningful UI pull request leaves draft:

1. Exercise the scenario matrix in the real rendered application using stable
   fixtures or named representative data where practical.
2. Run the repository's functional, accessibility, and visual checks.
3. Capture openable desktop and mobile evidence for each materially distinct
   visual state.
4. Inspect relevant console and network failures.
5. Record the evidence manifest in the pull request and retain artifacts long
   enough for review and release reconciliation.
6. Obtain independent review of the rendered result against the design intent
   or approved baseline.

A prose claim such as `responsive checks pass` without openable rendered
evidence is incomplete. A mutable development preview may help answer a bounded
question, but it is not durable pull-request evidence or product acceptance.

## Visual Regression

Add automated visual comparisons for stable, high-risk screens and states where
semantic assertions would miss a material regression. Prefer a small,
intentional baseline over snapshots of every page.

- Generate and compare baselines in one pinned rendering environment.
- Stabilize fonts, animations, clocks, random data, and network-dependent
  content.
- Mask only irreducibly dynamic regions and document every mask.
- Retain expected, current, and diff artifacts when a comparison fails.
- Require review before updating an approved baseline; a new baseline or a
  passing threshold is not approval.

Use the repository's existing test stack when practical. A repository may use
browser-native screenshots, a hosted review service, or a native GUI driver,
but the tool must preserve the evidence contract above. Record the chosen
backend, baseline location, update command, comparison command, and artifact
retention below.

## Exact-Candidate Acceptance

Product acceptance is independent of contributor verification. For a deployed
service, an acceptor other than the implementing contributor repeats every
criterion that depends on hands-on behavior, deployed data, third-party
content, browser integration, or product judgment against the exact staged
commit and immutable artifact.

Acceptance evidence must be fresh, openable, and bound to the candidate. It
records the deployment, scenario matrix, screenshots or recordings, relevant
console/network results, accessibility checks, limitations, and decision. A
local preview or implementation screenshot cannot be relabeled as staging
acceptance.

For an artifact without staging, bind the same evidence to the exact nominated
artifact. For a non-deployable repository, document the equivalent immutable
review surface rather than inventing an environment.

## Evidence Manifest

Record one compact manifest in the pull request and again for product
acceptance:

- commit SHA and, for acceptance, immutable artifact plus deployment;
- environment and URL or application identity;
- scenario matrix and representative fixture or data identity;
- browser/platform, viewport, theme, locale, and input mode;
- actions performed and expected outcome;
- links to screenshots, recordings, traces, visual baselines/diffs, or CI runs;
- distinct functional, visual, console/network, and accessibility results;
- reviewed and unreviewed surfaces with limitations; and
- `pass`, `changes-required`, or `blocked` verdict plus follow-up issues.

## Failure Policy

An agreed-scope visual, interaction, responsive, or accessibility failure keeps
the pull request in implementation or returns the candidate to `In Progress`.
An unrelated non-blocking improvement becomes a separately prioritized linked
issue; it does not weaken the original acceptance criteria.

## Repository-Specific Configuration

Replace this section as the repository adopts visual automation:

```text
UI test command: not configured
Visual comparison backend: not configured
Pinned rendering environment: not configured
Baseline location and approval command: not configured
Current/diff artifact retention: not configured
Staging capture command: not configured
Required real-browser or native-platform checks: not configured
Manual accessibility checks: defined per issue risk
```
