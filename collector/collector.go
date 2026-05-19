package collector

import (
	"log/slog"
	"math"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/vexxhost/transceiver_exporter/pkg/netdev"
	"github.com/vexxhost/transceiver_exporter/pkg/sff"
	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

// Reader supplies raw module EEPROM bytes to the collector.
type Reader interface {
	ModuleEEPROM(interfaceName string) ([]byte, error)
}

type decoder func(data []byte) (transceiver.Module, error)

type transceiverCollector struct {
	reader     Reader
	decoder    decoder
	interfaces []string
	logger     *slog.Logger

	scrapeSuccess       *prometheus.Desc
	info                *prometheus.Desc
	temperature         *prometheus.Desc
	voltage             *prometheus.Desc
	txBias              *prometheus.Desc
	txPower             *prometheus.Desc
	txPowerDBm          *prometheus.Desc
	rxPower             *prometheus.Desc
	rxPowerDBm          *prometheus.Desc
	threshold           *prometheus.Desc
	alarm               *prometheus.Desc
	moduleStatus        *prometheus.Desc
	moduleLowPower      *prometheus.Desc
	laneDataPathState   *prometheus.Desc
	wavelength          *prometheus.Desc
	wavelengthTolerance *prometheus.Desc
	modulePowerClass    *prometheus.Desc
	modulePowerMaxWatts *prometheus.Desc
	bitRate             *prometheus.Desc
	length              *prometheus.Desc
}

// NewTransceiverCollector returns a Prometheus collector for decoded transceiver
// EEPROM telemetry.
//
// If interfaces is empty, the collector discovers physical non-loopback
// interfaces on each scrape and ignores devices that do not support module
// EEPROM access. If interfaces is non-empty, scrape errors are logged for those
// explicit interfaces.
func NewTransceiverCollector(reader Reader, interfaces []string, logger *slog.Logger) prometheus.Collector {
	return newTransceiverCollector(reader, sff.Decode, interfaces, logger)
}

func newTransceiverCollector(reader Reader, decoder decoder, interfaces []string, logger *slog.Logger) *transceiverCollector {
	labels := []string{"interface", "format"}
	infoLabels := append(
		labels[:len(labels):len(labels)],
		"form_factor",
		"connector",
		"vendor",
		"vendor_oui",
		"part_number",
		"revision",
		"serial_number",
		"date_code",
		"encoding",
		"rate_identifier",
		"cable_kind",
		"cable_standard",
		"media_type",
		"interface_technology",
	)
	laneLabels := append(labels[:len(labels):len(labels)], "lane")
	alarmLabels := append(labels[:len(labels):len(labels)], "alarm", "severity", "lane")
	thresholdLabels := append(labels[:len(labels):len(labels)], "metric", "boundary", "severity", "lane")
	statusLabels := append(labels[:len(labels):len(labels)], "state", "fault")
	lowPowerLabels := append(labels[:len(labels):len(labels)], "source")
	laneStateLabels := append(labels[:len(labels):len(labels)], "lane", "state")
	lengthLabels := append(labels[:len(labels):len(labels)], "medium")

	return &transceiverCollector{
		reader:     reader,
		decoder:    decoder,
		interfaces: interfaces,
		logger:     logger,
		scrapeSuccess: prometheus.NewDesc(
			"transceiver_scrape_success",
			"Whether the exporter successfully read and decoded the interface during this scrape.",
			[]string{"interface"},
			nil,
		),
		info: prometheus.NewDesc(
			"transceiver_module_info",
			"Transceiver module static information.",
			infoLabels,
			nil,
		),
		temperature: prometheus.NewDesc(
			"transceiver_temperature_celsius",
			"Transceiver module temperature in degrees Celsius.",
			labels,
			nil,
		),
		voltage: prometheus.NewDesc(
			"transceiver_voltage_volts",
			"Transceiver module supply voltage in volts.",
			labels,
			nil,
		),
		txBias: prometheus.NewDesc(
			"transceiver_tx_bias_milliamps",
			"Transceiver transmit bias current in milliamps.",
			laneLabels,
			nil,
		),
		txPower: prometheus.NewDesc(
			"transceiver_tx_power_milliwatts",
			"Transceiver transmit optical power in milliwatts.",
			laneLabels,
			nil,
		),
		txPowerDBm: prometheus.NewDesc(
			"transceiver_tx_power_dbm",
			"Transceiver transmit optical power in dBm.",
			laneLabels,
			nil,
		),
		rxPower: prometheus.NewDesc(
			"transceiver_rx_power_milliwatts",
			"Transceiver receive optical power in milliwatts.",
			laneLabels,
			nil,
		),
		rxPowerDBm: prometheus.NewDesc(
			"transceiver_rx_power_dbm",
			"Transceiver receive optical power in dBm.",
			laneLabels,
			nil,
		),
		threshold: prometheus.NewDesc(
			"transceiver_diagnostic_threshold",
			"Transceiver diagnostic threshold value.",
			thresholdLabels,
			nil,
		),
		alarm: prometheus.NewDesc(
			"transceiver_alarm_status",
			"Transceiver EEPROM alarm or warning threshold status, 1 for active and 0 for inactive.",
			alarmLabels,
			nil,
		),
		moduleStatus: prometheus.NewDesc(
			"transceiver_module_status",
			"Transceiver module state and fault status, 1 for the current state.",
			statusLabels,
			nil,
		),
		moduleLowPower: prometheus.NewDesc(
			"transceiver_module_low_power_status",
			"Transceiver module low-power request status, 1 for requested and 0 otherwise.",
			lowPowerLabels,
			nil,
		),
		laneDataPathState: prometheus.NewDesc(
			"transceiver_lane_datapath_state",
			"Transceiver lane data path state, 1 for the current state.",
			laneStateLabels,
			nil,
		),
		wavelength: prometheus.NewDesc(
			"transceiver_wavelength_nanometers",
			"Transceiver media wavelength in nanometers.",
			labels,
			nil,
		),
		wavelengthTolerance: prometheus.NewDesc(
			"transceiver_wavelength_tolerance_nanometers",
			"Transceiver media wavelength tolerance in nanometers.",
			labels,
			nil,
		),
		modulePowerClass: prometheus.NewDesc(
			"transceiver_module_power_class",
			"Transceiver advertised module power class.",
			labels,
			nil,
		),
		modulePowerMaxWatts: prometheus.NewDesc(
			"transceiver_module_power_max_watts",
			"Transceiver advertised maximum module power in watts.",
			labels,
			nil,
		),
		bitRate: prometheus.NewDesc(
			"transceiver_nominal_bitrate_mbd",
			"Transceiver nominal signaling rate in megabaud.",
			labels,
			nil,
		),
		length: prometheus.NewDesc(
			"transceiver_link_length_meters",
			"Transceiver supported link length in meters.",
			lengthLabels,
			nil,
		),
	}
}

func (c *transceiverCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.scrapeSuccess
	ch <- c.info
	ch <- c.temperature
	ch <- c.voltage
	ch <- c.txBias
	ch <- c.txPower
	ch <- c.txPowerDBm
	ch <- c.rxPower
	ch <- c.rxPowerDBm
	ch <- c.threshold
	ch <- c.alarm
	ch <- c.moduleStatus
	ch <- c.moduleLowPower
	ch <- c.laneDataPathState
	ch <- c.wavelength
	ch <- c.wavelengthTolerance
	ch <- c.modulePowerClass
	ch <- c.modulePowerMaxWatts
	ch <- c.bitRate
	ch <- c.length
}

func (c *transceiverCollector) Collect(ch chan<- prometheus.Metric) {
	interfaces := c.interfaces
	autoDiscovered := len(interfaces) == 0
	if len(interfaces) == 0 {
		discovered, err := netdev.CandidateNames()
		if err != nil {
			c.warn("discover interfaces failed", "err", err)
			return
		}
		interfaces = discovered
	}

	for _, interfaceName := range interfaces {
		data, err := c.reader.ModuleEEPROM(interfaceName)
		if err != nil {
			if autoDiscovered && netdev.IsUnsupportedModuleError(err) {
				continue
			}
			c.warn("read module eeprom failed", "interface", interfaceName, "err", err)
			c.emitScrapeSuccess(ch, interfaceName, false)
			continue
		}

		module, err := c.decoder(data)
		if err != nil {
			c.warn("decode module eeprom failed", "interface", interfaceName, "err", err)
			c.emitScrapeSuccess(ch, interfaceName, false)
			continue
		}
		observation := transceiver.Observation{Interface: interfaceName, Module: module}
		c.emitScrapeSuccess(ch, interfaceName, true)
		c.emitObservation(ch, observation)
	}
}

func (c *transceiverCollector) emitObservation(ch chan<- prometheus.Metric, observation transceiver.Observation) {
	labels := moduleLabels(observation)
	module := observation.Module
	diagnostics := module.Diagnostics

	ch <- prometheus.MustNewConstMetric(
		c.info,
		prometheus.GaugeValue,
		1,
		append(labels[:len(labels):len(labels)], moduleInfoLabels(module)...)...,
	)

	c.emitModuleInventory(ch, labels, module)

	if diagnostics.TemperatureCelsius.Valid {
		ch <- prometheus.MustNewConstMetric(c.temperature, prometheus.GaugeValue, diagnostics.TemperatureCelsius.Value, labels...)
	}
	if diagnostics.VoltageVolts.Valid {
		ch <- prometheus.MustNewConstMetric(c.voltage, prometheus.GaugeValue, diagnostics.VoltageVolts.Value, labels...)
	}

	for _, lane := range diagnostics.Lanes {
		laneLabels := append(labels[:len(labels):len(labels)], laneLabel(lane.Index))
		if lane.TXBiasMilliAmps.Valid {
			ch <- prometheus.MustNewConstMetric(c.txBias, prometheus.GaugeValue, lane.TXBiasMilliAmps.Value, laneLabels...)
		}
		if lane.TXPowerMilliWatts.Valid {
			ch <- prometheus.MustNewConstMetric(c.txPower, prometheus.GaugeValue, lane.TXPowerMilliWatts.Value, laneLabels...)
			if value, ok := milliwattsToDBm(lane.TXPowerMilliWatts.Value); ok {
				ch <- prometheus.MustNewConstMetric(c.txPowerDBm, prometheus.GaugeValue, value, laneLabels...)
			}
		}
		if lane.RXPowerMilliWatts.Valid {
			ch <- prometheus.MustNewConstMetric(c.rxPower, prometheus.GaugeValue, lane.RXPowerMilliWatts.Value, laneLabels...)
			if value, ok := milliwattsToDBm(lane.RXPowerMilliWatts.Value); ok {
				ch <- prometheus.MustNewConstMetric(c.rxPowerDBm, prometheus.GaugeValue, value, laneLabels...)
			}
		}
		if lane.DataPathState != "" {
			ch <- prometheus.MustNewConstMetric(
				c.laneDataPathState,
				prometheus.GaugeValue,
				1,
				append(laneLabels[:len(laneLabels):len(laneLabels)], lane.DataPathState)...,
			)
		}
	}

	for _, threshold := range diagnostics.Thresholds {
		thresholdLabels := append(
			labels[:len(labels):len(labels)],
			threshold.Metric,
			threshold.Boundary,
			threshold.Severity,
			laneLabel(threshold.Lane),
		)
		ch <- prometheus.MustNewConstMetric(c.threshold, prometheus.GaugeValue, threshold.Value, thresholdLabels...)
	}

	for _, alarm := range diagnostics.Alarms {
		value := 0.0
		if alarm.Active {
			value = 1
		}
		alarmLabels := append(labels[:len(labels):len(labels)], alarm.Name, alarm.Severity, laneLabel(alarm.Lane))
		ch <- prometheus.MustNewConstMetric(c.alarm, prometheus.GaugeValue, value, alarmLabels...)
	}
}

func (c *transceiverCollector) emitScrapeSuccess(ch chan<- prometheus.Metric, interfaceName string, success bool) {
	value := 0.0
	if success {
		value = 1
	}
	ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, value, interfaceName)
}

func (c *transceiverCollector) emitModuleInventory(ch chan<- prometheus.Metric, labels []string, module transceiver.Module) {
	if module.Status != nil {
		ch <- prometheus.MustNewConstMetric(
			c.moduleStatus,
			prometheus.GaugeValue,
			1,
			append(labels[:len(labels):len(labels)], module.Status.State, module.Status.Fault)...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.moduleLowPower,
			prometheus.GaugeValue,
			boolValue(module.Status.LowPowerAllowRequestHardware),
			append(labels[:len(labels):len(labels)], "hardware")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.moduleLowPower,
			prometheus.GaugeValue,
			boolValue(module.Status.LowPowerRequestSoftware),
			append(labels[:len(labels):len(labels)], "software")...,
		)
	}
	if module.Media != nil {
		if module.Media.WavelengthNanometers != 0 {
			ch <- prometheus.MustNewConstMetric(c.wavelength, prometheus.GaugeValue, module.Media.WavelengthNanometers, labels...)
		}
		if module.Media.WavelengthToleranceNanometers != 0 {
			ch <- prometheus.MustNewConstMetric(c.wavelengthTolerance, prometheus.GaugeValue, module.Media.WavelengthToleranceNanometers, labels...)
		}
	}
	if module.Power != nil {
		if module.Power.Class != 0 {
			ch <- prometheus.MustNewConstMetric(c.modulePowerClass, prometheus.GaugeValue, float64(module.Power.Class), labels...)
		}
		if module.Power.MaxWatts != 0 {
			ch <- prometheus.MustNewConstMetric(c.modulePowerMaxWatts, prometheus.GaugeValue, module.Power.MaxWatts, labels...)
		}
	}
	if module.BitRate.NominalMBd != 0 {
		ch <- prometheus.MustNewConstMetric(c.bitRate, prometheus.GaugeValue, float64(module.BitRate.NominalMBd), labels...)
	}
	for _, length := range module.Lengths {
		ch <- prometheus.MustNewConstMetric(
			c.length,
			prometheus.GaugeValue,
			float64(length.Meters),
			append(labels[:len(labels):len(labels)], string(length.Medium))...,
		)
	}
}

func (c *transceiverCollector) warn(msg string, args ...any) {
	if c.logger != nil {
		c.logger.Warn(msg, args...)
	}
}

func moduleLabels(observation transceiver.Observation) []string {
	module := observation.Module
	return []string{
		observation.Interface,
		string(module.MemoryMap),
	}
}

func moduleInfoLabels(module transceiver.Module) []string {
	labels := []string{
		module.FormFactor.String(),
		module.Connector.String(),
		module.Vendor.Name,
		module.Vendor.OUI,
		module.Vendor.PartNumber,
		module.Vendor.Revision,
		module.Vendor.SerialNumber,
		module.Vendor.DateCode,
		module.Encoding.String(),
		module.RateIdentifier.String(),
		string(module.CableCompliance.Kind),
		module.CableCompliance.Standard,
	}
	if module.Media != nil {
		labels = append(labels, module.Media.Type, module.Media.InterfaceTechnology)
	} else {
		labels = append(labels, "", "")
	}
	return labels
}

func laneLabel(index int) string {
	if index == 0 {
		return ""
	}
	return strconv.Itoa(index)
}

func boolValue(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func milliwattsToDBm(value float64) (float64, bool) {
	if value <= 0 {
		return 0, false
	}
	return 10 * math.Log10(value), true
}
