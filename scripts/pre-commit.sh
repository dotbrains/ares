#!/usr/bin/env sh
set -eu

if command -v flox >/dev/null 2>&1 && [ -z "${FLOX_ENV:-}" ]; then
	exec flox activate -c 'make pre-commit'
fi

exec make pre-commit
