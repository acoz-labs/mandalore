#!/bin/sh
# Explicit setup projects this private bridge alongside the public package.
mandalore_connection=$(dirname "$0")/connection.sh
if [ ! -f "$mandalore_connection" ]; then
  if [ "${1:-}" = hook ]; then
    printf '%s\n' '{"systemMessage":"Mandalore has no explicit connection. Run connection setup; no memory was changed."}'
    exit 0
  fi
  printf '%s\n' 'Mandalore has no explicit connection.' >&2
  exit 1
fi
exec /bin/sh "$mandalore_connection" "$@"
