# Solution Decision

Keep the existing report/receipt architecture. Add an explicit literal command
surface that bypasses hard wrapping only after verifying it is a safe single
line. It has a separate readable label; command bytes have no decoration.

Rejected: OSC52 or OS clipboard dependencies require new permissions and vary by
SSH/terminal; writing helper scripts adds new ownership and cleanup effects;
shell continuation chunking inside quoted paths increases escaping complexity.
Leaving only wrapped text or telling users to repair quotes fails copyability.

The terminal owns visual soft wrapping. Tests prove exact output bytes and shell
meaning; actual terminal evidence separately checks selection/readability. Do not
claim every terminal clipboard joins visual lines identically.
