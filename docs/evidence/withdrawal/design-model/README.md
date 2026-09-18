# Historical withdrawal design evidence

These unchanged scripts and the recorded old-reader result originated in the
#81 compatibility discovery. They preserve the design-stage evidence separately
from the implemented runtime scenarios in [the parent evidence](../README.md).

`model.mjs` and `model.test.mjs` model causal content/visibility decisions; run
them with `node --test model.test.mjs`. This small independent model is not the
production validator or a migration implementation.

`old-reader-probe.mjs` takes an explicit released CLI path and creates disposable
synthetic fixtures. `old-reader-result.json` records the original characterization:
old readers refuse format 2, while an unchanged offline clone retains old content.
It does not demonstrate an implemented upgrade or revoke prior knowledge.

The current contract is in the signet-format documentation and ADR0004. Historical
discovery rationale remains available through Git history and discovery PR121.
