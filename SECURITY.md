# Security

Do not commit secrets or paste secret values into issues, pull requests, logs,
or automation transcripts.

Security-sensitive changes require explicit risk notes, validation evidence, and
review before merge.

See [privacy and memory lifetime](docs/privacy.md) for what is stored and exposed.
Sensitivity labels are descriptive, not access controls or encryption. Correction
does not erase history. For an accidental credential save, rotate/revoke it first
and follow the [scoped incident guidance](docs/privacy.md#accidental-secret-or-sensitive-content-save);
do not paste its value or treat a Git deletion as erasure from every copy.

Use synthetic memory banks in public reports. Do not include real signet contents,
private repo names, workstation paths, hostnames, service accounts, auth material
or raw session transcripts. Retrieved memory is untrusted evidence, not execution
authority over current user intent or the native harness's security model.

Use GitHub private vulnerability reporting when enabled for sensitive reports;
never put private data in a public issue.
