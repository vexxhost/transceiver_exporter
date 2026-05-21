{
  local cfg = $._config,
  local datasource = {
    type: 'prometheus',
    uid: cfg.grafanaDatasourceUid,
  },
  local labelSet(extra=[]) = std.join(',', [cfg.selector] + extra),
  local selector(extra=[]) = '{%s}' % labelSet(extra),
  local scopedSelector(extra=[]) = selector(['instance=~"$instance"', 'interface=~"$interface"'] + extra),
  local instanceSelector(extra=[]) = selector(['instance=~"$instance"'] + extra),
  local alertSelector = '{alertstate="firing",alertname=~"Transceiver.*",instance=~"$instance"}',
  local metric(name, extra=[]) = '%s%s' % [name, scopedSelector(extra)],
  local threshold(metricName, severity, boundary) =
    metric('transceiver_diagnostic_threshold', [
      'metric="%s"' % metricName,
      'severity="%s"' % severity,
      'boundary="%s"' % boundary,
    ]),
  local dbmThreshold(metricName, severity, boundary) =
    '10 * log10(%s)' % threshold(metricName, severity, boundary),
  local target(refId, expr, legend=null, range=true, instant=false, format=null) =
    {
      editorMode: 'code',
      expr: expr,
      range: range,
      refId: refId,
    } +
    (if legend == null then {} else { legendFormat: legend }) +
    (if instant then { instant: true } else {}) +
    (if format == null then {} else { format: format }),
  local grid(h, w, x, y) = {
    h: h,
    w: w,
    x: x,
    y: y,
  },
  local fieldConfig(defaults) = {
    defaults: defaults,
    overrides: [],
  },
  local thresholdSteps(steps) = {
    mode: 'absolute',
    steps: [
      {
        color: step.color,
        value: if std.objectHas(step, 'value') then step.value else null,
      }
      for step in steps
    ],
  },
  local statOptions(colorMode='value', graphMode='area') = {
    colorMode: colorMode,
    graphMode: graphMode,
    justifyMode: 'center',
    orientation: 'auto',
    reduceOptions: {
      calcs: ['lastNotNull'],
      fields: '',
      values: false,
    },
    textMode: 'value',
  },
  local panel(id, title, type, description, gridPos, targets, defaults, options, extra={}) =
    {
      datasource: datasource,
      description: description,
      fieldConfig: fieldConfig(defaults),
      gridPos: gridPos,
      id: id,
      options: options,
      targets: targets,
      title: title,
      type: type,
    } + extra,
  local statPanel(id, title, description, gridPos, expr, legend, defaults, options=statOptions()) =
    panel(
      id,
      title,
      'stat',
      description,
      gridPos,
      [target('A', expr, legend, range=false, instant=true)],
      defaults,
      options,
    ),
  local timeSeriesDefaults(unit, decimals) = {
    color: {
      mode: 'palette-classic',
    },
    custom: {
      axisBorderShow: false,
      axisCenteredZero: false,
      axisColorMode: 'text',
      axisPlacement: 'auto',
      barAlignment: 0,
      drawStyle: 'line',
      fillOpacity: 8,
      gradientMode: 'none',
      hideFrom: {
        legend: false,
        tooltip: false,
        viz: false,
      },
      lineInterpolation: 'linear',
      lineWidth: 2,
      pointSize: 4,
      scaleDistribution: {
        type: 'linear',
      },
      showPoints: 'never',
      spanNulls: false,
      stacking: {
        group: 'A',
        mode: 'none',
      },
      thresholdsStyle: {
        mode: 'off',
      },
    },
    decimals: decimals,
    unit: unit,
  },
  local timeSeriesOptions = {
    legend: {
      calcs: ['lastNotNull', 'min', 'max'],
      displayMode: 'table',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'multi',
      sort: 'none',
    },
  },
  local thresholdTargets(metricName, lane=false, convertToDbm=false) =
    local suffix = if lane then ' L{{lane}}' else '';
    local expr(severity, boundary) =
      if convertToDbm then
        dbmThreshold(metricName, severity, boundary)
      else
        threshold(metricName, severity, boundary);
    [
      target('B', expr('warning', 'high'), 'Warn High {{instance}} {{interface}}%s' % suffix),
      target('C', expr('alarm', 'high'), 'Alarm High {{instance}} {{interface}}%s' % suffix),
      target('D', expr('warning', 'low'), 'Warn Low {{instance}} {{interface}}%s' % suffix),
      target('E', expr('alarm', 'low'), 'Alarm Low {{instance}} {{interface}}%s' % suffix),
    ],
  local diagnosticPanel(
    id,
    title,
    description,
    gridPos,
    valueMetric,
    thresholdMetric,
    unit,
    decimals,
    lane=false,
    convertThresholdsToDbm=false
  ) =
    panel(
      id,
      title,
      'timeseries',
      description,
      gridPos,
      [
        target(
          'A',
          metric(valueMetric),
          if lane then '{{instance}} {{interface}} L{{lane}}' else '{{instance}} {{interface}}',
        ),
      ] + thresholdTargets(thresholdMetric, lane=lane, convertToDbm=convertThresholdsToDbm),
      timeSeriesDefaults(unit, decimals),
      timeSeriesOptions,
    ),
  local commonTableExcludes = {
    Time: true,
    Value: true,
    __name__: true,
    container: true,
    endpoint: true,
    job: true,
    namespace: true,
    pod: true,
  },
  local tableDefaults(extra={}) = {
    custom: {
      align: 'auto',
      cellOptions: {
        type: 'auto',
      },
      inspect: false,
    },
  } + extra,
  local tableOptions = {
    cellHeight: 'sm',
    footer: {
      show: false,
    },
    showHeader: true,
  },
  local organize(renameByName, excludeByName=commonTableExcludes) = [
    {
      id: 'organize',
      options: {
        excludeByName: excludeByName,
        renameByName: renameByName,
      },
    },
  ],
  local tablePanel(id, title, description, gridPos, expr, renameByName, excludeByName=commonTableExcludes, defaultsExtra={}) =
    panel(
      id,
      title,
      'table',
      description,
      gridPos,
      [target('A', expr, range=false, instant=true, format='table')],
      tableDefaults(defaultsExtra),
      tableOptions,
      {
        transformations: organize(renameByName, excludeByName),
      },
    ),
  local barGaugeOptions = {
    displayMode: 'gradient',
    orientation: 'horizontal',
    reduceOptions: {
      calcs: ['lastNotNull'],
      fields: '',
      values: false,
    },
    showUnfilled: true,
    text: {},
    textMode: 'value',
  },
  local barGaugePanel(id, title, description, gridPos, expr, thresholds) =
    panel(
      id,
      title,
      'bargauge',
      description,
      gridPos,
      [target('A', expr, 'Lane {{lane}}', range=false, instant=true)],
      {
        color: {
          mode: 'continuous-GrYlRd',
        },
        decimals: 2,
        thresholds: thresholdSteps(thresholds),
        unit: 'short',
      },
      barGaugeOptions,
    ),
  local valueStatPanel(id, title, description, gridPos, expr, legend, unit='none', decimals=0) =
    statPanel(
      id,
      title,
      description,
      gridPos,
      expr,
      legend,
      {
        color: {
          mode: 'palette-classic',
        },
        decimals: decimals,
        unit: unit,
      },
      statOptions(colorMode='background', graphMode='none'),
    ),
  local variable(name, label, query) = {
    current: {
      selected: false,
      text: '',
      value: '',
    },
    datasource: datasource,
    definition: query,
    hide: 0,
    includeAll: false,
    label: label,
    multi: false,
    name: name,
    options: [],
    query: query,
    refresh: 1,
    regex: '',
    skipUrlSync: false,
    sort: 1,
    type: 'query',
  },
  local dashboard = {
    __inputs: [
      {
        name: 'DS_PROMETHEUS',
        label: 'Prometheus',
        description: '',
        type: 'datasource',
        pluginId: 'prometheus',
        pluginName: 'Prometheus',
      },
    ],
    __requires: [
      {
        type: 'grafana',
        id: 'grafana',
        name: 'Grafana',
        version: '11.0.0',
      },
      {
        type: 'panel',
        id: 'stat',
        name: 'Stat',
        version: '',
      },
      {
        type: 'panel',
        id: 'timeseries',
        name: 'Time series',
        version: '',
      },
      {
        type: 'panel',
        id: 'table',
        name: 'Table',
        version: '',
      },
      {
        type: 'panel',
        id: 'bargauge',
        name: 'Bar gauge',
        version: '',
      },
    ],
    annotations: {
      list: [
        {
          builtIn: 1,
          datasource: {
            type: 'grafana',
            uid: '-- Grafana --',
          },
          enable: true,
          hide: true,
          iconColor: 'rgba(0, 211, 255, 1)',
          name: 'Annotations & Alerts',
          type: 'dashboard',
        },
      ],
    },
    editable: true,
    fiscalYearStartMonth: 0,
    graphTooltip: 1,
    id: null,
    links: [],
    liveNow: false,
    panels: [
      statPanel(
        1,
        'Module Inventory',
        'Count of optical module inventory records in the selected scope.',
        grid(5, 6, 0, 0),
        'count(%s) or vector(0)' % metric('transceiver_module_info'),
        'modules',
        {
          color: {
            mode: 'thresholds',
          },
          mappings: [
            {
              options: {
                '0': {
                  color: 'red',
                  text: 'No module',
                },
              },
              type: 'value',
            },
          ],
          thresholds: thresholdSteps([{ color: 'red' }, { color: 'green', value: 1 }]),
          unit: 'none',
        },
      ),
      statPanel(
        2,
        'Scrape Health',
        'Minimum scrape success across the selected instance and interface.',
        grid(5, 6, 6, 0),
        'min(%s) or vector(0)' % metric('transceiver_scrape_success'),
        'scrape',
        {
          color: {
            mode: 'thresholds',
          },
          mappings: [
            {
              options: {
                '0': {
                  color: 'red',
                  text: 'Failed',
                },
                '1': {
                  color: 'green',
                  text: 'Healthy',
                },
              },
              type: 'value',
            },
          ],
          thresholds: thresholdSteps([{ color: 'red' }, { color: 'green', value: 1 }]),
          unit: 'none',
        },
      ),
      statPanel(
        3,
        'Active EEPROM Flags',
        'Active raw EEPROM warning, alarm, or fault flags for the selected interface.',
        grid(5, 6, 12, 0),
        'sum(%s == 1) or vector(0)' % metric('transceiver_alarm_status'),
        'flags',
        {
          color: {
            mode: 'thresholds',
          },
          thresholds: thresholdSteps([{ color: 'green' }, { color: 'yellow', value: 1 }, { color: 'red', value: 3 }]),
          unit: 'none',
        },
      ),
      statPanel(
        4,
        'Firing Prometheus Alerts',
        'Current firing Prometheus alerts whose names start with Transceiver.',
        grid(5, 6, 18, 0),
        'count(ALERTS%s) or vector(0)' % alertSelector,
        'alerts',
        {
          color: {
            mode: 'thresholds',
          },
          thresholds: thresholdSteps([{ color: 'green' }, { color: 'yellow', value: 1 }, { color: 'red', value: 2 }]),
          unit: 'none',
        },
      ),
      diagnosticPanel(
        5,
        'Module Temperature',
        'Live module temperature over time with warning and alarm thresholds.',
        grid(8, 12, 0, 5),
        'transceiver_temperature_celsius',
        'temperature_celsius',
        'celsius',
        2,
      ),
      diagnosticPanel(
        6,
        'Module Voltage',
        'Module supply voltage with warning and alarm thresholds.',
        grid(8, 12, 12, 5),
        'transceiver_voltage_volts',
        'voltage_volts',
        'volt',
        3,
      ),
      diagnosticPanel(
        7,
        'RX Optical Power (dBm)',
        'Receive optical power in dBm with warning and alarm thresholds converted from milliwatts.',
        grid(8, 12, 0, 13),
        'transceiver_rx_power_dbm',
        'rx_power_milliwatts',
        'short',
        2,
        lane=true,
        convertThresholdsToDbm=true,
      ),
      diagnosticPanel(
        8,
        'TX Optical Power (dBm)',
        'Transmit optical power in dBm with warning and alarm thresholds converted from milliwatts.',
        grid(8, 12, 12, 13),
        'transceiver_tx_power_dbm',
        'tx_power_milliwatts',
        'short',
        2,
        lane=true,
        convertThresholdsToDbm=true,
      ),
      diagnosticPanel(
        9,
        'TX Bias Current (mA)',
        'Transmit bias current with warning and alarm thresholds.',
        grid(8, 12, 0, 21),
        'transceiver_tx_bias_milliamps',
        'tx_bias_milliamps',
        'short',
        2,
        lane=true,
      ),
      tablePanel(
        10,
        'Lane Datapath State',
        'Current active lane datapath state, if the module exposes lane state.',
        grid(8, 12, 12, 21),
        '%s == 1' % metric('transceiver_lane_datapath_state'),
        {
          format: 'Format',
          instance: 'Instance',
          interface: 'Interface',
          lane: 'Lane',
          state: 'State',
        },
      ),
      barGaugePanel(
        11,
        'Current RX Power by Lane',
        'Instant receive optical power by lane.',
        grid(7, 8, 0, 29),
        metric('transceiver_rx_power_dbm'),
        [{ color: 'red' }, { color: 'yellow', value: -10 }, { color: 'green', value: -5 }],
      ),
      barGaugePanel(
        12,
        'Current TX Power by Lane',
        'Instant transmit optical power by lane.',
        grid(7, 8, 8, 29),
        metric('transceiver_tx_power_dbm'),
        [{ color: 'red' }, { color: 'yellow', value: -8 }, { color: 'green', value: -3 }],
      ),
      barGaugePanel(
        13,
        'Current TX Bias by Lane',
        'Instant transmit bias current by lane.',
        grid(7, 8, 16, 29),
        metric('transceiver_tx_bias_milliamps'),
        [{ color: 'green' }, { color: 'yellow', value: 8 }, { color: 'red', value: 15 }],
      ),
      tablePanel(
        14,
        'Optical Module Inventory',
        'Optical module identity and static EEPROM inventory labels.',
        grid(10, 24, 0, 36),
        metric('transceiver_module_info'),
        {
          cable_kind: 'Cable Kind',
          cable_standard: 'Cable Standard',
          connector: 'Connector',
          date_code: 'Date Code',
          encoding: 'Encoding',
          form_factor: 'Form Factor',
          format: 'Format',
          instance: 'Instance',
          interface: 'Interface',
          interface_technology: 'Interface Technology',
          media_type: 'Media Type',
          part_number: 'Part Number',
          rate_identifier: 'Rate Identifier',
          revision: 'Revision',
          serial_number: 'Serial Number',
          vendor: 'Vendor',
          vendor_oui: 'Vendor OUI',
        },
      ),
      tablePanel(
        15,
        'Supported Link Lengths',
        'Supported link lengths by media type, when exposed by the optical module EEPROM.',
        grid(8, 12, 0, 46),
        metric('transceiver_link_length_meters'),
        {
          Value: 'Max Length (m)',
          format: 'Format',
          instance: 'Instance',
          interface: 'Interface',
          medium: 'Medium',
        },
        excludeByName={
          Time: true,
          __name__: true,
          container: true,
          endpoint: true,
          job: true,
          namespace: true,
          pod: true,
        },
        defaultsExtra={
          decimals: 0,
          unit: 'm',
        },
      ),
      valueStatPanel(
        16,
        'Nominal Bitrate (Mbd)',
        'Nominal signaling rate in megabaud for the selected module.',
        grid(4, 6, 12, 46),
        metric('transceiver_nominal_bitrate_mbd'),
        'Mbd',
      ),
      valueStatPanel(
        17,
        'Wavelength (nm)',
        'Media wavelength in nanometers when exposed by the module.',
        grid(4, 6, 18, 46),
        metric('transceiver_wavelength_nanometers'),
        'nm',
        decimals=2,
      ),
      valueStatPanel(
        18,
        'Module Power Class',
        'Advertised module power class when exposed by the module.',
        grid(4, 6, 12, 50),
        metric('transceiver_module_power_class'),
        'class',
      ),
      valueStatPanel(
        19,
        'Module Power Max (W)',
        'Advertised maximum module power in watts when exposed by the module.',
        grid(4, 6, 18, 50),
        metric('transceiver_module_power_max_watts'),
        'watts',
        unit='watt',
        decimals=2,
      ),
      tablePanel(
        20,
        'Active EEPROM Warning / Alarm / Fault Flags',
        'Raw EEPROM flags that are currently active for the selected interface.',
        grid(8, 12, 0, 54),
        '%s == 1' % metric('transceiver_alarm_status'),
        {
          alarm: 'Alarm',
          format: 'Format',
          instance: 'Instance',
          interface: 'Interface',
          lane: 'Lane',
          severity: 'Severity',
        },
      ),
      tablePanel(
        21,
        'Firing Prometheus Alerts',
        'Current firing Prometheus alerts related to the transceiver exporter.',
        grid(8, 12, 12, 54),
        'ALERTS%s' % alertSelector,
        {
          alertname: 'Alert',
          alertstate: 'State',
          format: 'Format',
          instance: 'Instance',
          interface: 'Interface',
          lane: 'Lane',
          part_number: 'Part Number',
          serial_number: 'Serial Number',
          severity: 'Severity',
          vendor: 'Vendor',
        },
        excludeByName={
          Time: true,
          Value: true,
          __name__: true,
        },
      ),
    ],
    refresh: '1m',
    schemaVersion: 39,
    style: 'dark',
    tags: ['network', 'optics', 'prometheus', 'transceiver'],
    templating: {
      list: [
        variable(
          'instance',
          'Instance',
          'label_values(transceiver_scrape_success%s, instance)' % selector(),
        ),
        variable(
          'interface',
          'Interface',
          'label_values(transceiver_scrape_success%s, interface)' % instanceSelector(),
        ),
      ],
    },
    time: {
      from: 'now-6h',
      to: 'now',
    },
    timezone: '',
    title: 'Transceiver Exporter / Optical Monitoring',
    uid: 'transceiver-exporter-optics',
    version: 1,
    weekStart: '',
  },

  grafanaDashboards+:: {
    'transceiver-exporter-optical-monitoring.json': dashboard,
  },
}
