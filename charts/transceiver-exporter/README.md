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

When scraping with `kube-prometheus-stack`, enabling `podMonitor.enabled` is not
enough on its own in the default configuration. The Prometheus instance usually
selects `PodMonitor` objects by label, so the generated `PodMonitor` must carry
the stack release label. This chart exposes that through
`podMonitor.additionalLabels`.

Example for a stack release named `kube-prometheus-stack` installed in the `monitoring` namespace:

```sh
helm install transceiver-exporter charts/transceiver-exporter \
  -n monitoring \
  --set podMonitor.enabled=true \
  --set podMonitor.additionalLabels.release=kube-prometheus-stack
```

If the exporter is installed into a different namespace from Prometheus, also
make sure the Prometheus `podMonitorNamespaceSelector` allows that namespace.
The chart's CI values mirror the common `kube-prometheus-stack` case in
`charts/transceiver-exporter/ci/podmonitor-values.yaml`.

## PrometheusRule Alerts

The chart can render the same alert rules as the Jsonnet mixin:

```sh
helm install transceiver-exporter charts/transceiver-exporter \
  -n monitoring \
  --set podMonitor.enabled=true \
  --set prometheusRule.enabled=true \
  --set prometheusRule.namespace=monitoring \
  --set prometheusRule.additionalLabels.release=kube-prometheus-stack
```

The alert payload is generated from `mixin/alerts.jsonnet` into
`charts/transceiver-exporter/files/prometheus-alerts.yaml`. Run
`./hack/render-alerts.sh` after editing the mixin.

## Interface Selection

By default, the exporter discovers physical non-loopback interfaces. To
restrict collection, set `interfaces`; each value renders as its own repeated
`--interface` flag.

```yaml
interfaces:
  - ens3f0np0
  - ens3f1np1
```
