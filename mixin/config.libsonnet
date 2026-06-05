{
  _config+:: {
    selector: 'job="transceiver-exporter"',
    grafanaDatasourceUid: '${DS_PROMETHEUS}',
    runbookURLPattern: 'https://github.com/vexxhost/transceiver_exporter/tree/main/mixin/runbook.md#alert-name-%s',
  },
}
