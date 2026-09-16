# Quota presentation — candidate-source engineering checkpoint

Fresh recordings from clean candidate source
`bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`, linked to #74/#88.
These are **source-bound test executables, not the retained release executable**.
The production renderer and keyboard handling run against explicit synthetic
API responses. The test seam is absent from the production executable. Neither
live GitHub quota exhaustion nor exact-artifact network acceptance is claimed.

[The manifest](evidence.json) binds the unchanged driver SHA-256 hashes and eight
passing recordings. Its SHA-256 is
`1631b02ee2295a99fe7cb4c2f6eeeeef3c7dc1b03a5522c63d0f2a433bd963ce`.
macOS arm64, Go 1.26.4, real Herdr terminal, English, 30 rows. No personal
connection, credentials, memories, installation or system trust was changed.

## Fresh scenario matrix

| Recording | Action and observed result |
| --- | --- |
| [Known timing](timing.recording) | 80-column color. Sixty-second lower bound and UTC reset are advisory; later availability is not guaranteed. No automatic retry. |
| [Unknown timing](unknown-narrow.recording) | 32-column plain/NO_COLOR. Unknown timing is explicit; wrapped guidance remains readable. |
| [Ordinary refusal](ordinary-nocolor.recording) | 80-column NO_COLOR. Refusal is not mislabeled as quota exhaustion. |
| [Partial state](partial.recording) | 80-column color. Down/Enter chooses Yes from default No. Receipt retains incomplete activation, old/new runtimes, pending record, explicit original-executable retry and possible prior effects. |
| [Narrow partial state](partial-narrow.recording) | 32-column plain/NO_COLOR, `2`/Enter. Same fields and warning survive; recovery command wraps and is not safely copyable verbatim (#77). |
| [Default No](default-no.recording) | 80-column color. Enter invokes zero applies. |
| [Back](back.recording) | 80-column color. Escape invokes zero applies. |
| [Effect regressions, corrected cwd](effect-checks-cwd.recording) | Compiled distribution tests with fixture HTTP/filesystem effects. Fresh/update each plan4 + apply9 =13; replay0; full verification13 with eight distinct assets. Explicit retry preserves prior runtime/fresh destination on refusal and succeeds after the synthetic refusal clears. |

The effect run also covers independent concurrent verification, missing/corrupt
private verification state, changed source/tag/assets, original manifest bytes,
quota-header boundaries, cancellation precedence, unsafe redirects, anonymous
transport, publisher integrity and bootstrap verification/refusals. These are
fixture tests, not live publication. The apply verifier is inert: do not describe
these as published-payload execution or retained-runtime installation acceptance.
The partial UI response is composition coverage, not evidence of a production
network call after activation.

## Controller error retained

The initial [effect-checks recording](effect-checks.recording) is **failed and
excluded from the passing matrix**. The broader test selection added a bootstrap
test which expects its package cwd. Running it from the lab directory could not
find `../../packaging/install.sh`. The other quota tests passed in that run, but
the run as a whole did not. Only the capture launcher was corrected to enter
`internal/distribution`; the same unchanged test executable then passed the full
selection. Neither product source nor fixtures were adjusted to make it pass.
The failed recording's SHA-256 is
`d3353e27557ab4dfdd33394cea4e34071b95e69c8fb33ec88b6c16c98c939627`.

## Method and scoped review

Compile once from clean source with pinned `go test -c` for `./cmd/mandalore`
and `./internal/distribution`. Run the menu driver with
`-test.run '^TestReleaseQuotaNativePresentation$' -test.v`, setting
`MANDALORE_QUOTA_NATIVE_SCENARIO` to timing/unknown/ordinary/partial/default-no.
Set `MANDALORE_QUOTA_NATIVE_PLAIN=1` for plain mode. Set NO_COLOR for no-color
cases; otherwise unset it and use TERM=xterm-256color. The effects selection is
`-test.run 'Test(Release|EfficientRelease|Bootstrap|Publication|Publisher)' -test.v`
from the source's `internal/distribution` directory.

All files are unedited BSD `script -qr` recordings. The capture child sets its
dimensions and restores terminal settings on exit; actual outer size afterward
was 67 rows by 309 columns. Replay with macOS `script -p RECORDING` in a compatible
terminal. A verifier parsed the recording frames and output, checked complete
PASS/restoration markers, source cleanliness, unchanged executables, critical
receipt/advisory text and absence of SGR color in no-color captures. It also
checked absence of private home paths; only synthetic receipt paths are present.

Contributor rendered review inspected hierarchy, normal/narrow wrapping,
color-independent failure labels, default-No/Back and preservation of partial
effects. Engineering verdict: pass for this source-bound synthetic matrix,
with the existing #77 copyability limitation retained. This is neither human
acceptance nor a relaxation of original issue criteria. There is no approved
pixel baseline, screen-reader certification, alternate locale/font or other-
platform native claim. No release, personal activation or candidate rebuild.

Full pinned host `mise exec -- bin/ci` passed after adding this evidence: 22 Pi
tests, Go race tests/vet and four-target builds (valid Go test-cache hits retained).
The documented host fallback was used; the container daemon was unavailable in
this lab. Public copies were checked against original recording hashes and
scanned for private home paths and credential markers before retention.
