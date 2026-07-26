#!/usr/bin/env sh
set -eu

budget_file="${1:-scripts/flat-directory-budgets.json}"
default_budget="$(jq -r '.default_files' "$budget_file")"

source_extensions='(\.go|\.sh|\.ya?ml|\.toml|\.md|\.json)$'
excluded_paths='(^|/)(node_modules|target|dist|build|deps|_build|site|fixtures|vendor|\.git)(/|$)|(^|/)(go\.sum|manifest\.lock|.*\.lock)$'

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

git ls-files |
  grep -E "$source_extensions" |
  grep -Ev "$excluded_paths" |
  xargs -n1 dirname |
  sort |
  uniq -c > "$tmp"

failures=0
while read -r count dir; do
  budget="$(jq -r --arg dir "$dir" '.directories[$dir].limit // empty' "$budget_file")"
  if [ -z "$budget" ]; then
    budget="$default_budget"
  fi
  if [ "$count" -gt "$budget" ]; then
    printf '%s: %s direct source files > budget %s\n' "$dir" "$count" "$budget" >&2
    failures=$((failures + 1))
  fi
done < "$tmp"

if [ "$failures" -gt 0 ]; then
  exit 1
fi
