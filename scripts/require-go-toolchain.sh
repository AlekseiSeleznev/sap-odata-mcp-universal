#!/usr/bin/env bash

set -euo pipefail

go_bin="${1:-go}"
go_version="$($go_bin version 2>/dev/null || true)"
version="$(printf '%s\n' "$go_version" | sed -n 's/^go version go\([0-9][0-9]*\)\.\([0-9][0-9]*\).*/\1.\2/p')"

if [[ -z "$version" ]]; then
    printf 'Go >=1.24 is required for Darwin artifacts; unable to parse: %s\n' "$go_version" >&2
    exit 1
fi

major="${version%%.*}"
minor="${version#*.}"
if (( major < 1 || (major == 1 && minor < 24) )); then
    printf 'Go >=1.24 is required for Darwin artifacts; found %s\n' "$go_version" >&2
    exit 1
fi

printf 'Darwin artifact toolchain OK: %s\n' "$go_version"
