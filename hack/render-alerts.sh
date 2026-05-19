#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
rendered="$(mktemp)"
trap 'rm -f "$rendered"' EXIT

jsonnet -S "$repo_root/mixin/alerts.jsonnet" > "$rendered"

if [[ "${1:-}" == "--check" ]]; then
  diff -u "$repo_root/mixin/prometheus_alerts.yaml" "$rendered"
  diff -u "$repo_root/mixin/prometheus_alerts.yaml" "$repo_root/charts/transceiver-exporter/files/prometheus-alerts.yaml"
  exit 0
fi

install -m 0644 "$rendered" "$repo_root/mixin/prometheus_alerts.yaml"
install -m 0644 "$rendered" "$repo_root/charts/transceiver-exporter/files/prometheus-alerts.yaml"
