# Decision

Use one small shared shell eligibility check: exact repository, source SHA, full artifact identity, issue 2–12, configured owner matching workflow actor, and literal true owner confirmation. Keep ACCEPTANCE_ACTORS authorization before this check. No wildcard, alternate-account impersonation, broad self-approval switch or automatic approval. Disable by clearing the owner variable; different candidates require a separately reviewed change.
