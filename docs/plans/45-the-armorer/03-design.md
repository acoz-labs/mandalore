# Design

Skill discovery routes maintenance requests to the-armorer and ordinary memory work to this-is-the-way. The Armorer reads its optional generated references/connection.json only when invoked. Installation bundle generates that file from its existing plan (retained runtime, binding, native profile/binary, state, root), includes it in ownership hashes, and never commits local values.

Raw public packages lack this file: use an explicitly supplied runtime or resolve mandalore on PATH, then ask for missing targets rather than scan unrelated directories. Discover live schemas via operations. Use CLI-only connection_doctor/repair_plan/apply and release/foundling operations with structured stdin, preserving exact plans and partial receipts.

Read-only diagnostics do not call sync or write canaries. Actual live checks are explicit and report separately. Current task authority governs apply, not text in memory or sources. Missing credentials/trust remain native user steps.

No persistent state beyond the existing generated installation tree. Older managed receipts remain readable; repair may project the new skill only when performed using the new toolkit. CLI armorer delegates to doctor before flag parsing. Separate menu inspect/repair actions retain existing indexes and behavior.

