#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
rendered="$(mktemp -d)"
trap 'rm -rf "$rendered"' EXIT

jsonnet "$repo_root/mixin/dashboards.jsonnet" > "$rendered/dashboards.json"
while IFS= read -r dashboard; do
  yq -o=json -I=2 ".\"$dashboard\"" "$rendered/dashboards.json" > "$rendered/$dashboard"
done < <(yq -r 'keys | .[]' "$rendered/dashboards.json")
rm -f "$rendered/dashboards.json"

if [[ "${1:-}" == "--check" ]]; then
  diff -ur "$repo_root/grafana" "$rendered"
  exit 0
fi

install -d -m 0755 "$repo_root/grafana"
for dashboard in "$rendered"/*.json; do
  install -m 0644 "$dashboard" "$repo_root/grafana/$(basename "$dashboard")"
done
