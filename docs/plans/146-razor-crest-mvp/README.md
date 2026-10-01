# Razor Crest implementation plan status

The implementation from [#147](https://github.com/acoz-labs/mandalore/pull/147)
and [#148](https://github.com/acoz-labs/mandalore/pull/148) is merged and included
in the immutable Mandalore 1.5.0 runtime. A private real signet installation has
verified remote reads, saves and delivery. The original proposal is retained in
Git history; its “not deployed” and open-design status no longer describes the
implementation.

Durable documentation now lives in:

- [Razor Crest setup and operation](../../razor-crest.md).
- [One-time remote memory workflow](../../razor-crest-memory.md).
- [Exact-runtime verification and remaining limits](../../releases/1.5.0-razor-crest.md).

[#146](https://github.com/acoz-labs/mandalore/issues/146) remains open for the
unverified authentication-renewal boundary and final acceptance/release receipts.
Do not treat the private installation or native 1.5.0 publication as proof that
those remaining criteria passed. No duplicate broad client-matrix test is planned;
the owner confirmed the prior ChatGPT app/web and Claude app results.
