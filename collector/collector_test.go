package collector

import (
	"errors"
	"slices"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/vexxhost/transceiver_exporter/pkg/moduleeeprom"
	"github.com/vexxhost/transceiver_exporter/pkg/transceiver"
)

func TestCollectorEmitsModuleInfo(t *testing.T) {
	c := newTransceiverCollector(fakeReader{data: map[string][]byte{"eth0": {0x03}}}, fakeDecoder, []string{"eth0"}, nil)
	if got := testutil.CollectAndCount(c, "transceiver_module_info"); got != 1 {
		t.Fatalf("transceiver_module_info metric count = %d, want 1", got)
	}
}

func TestCollectorEmitsDiagnostics(t *testing.T) {
	c := newTransceiverCollector(fakeReader{data: map[string][]byte{"eth0": {0x03}}}, fakeDecoder, []string{"eth0"}, nil)

	for _, tt := range []struct {
		name       string
		metricName string
		expected   int
	}{
		{name: "module info", metricName: "transceiver_module_info", expected: 1},
		{name: "scrape success", metricName: "transceiver_scrape_success", expected: 1},
		{name: "temperature", metricName: "transceiver_temperature_celsius", expected: 1},
		{name: "voltage", metricName: "transceiver_voltage_volts", expected: 1},
		{name: "tx bias", metricName: "transceiver_tx_bias_milliamps", expected: 1},
		{name: "tx power", metricName: "transceiver_tx_power_milliwatts", expected: 1},
		{name: "tx power dbm", metricName: "transceiver_tx_power_dbm", expected: 1},
		{name: "rx power", metricName: "transceiver_rx_power_milliwatts", expected: 1},
		{name: "rx power dbm", metricName: "transceiver_rx_power_dbm", expected: 1},
		{name: "thresholds", metricName: "transceiver_diagnostic_threshold", expected: 1},
		{name: "alarms", metricName: "transceiver_alarm_status", expected: 2},
		{name: "module status", metricName: "transceiver_module_status", expected: 1},
		{name: "module low power", metricName: "transceiver_module_low_power_status", expected: 2},
		{name: "lane state", metricName: "transceiver_lane_datapath_state", expected: 1},
		{name: "wavelength", metricName: "transceiver_wavelength_nanometers", expected: 1},
		{name: "wavelength tolerance", metricName: "transceiver_wavelength_tolerance_nanometers", expected: 1},
		{name: "power class", metricName: "transceiver_module_power_class", expected: 1},
		{name: "max power", metricName: "transceiver_module_power_max_watts", expected: 1},
		{name: "bitrate", metricName: "transceiver_nominal_bitrate_mbd", expected: 1},
		{name: "length", metricName: "transceiver_link_length_meters", expected: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := testutil.CollectAndCount(c, tt.metricName); got != tt.expected {
				t.Fatalf("%s metric count = %d, want %d", tt.metricName, got, tt.expected)
			}
		})
	}
}

func TestCollectorKeepsModuleIdentityOnInfoMetric(t *testing.T) {
	c := newTransceiverCollector(fakeReader{data: map[string][]byte{"eth0": {0x03}}}, fakeDecoder, []string{"eth0"}, nil)

	dynamicLabels := labelNames(t, c, "transceiver_tx_power_milliwatts")
	for _, label := range []string{"vendor", "part_number", "serial_number"} {
		if slices.Contains(dynamicLabels, label) {
			t.Fatalf("dynamic metric has high-cardinality module label %q: %v", label, dynamicLabels)
		}
	}

	infoLabels := labelNames(t, c, "transceiver_module_info")
	for _, label := range []string{"vendor", "part_number", "serial_number"} {
		if !slices.Contains(infoLabels, label) {
			t.Fatalf("module info metric missing module label %q: %v", label, infoLabels)
		}
	}
}

func TestCollectorSkipsReaderAndDecoderErrors(t *testing.T) {
	tests := []struct {
		name    string
		reader  fakeReader
		decoder decoder
	}{
		{
			name:   "reader error",
			reader: fakeReader{errs: map[string]error{"eth0": errors.New("read failed")}},
			decoder: func(data []byte) (transceiver.Module, error) {
				t.Fatalf("decoder called for reader failure with %x", data)
				return transceiver.Module{}, nil
			},
		},
		{
			name:   "decoder error",
			reader: fakeReader{data: map[string][]byte{"eth0": {0x03}}},
			decoder: func([]byte) (transceiver.Module, error) {
				return transceiver.Module{}, errors.New("decode failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTransceiverCollector(tt.reader, tt.decoder, []string{"eth0"}, nil)
			if got := testutil.CollectAndCount(c, "transceiver_module_info"); got != 0 {
				t.Fatalf("transceiver_module_info metric count = %d, want 0", got)
			}
			if got := testutil.CollectAndCount(c, "transceiver_scrape_success"); got != 1 {
				t.Fatalf("transceiver_scrape_success metric count = %d, want 1", got)
			}
		})
	}
}

func TestCollectorAutoDiscoverySkipsAbsentModules(t *testing.T) {
	c := newTransceiverCollector(
		fakeReader{errs: map[string]error{"eth0": moduleeeprom.ErrModuleAbsent}},
		fakeDecoder,
		nil,
		nil,
	)
	c.discoverer = func() ([]string, error) { return []string{"eth0"}, nil }

	if got := testutil.CollectAndCount(c, "transceiver_scrape_success"); got != 0 {
		t.Fatalf("transceiver_scrape_success metric count = %d, want 0", got)
	}
	if got := testutil.CollectAndCount(c, "transceiver_module_info"); got != 0 {
		t.Fatalf("transceiver_module_info metric count = %d, want 0", got)
	}
}

func TestCollectorAutoDiscoveryReportsPresentModuleReadErrors(t *testing.T) {
	c := newTransceiverCollector(
		fakeReader{
			errs: map[string]error{"eth0": errors.New("read failed")},
		},
		fakeDecoder,
		nil,
		nil,
	)
	c.discoverer = func() ([]string, error) { return []string{"eth0"}, nil }

	if got := testutil.CollectAndCount(c, "transceiver_scrape_success"); got != 1 {
		t.Fatalf("transceiver_scrape_success metric count = %d, want 1", got)
	}
}

func TestCollectorExplicitInterfacesReportAbsentModules(t *testing.T) {
	c := newTransceiverCollector(
		fakeReader{errs: map[string]error{"eth0": moduleeeprom.ErrModuleAbsent}},
		fakeDecoder,
		[]string{"eth0"},
		nil,
	)

	if got := testutil.CollectAndCount(c, "transceiver_scrape_success"); got != 1 {
		t.Fatalf("transceiver_scrape_success metric count = %d, want 1", got)
	}
	if got := testutil.CollectAndCount(c, "transceiver_module_info"); got != 0 {
		t.Fatalf("transceiver_module_info metric count = %d, want 0", got)
	}
}

func labelNames(t *testing.T, c *transceiverCollector, metricName string) []string {
	t.Helper()

	registry := prometheus.NewRegistry()
	registry.MustRegister(c)
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("collect metric %s: %v", metricName, err)
	}
	for _, mf := range mfs {
		if mf.GetName() != metricName {
			continue
		}
		metrics := mf.GetMetric()
		if len(metrics) == 0 {
			t.Fatalf("metric %s has no samples", metricName)
		}

		names := make([]string, 0, len(metrics[0].GetLabel()))
		for _, label := range metrics[0].GetLabel() {
			names = append(names, label.GetName())
		}
		return names
	}
	t.Fatalf("metric %s not collected", metricName)
	return nil
}

type fakeReader struct {
	data map[string][]byte
	errs map[string]error
}

func (f fakeReader) ModuleEEPROM(interfaceName string) ([]byte, error) {
	if err := f.errs[interfaceName]; err != nil {
		return nil, err
	}
	return f.data[interfaceName], nil
}

func fakeDecoder(_ []byte) (transceiver.Module, error) {
	return transceiver.Module{
		MemoryMap:  transceiver.MemoryMapSFF8472,
		FormFactor: transceiver.FormFactorSFP,
		Connector:  transceiver.ConnectorCopperPigtail,
		Media: &transceiver.Media{
			Type:                          "optical",
			InterfaceTechnology:           "vcsel",
			WavelengthNanometers:          850,
			WavelengthToleranceNanometers: 10,
		},
		Power:   &transceiver.ModulePower{Class: 1, MaxWatts: 1.5},
		BitRate: transceiver.BitRate{NominalMBd: 25781},
		Lengths: []transceiver.Length{
			{Medium: transceiver.LengthMediumOM4, Meters: 100},
		},
		Status: &transceiver.ModuleStatus{
			State:                   "ready",
			LowPowerRequestSoftware: true,
		},
		Vendor: transceiver.Vendor{
			Name:         "fixture-vendor",
			OUI:          "00:11:22",
			PartNumber:   "fixture-part",
			Revision:     "A",
			SerialNumber: "fixture-serial",
			DateCode:     "260101",
		},
		Diagnostics: transceiver.Diagnostics{
			TemperatureCelsius: transceiver.NewReading(30.0),
			VoltageVolts:       transceiver.NewReading(3.3),
			Lanes: []transceiver.Lane{
				{
					Index:             1,
					DataPathState:     "activated",
					TXBiasMilliAmps:   transceiver.NewReading(7.5),
					TXPowerMilliWatts: transceiver.NewReading(1),
					RXPowerMilliWatts: transceiver.NewReading(0.13),
				},
			},
			Alarms: []transceiver.Alarm{
				{Name: "temperature_high", Severity: "warning", Active: true},
				{Name: "rx_power_low", Severity: "alarm", Lane: 1},
			},
			Thresholds: []transceiver.Threshold{
				{
					Metric:   "temperature_celsius",
					Boundary: "high",
					Severity: "warning",
					Value:    70,
				},
			},
		},
	}, nil
}
