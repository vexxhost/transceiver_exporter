#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
rendered_chart="$(mktemp)"
rendered_mixin_json="$(mktemp)"
rendered_chart_json="$(mktemp)"
trap 'rm -f "$rendered_chart" "$rendered_mixin_json" "$rendered_chart_json"' EXIT

"$repo_root/hack/render-alerts.sh" --check
"$repo_root/hack/render-dashboards.sh" --check
promtool check rules "$repo_root/mixin/prometheus_alerts.yaml"
promtool test rules "$repo_root/mixin/tests.yml"

helm template transceiver-exporter "$repo_root/charts/transceiver-exporter" \
  --set podMonitor.enabled=true \
  --set prometheusRule.enabled=true \
  > "$rendered_chart"

grep -q '^kind: PodMonitor$' "$rendered_chart"
grep -q '^kind: PrometheusRule$' "$rendered_chart"
grep -q 'alert: TransceiverFaultActive' "$rendered_chart"

yq -o=json "." "$repo_root/mixin/prometheus_alerts.yaml" > "$rendered_mixin_json"
yq -o=json 'select(.kind == "PrometheusRule") | .spec' "$rendered_chart" > "$rendered_chart_json"
diff -u "$rendered_mixin_json" "$rendered_chart_json"
