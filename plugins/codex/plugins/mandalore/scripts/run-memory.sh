#!/bin/sh
# Thin public bridge. Environment selection is machine-local, never in the signet.
mandalore_runtime=${MANDALORE_BIN:-mandalore}
case ${MANDALORE_BIN:-} in
  ''|/*) ;;
  *)
    if [ "${1:-}" = hook ]; then
      printf '%s\n' '{"systemMessage":"Mandalore runtime override must be an absolute executable path; no memory was changed."}'
      exit 0
    fi
    printf '%s\n' 'Mandalore runtime override must be an absolute executable path.' >&2
    exit 1
    ;;
esac

case ${1:-} in
  hook)
    # A missing/older runtime must not block unrelated work or leak failed output.
    if mandalore_output=$("$mandalore_runtime" codex-memory-hook 2>/dev/null); then
      printf '%s\n' "$mandalore_output"
    else
      printf '%s\n' '{"systemMessage":"Mandalore memory hook is unavailable. Check MANDALORE_BIN, the installed version and local binding; automatic recall did not run."}'
    fi
    ;;
  mcp) exec "$mandalore_runtime" mcp --harness codex ;;
  *) exit 1 ;;
esac
