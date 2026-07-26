#!/usr/bin/env sh
set -eu

missing=0
for plugin in marketplace/plugins/builtin/provider-*.toml; do
  provider="$(basename "$plugin" .toml | sed 's/^provider-//')"
  doc="docs/providers/$provider.md"
  if [ "$provider" = "lightsail" ]; then
    doc="docs/providers/lightsail.md"
  fi
  if [ ! -f "$doc" ]; then
    printf '%s: missing provider doc %s\n' "$plugin" "$doc" >&2
    missing=$((missing + 1))
  fi
done

if [ "$missing" -gt 0 ]; then
  exit 1
fi
