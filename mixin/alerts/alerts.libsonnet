{
  local cfg = $._config,
  local withModuleInfo(expr) = |||
    (
      %(expr)s
    )
    * on(cluster, job, instance, interface, format) group_left(vendor, part_number, serial_number, form_factor)
      transceiver_module_info{%(selector)s}
  ||| % { expr: std.stripChars(expr, ' \n\t'), selector: cfg.selector },
  local activeAlarmExpr(severity) = |||
    (
      transceiver_alarm_status{%(selector)s,severity="%(severity)s",lane=""} == 1
    )
    or
    (
      (transceiver_alarm_status{%(selector)s,severity="%(severity)s",lane!=""} == 1)
      and on(cluster, job, instance, interface, format, lane)
        (transceiver_lane_datapath_state{%(selector)s,state="activated"} == 1)
    )
  ||| % { selector: cfg.selector, severity: severity },

  prometheusAlerts+:: {
    groups+: [
      {
        name: 'transceiver-exporter',
        rules: [
          {
            alert: 'TransceiverExporterDown',
            expr: 'up{%s} == 0' % cfg.selector,
            'for': '15m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Transceiver telemetry is unavailable.',
              description: 'Prometheus has not scraped transceiver_exporter on {{ $labels.instance }} for 15 minutes, so optical module health for this target is unknown.',
              runbook_url: cfg.runbookURLPattern % 'transceiverexporterdown',
            },
          },
          {
            alert: 'TransceiverScrapeFailed',
            expr: 'transceiver_scrape_success{%s} == 0' % cfg.selector,
            'for': '15m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Transceiver EEPROM scrape failed.',
              description: 'transceiver_exporter is reachable, but it has not been able to read or decode EEPROM data for {{ $labels.instance }} interface {{ $labels.interface }} for 15 minutes.',
              runbook_url: cfg.runbookURLPattern % 'transceiverscrapefailed',
            },
          },
          {
            alert: 'TransceiverWarningActive',
            expr: withModuleInfo(activeAlarmExpr('warning')),
            'for': '30m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Transceiver warning threshold is active.',
              description: '{{ $labels.instance }} interface {{ $labels.interface }} lane {{ $labels.lane }} reports {{ $labels.alarm }} warning on {{ $labels.vendor }} {{ $labels.part_number }} {{ $labels.serial_number }}. This is a ticket-level early warning unless it correlates with link errors or service impact.',
              runbook_url: cfg.runbookURLPattern % 'transceiverwarningactive',
            },
          },
          {
            alert: 'TransceiverAlarmActive',
            expr: withModuleInfo(activeAlarmExpr('alarm')),
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Transceiver alarm threshold is active.',
              description: '{{ $labels.instance }} interface {{ $labels.interface }} lane {{ $labels.lane }} reports {{ $labels.alarm }} alarm on {{ $labels.vendor }} {{ $labels.part_number }} {{ $labels.serial_number }}. The module is outside vendor operating thresholds and may lose signal.',
              runbook_url: cfg.runbookURLPattern % 'transceiveralarmactive',
            },
          },
          {
            alert: 'TransceiverFaultActive',
            expr: withModuleInfo(activeAlarmExpr('fault')),
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Transceiver fault is active.',
              description: '{{ $labels.instance }} interface {{ $labels.interface }} lane {{ $labels.lane }} reports {{ $labels.alarm }} fault on {{ $labels.vendor }} {{ $labels.part_number }} {{ $labels.serial_number }}. This is a module-reported loss-of-signal, loss-of-lock, or transmitter fault symptom.',
              runbook_url: cfg.runbookURLPattern % 'transceiverfaultactive',
            },
          },
          {
            alert: 'TransceiverModuleFault',
            expr: withModuleInfo('transceiver_module_status{%s,state="fault"} == 1' % cfg.selector),
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Transceiver module is in fault state.',
              description: '{{ $labels.instance }} interface {{ $labels.interface }} reports module state fault {{ $labels.fault }} on {{ $labels.vendor }} {{ $labels.part_number }} {{ $labels.serial_number }}.',
              runbook_url: cfg.runbookURLPattern % 'transceivermodulefault',
            },
          },
          {
            alert: 'TransceiverModuleLowPower',
            expr: withModuleInfo('transceiver_module_low_power_status{%s} == 1' % cfg.selector),
            'for': '30m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Transceiver module is requesting low-power mode.',
              description: '{{ $labels.instance }} interface {{ $labels.interface }} has low-power request source {{ $labels.source }} active on {{ $labels.vendor }} {{ $labels.part_number }} {{ $labels.serial_number }}. Investigate if this interface is expected to carry production traffic.',
              runbook_url: cfg.runbookURLPattern % 'transceivermodulelowpower',
            },
          },
        ],
      },
    ],
  },
}
