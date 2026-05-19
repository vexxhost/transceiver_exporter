# transceiver-exporter

Helm chart for deploying `transceiver_exporter` as a host-networked DaemonSet.

The chart follows the same shape as Prometheus community hardware exporter
charts: the exporter runs on every Linux node, exposes a named `metrics` port,
and can be discovered by Prometheus Operator through a `PodMonitor`. It does not
create a `ServiceMonitor` because scraping the DaemonSet pods directly is enough
for this exporter.

## Install

```sh
helm install transceiver-exporter charts/transceiver-exporter \
  --set podMonitor.enabled=true
```

## PrometheusRule Alerts

The chart can render the same alert rules as the Jsonnet mixin:

```sh
helm install transceiver-exporter charts/transceiver-exporter \
  --set podMonitor.enabled=true \
  --set prometheusRule.enabled=true
```

The alert payload is generated from `mixin/alerts.jsonnet` into
`charts/transceiver-exporter/files/prometheus-alerts.yaml`. Run
`./hack/render-alerts.sh` after editing the mixin.

## Interface Selection

By default, the exporter discovers physical non-loopback interfaces. To restrict
collection, set `interfaces`; each value renders as its own repeated
`--interface` flag.

```yaml
interfaces:
  - ens3f0np0
  - ens3f1np1
```
