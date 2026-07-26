#!/usr/bin/env sh
set -eu

budget_file="${1:-scripts/file-size-budgets.json}"
default_budget="$(jq -r '.default_lines' "$budget_file")"

source_extensions='(\.go|\.sh|\.ya?ml|\.toml|\.md|\.json)$'
excluded_paths='(^|/)(node_modules|target|dist|build|deps|_build|site|fixtures|vendor|\.git)(/|$)|(^|/)(go\.sum|manifest\.lock|.*\.lock)$'

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

git ls-files |
  grep -E "$source_extensions" |
  grep -Ev "$excluded_paths" > "$tmp"

failures=0
while read -r file; do
  test -f "$file" || continue
  budget="$(jq -r --arg file "$file" '.files[$file] // empty' "$budget_file")"
  if [ -z "$budget" ]; then
    budget="$default_budget"
  fi
  lines="$(wc -l < "$file" | tr -d ' ')"
  if [ "$lines" -gt "$budget" ]; then
    printf '%s: %s lines > budget %s\n' "$file" "$lines" "$budget" >&2
    failures=$((failures + 1))
  fi
done < "$tmp"

if [ "$failures" -gt 0 ]; then
  exit 1
fi
